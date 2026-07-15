// Command dietdaemon is the DietDaemon entrypoint. It loads configuration,
// selects adapters by config, wires the parse→resolve→persist→reply pipeline,
// and runs the ingest loop: messaging adapter → in-memory queue → pipeline.
//
// The whole graph is assembled here against the core interfaces; this is the
// only place that knows which concrete adapters are in use.
package main

import (
	"context"
	"errors"
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
	"github.com/gsaraiva2109/dietdaemon/adapters/model/anthropic"
	"github.com/gsaraiva2109/dietdaemon/adapters/model/ollama"
	"github.com/gsaraiva2109/dietdaemon/adapters/model/openai"
	"github.com/gsaraiva2109/dietdaemon/adapters/notifier/gotify"
	"github.com/gsaraiva2109/dietdaemon/adapters/notifier/ntfy"
	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/openfoodfacts"
	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/taco"
	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/usda"
	"github.com/gsaraiva2109/dietdaemon/adapters/stt/whisper"
	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/api"
	"github.com/gsaraiva2109/dietdaemon/internal/assistant"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/backup"
	"github.com/gsaraiva2109/dietdaemon/internal/backup/localdisk"
	"github.com/gsaraiva2109/dietdaemon/internal/backup/s3dest"
	"github.com/gsaraiva2109/dietdaemon/internal/commands"
	"github.com/gsaraiva2109/dietdaemon/internal/config"
	"github.com/gsaraiva2109/dietdaemon/internal/foodimport"
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
	"github.com/gsaraiva2109/dietdaemon/internal/suggest"
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

	// File-based liveness probe for distroless HEALTHCHECK. The healthcheck
	// binary checks /tmp/healthy age — works even without the dashboard HTTP
	// server (bot-only deployments). Goroutine dies with the process; no
	// explicit cancellation needed.
	go touchHealthy()

	dialect, err := store.NewDialect(cfg.DBDriver)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	dsn := cfg.DBPath
	if cfg.DBDriver == "postgres" {
		dsn = cfg.DatabaseURL
	}
	st, err := store.New(cfg.DBDriver, dsn, dialect)
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

	// Completion adapter is built unconditionally (not just for PARSER_TIER>0):
	// Tier-2 parsing uses it for text splitting, and /suggest uses it for
	// meal ranking regardless of parser tier. Construction never dials out, so
	// this stays safe with COMPLETION_ADAPTER left at its "ollama" default and
	// no OLLAMA_URL set (zero-keys boot).
	completionModel, err := buildCompletionAdapter(cfg)
	if err != nil {
		return err
	}
	slog.Info("completion adapter ready", "adapter", cfg.CompletionAdapter)

	chatModel, err := buildChatAdapter(cfg)
	if err != nil {
		return err
	}
	slog.Info("chat adapter ready", "adapter", cfg.CompletionAdapter)

	var (
		parser  ports.Parser
		matcher resolver.Matcher  = nil
		embed   resolver.Embedder = nil
	)

	switch {
	case cfg.ParserTier >= types.TierLLM:
		// Tier 2: LLM splitter (completion adapter) + embedding matcher (embed adapter).
		embedModel, err := buildEmbedAdapter(cfg)
		if err != nil {
			return err
		}
		idx := index.New(st.DB())
		parser = llm.New(completionModel, deterministic.New())
		emb := embedding.New(embedModel, idx, st, cfg.EmbedMatchThreshold)
		matcher = emb
		embed = emb
		slog.Info("parser tier 2 (LLM + embedding)", "embed_adapter", cfg.EmbedAdapter, "completion_adapter", cfg.CompletionAdapter)

	case cfg.ParserTier >= types.TierEmbedding:
		// Tier 1: deterministic splitter + embedding matcher (embed adapter).
		embedModel, err := buildEmbedAdapter(cfg)
		if err != nil {
			return err
		}
		idx := index.New(st.DB())
		parser = deterministic.New()
		emb := embedding.New(embedModel, idx, st, cfg.EmbedMatchThreshold)
		matcher = emb
		embed = emb
		slog.Info("parser tier 1 (deterministic + embedding)", "embed_adapter", cfg.EmbedAdapter)

	default:
		// Tier 0: deterministic splitter, exact-alias match.
		parser = deterministic.New()
		slog.Info("parser tier 0 (deterministic, no model)")
	}

	res := resolver.New(st, matcher, embed, cfg.AliasWriteBackThreshold, st, sources...)
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
	if err := cmdRegistry.Register(commands.NewHelpCommand(cmdRegistry, i18nBundle)); err != nil {
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
	if err := cmdRegistry.Register(commands.NewNudgeCommand(st)); err != nil {
		return fmt.Errorf("register nudge command: %w", err)
	}

	suggestEngine := suggest.New(st, completionModel, cfg.Location)
	if err := cmdRegistry.Register(commands.NewSuggestCommand(suggestEngine, st)); err != nil {
		return fmt.Errorf("register suggest command: %w", err)
	}

	engine := pipeline.New(parser, res, st, pend, msg, cfg.Location, confidenceThreshold, cfg.MessagingAdapter, transcriber, cmdRegistry, i18nBundle)

	if err := cmdRegistry.Register(commands.NewTemplateCommand(st, engine, engine)); err != nil {
		return fmt.Errorf("register template command: %w", err)
	}

	if err := cmdRegistry.Register(commands.NewLogMealCommand(engine)); err != nil {
		return fmt.Errorf("register logmeal command: %w", err)
	}

	if err := cmdRegistry.Register(commands.NewCorrectCommand(st, res)); err != nil {
		return fmt.Errorf("register correct command: %w", err)
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
	if err := ensureOllamaModels(ctx, cfg); err != nil {
		return err
	}

	// Nudge scheduler: delivers through chat (Telegram/Discord/Matrix) whenever
	// a route is known for the user, falling back to the notifier (ntfy/gotify)
	// when it isn't or when EnableNotifications is false, notifier is nil and
	// deliver() surfaces the "no delivery channel" error instead of nudging.
	sched := scheduler.New(st, st, notifier, scheduler.DefaultRules(), cfg.Location, nudgeInterval,
		scheduler.WithHealthRules(st, scheduler.DefaultHealthRules()),
		scheduler.WithRuleConfig(st),
		scheduler.WithDigestRules(st, scheduler.DefaultDigestRules()),
		scheduler.WithChatSender(st, msg),
		scheduler.WithSentNudges(st),
		scheduler.WithWeeklyBudgetRules(st, scheduler.DefaultWeeklyBudgetRules()),
		scheduler.WithSmartMealRules(st, scheduler.DefaultSmartMealRules()),
	)
	go sched.Run(ctx)
	slog.Info("scheduler running", "interval", nudgeInterval.String())

	// Scheduled backup/export: an independent background loop (separate from
	// the nudge scheduler above). The "local" destination only exists when an
	// operator sets BACKUP_LOCAL_DIR; "s3" uses the ambient AWS credential
	// chain and per-user bucket/prefix/region/endpoint, so it's always wired
	// up (whether any user actually picks it is a per-user setting).
	var localDst backup.Destination
	if cfg.BackupLocalDir != "" {
		ld, err := localdisk.New(cfg.BackupLocalDir)
		if err != nil {
			return fmt.Errorf("backup: local destination: %w", err)
		}
		localDst = ld
	}
	var s3Dst backup.Destination
	if sd, err := s3dest.New(ctx); err != nil {
		slog.Warn("backup: s3 destination unavailable", "err", err)
	} else {
		s3Dst = sd
	}
	backupRunner := backup.New(st, localDst, s3Dst, cfg.BackupCheckInterval)
	go backupRunner.Run(ctx)
	slog.Info("backup runner running", "check_interval", cfg.BackupCheckInterval.String())

	go assistant.NewPurgeRunner(st, 24*time.Hour).Run(ctx)
	slog.Info("chat session purge runner running", "retention", "30d")

	// Scheduled bulk food import: opt-in, disabled by default so there's no
	// surprise startup traffic against USDA/OpenFoodFacts.
	if cfg.FoodImportEnabled && len(cfg.FoodImportSources) > 0 {
		var srcs []ports.BulkSource
		filters := map[string]ports.BulkFilter{}
		localPaths := foodimport.LocalPaths(cfg)
		refresh := map[string]foodimport.SourceFactory{}
		for _, name := range cfg.FoodImportSources {
			src, filter, err := foodimport.BuildSource(name, cfg)
			if err != nil {
				return fmt.Errorf("food import: %w", err)
			}
			srcs = append(srcs, src)
			filters[src.Name()] = filter
			if localPaths[src.Name()] != "" {
				sourceName := name
				refresh[src.Name()] = func() (ports.BulkSource, error) {
					source, _, err := foodimport.BuildSource(sourceName, cfg)
					return source, err
				}
			}
		}
		importRunner := foodimport.NewWithLocalPaths(st, srcs, filters, cfg.FoodImportInterval, slog.Default(), localPaths, refresh)
		if embedder, ok := embed.(foodimport.Embedder); ok {
			importRunner = importRunner.WithEmbedder(embedder)
		}
		go importRunner.Run(ctx)
		slog.Info("food import runner running", "sources", cfg.FoodImportSources, "interval", cfg.FoodImportInterval.String())
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

		// Build assistant router (tool-calling loop) for the chat endpoint.
		// nil when chatModel is nil (unsupported adapter).
		var assistantRouter *assistant.Router
		var toolDescs map[string]string
		if chatModel != nil {
			cmds := cmdRegistry.List()
			toolDescs = make(map[string]string, len(cmds))
			for _, c := range cmds {
				desc := i18nBundle.T("en", c.Help(), nil)
				if desc == "" {
					desc = c.Name()
				}
				toolDescs[c.Name()] = desc
			}
			assistantRouter = assistant.New(chatModel, cmds, toolDescs)
		}

		apiHandler := api.New(st, st, engine, cfg.Location, st, st, st, st, st, cfg.TOTPEncKey, cfg.TOTPIssuer, oidcRegistry, m, cfg.EmailProvider, cfg.PublicBaseURL, authCfg, wa, backupRunner, suggestEngine, cfg, chatModel, assistantRouter, cmdRegistry.List(), toolDescs, st, i18nBundle)
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
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	defer func() { _ = q.Close() }()

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

// buildEmbedAdapter creates the adapter used for food-matching embeddings
// (Tier-1/2 parsing, /suggest candidate pool). Only ollama offers embeddings.
func buildEmbedAdapter(cfg *config.Config) (ports.ModelAdapter, error) {
	switch cfg.EmbedAdapter {
	case "ollama":
		return ollama.New(cfg.OllamaURL, cfg.EmbedModel, cfg.LLMModel, cfg.ModelTimeout), nil
	default:
		return nil, fmt.Errorf("unsupported EMBED_ADAPTER %q", cfg.EmbedAdapter)
	}
}

// ensureOllamaModels provisions only the models enabled by the current feature
// set. It is opt-in because models can be several gigabytes.
func ensureOllamaModels(ctx context.Context, cfg *config.Config) error {
	if !cfg.OllamaAutoPull {
		return nil
	}
	models := requiredOllamaModels(cfg)
	if len(models) == 0 {
		return nil
	}
	if err := ollama.New(cfg.OllamaURL, "", "", cfg.ModelTimeout).EnsureModels(ctx, models...); err != nil {
		return fmt.Errorf("ensure Ollama models: %w", err)
	}
	slog.Info("Ollama models ready", "models", models)
	return nil
}

func requiredOllamaModels(cfg *config.Config) []string {
	var models []string
	if cfg.ParserTier >= types.TierEmbedding {
		models = append(models, cfg.EmbedModel)
	}
	if cfg.CompletionAdapter == "ollama" && (cfg.ParserTier >= types.TierLLM || cfg.EnableDashboard) {
		models = append(models, cfg.LLMModel)
	}
	return models
}

// buildCompletionAdapter creates the adapter used for text completion
// (Tier-2 parsing, /suggest ranking). Opt-in cloud providers require their
// API key, validated in config.validate.
func buildCompletionAdapter(cfg *config.Config) (ports.ModelAdapter, error) {
	switch cfg.CompletionAdapter {
	case "ollama":
		return ollama.New(cfg.OllamaURL, cfg.EmbedModel, cfg.LLMModel, cfg.ModelTimeout), nil
	case "anthropic":
		return anthropic.New(cfg.AnthropicAPIKey, cfg.AnthropicModel, cfg.ModelTimeout), nil
	case "openai":
		return openai.New(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.ModelTimeout), nil
	default:
		return nil, fmt.Errorf("unsupported COMPLETION_ADAPTER %q", cfg.CompletionAdapter)
	}
}

// buildChatAdapter creates the ChatAdapter for the conversational assistant.
// All three providers are supported; returns nil when chat is unavailable
// (e.g. missing API keys) — the chat endpoint then returns 503.
func buildChatAdapter(cfg *config.Config) (ports.ChatAdapter, error) {
	switch cfg.CompletionAdapter {
	case "anthropic":
		return anthropic.NewChatAdapter(cfg.AnthropicAPIKey, cfg.AnthropicModel, cfg.ModelTimeout), nil
	case "openai":
		return openai.NewChatAdapter(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.ModelTimeout), nil
	case "ollama":
		return ollama.NewChatAdapter(cfg.OllamaURL, cfg.LLMModel, cfg.ModelTimeout), nil
	default:
		slog.Warn("chat adapter not available for configured COMPLETION_ADAPTER, chat endpoint will return 503", "adapter", cfg.CompletionAdapter)
		return nil, nil
	}
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
		case "usda":
			sources = append(sources, usda.New(cfg.USDAFDCAPIKey))
		default:
			return nil, fmt.Errorf("unsupported NUTRITION_SOURCE %q", name)
		}
	}
	return sources, nil
}

// touchHealthy writes a timestamp file every 5 seconds so the distroless
// HEALTHCHECK probe (/bin/healthcheck) can verify the process is alive
// without depending on the dashboard HTTP server.
func touchHealthy() {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for range t.C {
		_ = os.WriteFile("/data/healthy", []byte(time.Now().UTC().Format(time.RFC3339)), 0600)
	}
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
