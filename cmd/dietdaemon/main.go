// Command dietdaemon is the DietDaemon entrypoint. It loads configuration,
// selects adapters by config, wires the parse→resolve→persist→reply pipeline,
// and runs the ingest loop: messaging adapter → in-memory queue → pipeline.
//
// The whole graph is assembled here against the core interfaces; this is the
// only place that knows which concrete adapters are in use.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gsaraiva2109/dietdaemon/adapters/messaging/discord"
	"github.com/gsaraiva2109/dietdaemon/adapters/messaging/matrix"
	"github.com/gsaraiva2109/dietdaemon/adapters/messaging/telegram"
	"github.com/gsaraiva2109/dietdaemon/adapters/model/ollama"
	"github.com/gsaraiva2109/dietdaemon/adapters/notifier/gotify"
	"github.com/gsaraiva2109/dietdaemon/adapters/notifier/ntfy"
	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/openfoodfacts"
	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/taco"
	"github.com/gsaraiva2109/dietdaemon/adapters/stt/whisper"
	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/api"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/commands"
	"github.com/gsaraiva2109/dietdaemon/internal/config"
	"github.com/gsaraiva2109/dietdaemon/internal/i18n"
	"github.com/gsaraiva2109/dietdaemon/internal/i18n/locales"
	"github.com/gsaraiva2109/dietdaemon/internal/index"
	"github.com/gsaraiva2109/dietdaemon/internal/mailer"
	"github.com/gsaraiva2109/dietdaemon/internal/oidc"
	"github.com/gsaraiva2109/dietdaemon/internal/parser/deterministic"
	"github.com/gsaraiva2109/dietdaemon/internal/parser/llm"
	"github.com/gsaraiva2109/dietdaemon/internal/pendingstore"
	"github.com/gsaraiva2109/dietdaemon/internal/pipeline"
	"github.com/gsaraiva2109/dietdaemon/internal/queue"
	"github.com/gsaraiva2109/dietdaemon/internal/resolver"
	"github.com/gsaraiva2109/dietdaemon/internal/resolver/embedding"
	"github.com/gsaraiva2109/dietdaemon/internal/scheduler"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
	"github.com/gsaraiva2109/dietdaemon/internal/web"
)

const (
	// confidenceThreshold below which the pipeline nudges the user to double-check.
	confidenceThreshold = 0.6
	// nudgeInterval is how often the scheduler re-evaluates daily progress.
	nudgeInterval = 5 * time.Minute
	// pendingTTL is how long a meal awaiting clarification is held before the
	// state expires and the next message is treated as a fresh meal.
	pendingTTL = 30 * time.Minute
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	setupLogging(cfg.LogLevel)
	slog.Info("starting dietdaemon",
		"messaging", cfg.MessagingAdapter,
		"parser_tier", cfg.ParserTier,
		"sources", cfg.NutritionSources,
		"timezone", cfg.Location.String(),
	)

	st, err := store.New(cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	msg, err := buildMessaging(cfg)
	if err != nil {
		return err
	}
	sources, err := buildSources(cfg)
	if err != nil {
		return err
	}

	var (
		parser  ports.Parser
		matcher resolver.Matcher  = nil
		embed   resolver.Embedder = nil
	)

	switch {
	case cfg.ParserTier >= types.TierLLM:
		// Tier 2: LLM splitter + embedding matcher.
		model, idx := buildModelAndIndex(cfg, st)
		parser = llm.New(model, deterministic.New())
		emb := embedding.New(model, idx, st, cfg.EmbedMatchThreshold)
		matcher = emb
		embed = emb
		slog.Info("parser tier 2 (LLM + embedding)", "embed_model", cfg.EmbedModel, "llm_model", cfg.LLMModel)

	case cfg.ParserTier >= types.TierEmbedding:
		// Tier 1: deterministic splitter + embedding matcher.
		model, idx := buildModelAndIndex(cfg, st)
		parser = deterministic.New()
		emb := embedding.New(model, idx, st, cfg.EmbedMatchThreshold)
		matcher = emb
		embed = emb
		slog.Info("parser tier 1 (deterministic + embedding)", "embed_model", cfg.EmbedModel)

	default:
		// Tier 0: deterministic splitter, exact-alias match.
		parser = deterministic.New()
		slog.Info("parser tier 0 (deterministic, no model)")
	}

	res := resolver.New(st, matcher, embed, cfg.AliasWriteBackThreshold, sources...)
	pend := pendingstore.New(st.DB(), pendingTTL)

	var transcriber pipeline.Transcriber
	if cfg.EnableSTT {
		transcriber = whisper.New(cfg.WhisperURL)
		slog.Info("STT enabled", "whisper_url", cfg.WhisperURL)
	}

	// Set up i18n bundle.
	i18nBundle := i18n.NewBundle()
	if err := i18nBundle.LoadEmbedded(locales.FS); err != nil {
		return fmt.Errorf("i18n: load embedded locales: %w", err)
	}
	slog.Info("i18n loaded", "locales", "en,pt-BR")

	// Set up command registry with all bot commands.
	cmdRegistry := commands.NewRegistry()
	if err := cmdRegistry.Register(commands.NewTargetCommand(st)); err != nil {
		return fmt.Errorf("register target command: %w", err)
	}
	if err := cmdRegistry.Register(commands.NewCancelCommand(pend)); err != nil {
		return fmt.Errorf("register cancel command: %w", err)
	}
	if err := cmdRegistry.Register(commands.NewTimezoneCommand(st)); err != nil {
		return fmt.Errorf("register timezone command: %w", err)
	}
	if err := cmdRegistry.Register(commands.NewHelpCommand(cmdRegistry)); err != nil {
		return fmt.Errorf("register help command: %w", err)
	}
	if err := cmdRegistry.Register(commands.NewStartCommand(st)); err != nil {
		return fmt.Errorf("register start command: %w", err)
	}
	if err := cmdRegistry.Register(commands.NewLinkCommand(st, st, cfg.MessagingAdapter)); err != nil {
		return fmt.Errorf("register link command: %w", err)
	}
	if err := cmdRegistry.Register(commands.NewStatusCommand(st, cfg.Location)); err != nil {
		return fmt.Errorf("register status command: %w", err)
	}
	if err := cmdRegistry.Register(commands.NewWeightCommand(st)); err != nil {
		return fmt.Errorf("register weight command: %w", err)
	}
	if err := cmdRegistry.Register(commands.NewProfileCommand(st)); err != nil {
		return fmt.Errorf("register profile command: %w", err)
	}
	if err := cmdRegistry.Register(commands.NewFoodCommand(st)); err != nil {
		return fmt.Errorf("register food command: %w", err)
	}
	if err := cmdRegistry.Register(commands.NewWaterCommand(st)); err != nil {
		return fmt.Errorf("register water command: %w", err)
	}
	if err := cmdRegistry.Register(commands.NewWorkoutCommand(st)); err != nil {
		return fmt.Errorf("register workout command: %w", err)
	}
	if err := cmdRegistry.Register(commands.NewSleepCommand(st)); err != nil {
		return fmt.Errorf("register sleep command: %w", err)
	}
	if err := cmdRegistry.Register(commands.NewFastCommand(st)); err != nil {
		return fmt.Errorf("register fast command: %w", err)
	}

	engine := pipeline.New(parser, res, st, pend, msg, cfg.Location, confidenceThreshold, cfg.MessagingAdapter, transcriber, cmdRegistry, i18nBundle)

	if err := cmdRegistry.Register(commands.NewTemplateCommand(st, engine)); err != nil {
		return fmt.Errorf("register template command: %w", err)
	}

	var notifier ports.Notifier
	if cfg.EnableNotifications {
		notifier, err = buildNotifier(cfg)
		if err != nil {
			return err
		}
		slog.Info("notifier ready", "notifier", notifier.Name())
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Nudge scheduler: only meaningful when a notifier is configured.
	if notifier != nil {
		sched := scheduler.New(st, st, notifier, scheduler.DefaultRules(), cfg.Location, nudgeInterval)
		go sched.Run(ctx)
		slog.Info("scheduler running", "interval", nudgeInterval.String())
	}

	// --- Dashboard API server ---
	if cfg.EnableDashboard {
		authCfg := api.AuthConfig{
			SessionCfg: auth.SessionConfig{
				IdleTTL:     cfg.SessionIdleTTL,
				AbsoluteTTL: cfg.SessionAbsoluteTTL,
				RememberTTL: cfg.SessionRememberTTL,
			},
			LockoutCfg:       auth.DefaultLockoutConfig(),
			RegistrationMode: types.RegistrationMode(cfg.RegistrationMode),
			CookieSecure:     cfg.CookieSecure,
		}
		oidcConfigs := make([]oidc.ProviderConfig, len(cfg.OIDCProviders))
		for i, c := range cfg.OIDCProviders {
			oidcConfigs[i] = oidc.ProviderConfig{
				ID: c.ID, Name: c.Name, Issuer: c.Issuer,
				ClientID: c.ClientID, ClientSecret: c.ClientSecret,
				RedirectURL: c.RedirectURL, Scopes: c.Scopes,
				TrustEmail: c.TrustEmail,
			}
		}
		oidcRegistry := oidc.BuildRegistry(oidcConfigs)

		mailCfg := mailer.Config{
			Provider:      cfg.EmailProvider,
			From:          cfg.EmailFrom,
			ResendAPIKey:  cfg.ResendAPIKey,
			SESRegion:     cfg.SESRegion,
			SMTPHost:      cfg.SMTPHost,
			SMTPPort:      cfg.SMTPPort,
			SMTPUsername:  cfg.SMTPUsername,
			SMTPPassword:  cfg.SMTPPassword,
			SMTPTLS:       cfg.SMTPTLS,
			PublicBaseURL: cfg.PublicBaseURL,
		}
		m, err := mailer.New(mailCfg)
		if err != nil {
			return fmt.Errorf("mailer: %w", err)
		}

		wa, waErr := auth.NewWebAuthn(cfg.WebAuthnConfig())
		if waErr != nil {
			return fmt.Errorf("webauthn: %w", waErr)
		}

		apiHandler := api.New(st, st, engine, cfg.Location, st, st, st, st, st, cfg.TOTPEncKey, cfg.TOTPIssuer, oidcRegistry, m, cfg.EmailProvider, cfg.PublicBaseURL, authCfg, wa)
		mux := http.NewServeMux()
		apiHandler.RegisterRoutes(mux)

		// Serve the embedded dashboard SPA on all non-API routes. ServeMux
		// matches the more specific /api/v1/* patterns first, so this only
		// catches asset and client-route requests.
		if spa, spaErr := web.Handler(); spaErr != nil {
			slog.Error("dashboard assets", "err", spaErr)
		} else {
			mux.Handle("/", spa)
		}

		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		srv := &http.Server{
			Addr:              ":" + port,
			Handler:           mux,
			ReadHeaderTimeout: 3 * time.Second,
		}

		go func() {
			slog.Info("dashboard listening", "port", port)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("dashboard server", "err", err)
			}
		}()

		// Shutdown on context cancellation.
		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
		}()
	}

	q := queue.NewMemory[types.InboundMessage](64)

	// Producer: messaging adapter → queue.
	in, err := msg.Receive(ctx)
	if err != nil {
		return fmt.Errorf("messaging receive: %w", err)
	}
	go func() {
		defer func() { _ = q.Close() }()
		for m := range in {
			if perr := q.Publish(ctx, m); perr != nil {
				return // queue closed or context cancelled
			}
		}
	}()

	// Consumer: queue → pipeline. Runs until the queue drains after shutdown.
	slog.Info("listening for messages")
	for m := range q.Consume() {
		if herr := engine.Handle(ctx, m); herr != nil {
			slog.Error("handle message", "user", m.UserID, "err", herr)
		}
	}
	slog.Info("shutdown complete")
	return nil
}

// buildModelAndIndex creates the Ollama adapter and the embedding index. It is
// only called when PARSER_TIER >= 1.
func buildModelAndIndex(cfg *config.Config, st *store.Store) (ports.ModelAdapter, *index.SQLIndex) {
	model := ollama.New(cfg.OllamaURL, cfg.EmbedModel, cfg.LLMModel, cfg.ModelTimeout)
	idx := index.New(st.DB())
	return model, idx
}

func buildMessaging(cfg *config.Config) (ports.MessagingAdapter, error) {
	switch cfg.MessagingAdapter {
	case "telegram":
		return telegram.New(cfg.TelegramBotToken), nil
	case "discord":
		return discord.New(cfg.DiscordBotToken), nil
	case "matrix":
		return matrix.New(cfg.MatrixHomeserverURL, cfg.MatrixUserID, cfg.MatrixToken), nil
	default:
		return nil, fmt.Errorf("unsupported MESSAGING_ADAPTER %q", cfg.MessagingAdapter)
	}
}

func buildNotifier(cfg *config.Config) (ports.Notifier, error) {
	switch cfg.Notifier {
	case "ntfy":
		return ntfy.New(cfg.NtfyURL, cfg.NtfyTopic, cfg.NtfyToken), nil
	case "gotify":
		return gotify.New(cfg.GotifyURL, cfg.GotifyToken), nil
	default:
		return nil, fmt.Errorf("unsupported NOTIFIER %q", cfg.Notifier)
	}
}

func buildSources(cfg *config.Config) ([]resolver.Source, error) {
	var sources []resolver.Source
	for _, name := range cfg.NutritionSources {
		switch name {
		case "openfoodfacts":
			sources = append(sources, openfoodfacts.New())
		case "taco":
			src, err := taco.New(cfg.TacoDataPath)
			if err != nil {
				return nil, fmt.Errorf("taco source: %w", err)
			}
			sources = append(sources, src)
		default:
			return nil, fmt.Errorf("unsupported NUTRITION_SOURCE %q", name)
		}
	}
	return sources, nil
}

func setupLogging(level string) {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: l})))
}
