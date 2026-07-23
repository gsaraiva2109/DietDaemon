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
	"sync"
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

var (
	loadConfig       = config.Load
	newSignalContext = signal.NotifyContext
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	return runWithConfig(cfg)
}

func runWithConfig(cfg *config.Config) error {
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
	go touchHealthy(cfg.HealthCheckPath)

	st, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	runtime, err := buildRuntime(cfg, st)
	if err != nil {
		return err
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := newSignalContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := ensureOllamaModels(ctx, cfg); err != nil {
		return err
	}

	backupRunner, err := startBackgroundServices(ctx, cfg, st, runtime)
	if err != nil {
		return err
	}
	if err := startDashboard(ctx, cfg, st, runtime, backupRunner); err != nil {
		return err
	}
	return runMessageLoop(ctx, cfg, runtime.message, runtime.engine)
}

type appRuntime struct {
	message  ports.MessagingAdapter
	engine   *pipeline.Engine
	notifier ports.Notifier
	embedder resolver.Embedder
	chat     ports.ChatAdapter
	vision   ports.VisionAdapter
	registry *commands.Registry
	i18n     *i18n.Bundle
	suggest  *suggest.Engine
}

func openStore(cfg *config.Config) (*store.Store, error) {
	dialect, err := store.NewDialect(cfg.DBDriver)
	if err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	dsn := cfg.DBPath
	if cfg.DBDriver == "postgres" {
		dsn = cfg.DatabaseURL
	}
	return store.New(cfg.DBDriver, dsn, dialect, cfg.Location)
}

func buildRuntime(cfg *config.Config, st *store.Store) (*appRuntime, error) {
	message, err := buildMessaging(cfg)
	if err != nil {
		return nil, err
	}
	sources, err := buildSources(cfg)
	if err != nil {
		return nil, err
	}
	completion, err := buildCompletionAdapter(cfg)
	if err != nil {
		return nil, err
	}
	slog.Info("completion adapter ready", "adapter", cfg.CompletionAdapter)
	chat, err := buildChatAdapter(cfg)
	if err != nil {
		return nil, err
	}
	slog.Info("chat adapter ready", "adapter", cfg.CompletionAdapter)
	vision, err := buildOCRAdapter(cfg)
	if err != nil {
		return nil, err
	}
	if vision != nil {
		slog.Info("OCR adapter ready", "adapter", cfg.OCRAdapter)
	}
	parser, matcher, embedder, err := buildParser(cfg, st, completion)
	if err != nil {
		return nil, err
	}
	res := resolver.New(st, matcher, embedder, cfg.AliasWriteBackThreshold, st, sources...)
	pend := pendingstore.New(st.DB(), cfg.PendingTTL)
	transcriber := newTranscriber(cfg)
	i18nBundle, err := loadI18nBundle()
	if err != nil {
		return nil, err
	}
	registry := commands.NewRegistry()
	if err := registerCoreCommands(registry, st, pend, cfg, i18nBundle); err != nil {
		return nil, err
	}
	suggestEngine := suggest.New(st, completion, cfg.Location)
	engine := pipeline.New(parser, res, st, pend, message, cfg.Location, cfg.ConfidenceThreshold, cfg.MessagingAdapter, transcriber, registry, i18nBundle)
	if err := registerPipelineCommands(registry, st, engine, res, suggestEngine); err != nil {
		return nil, err
	}
	notifier, err := newNotifier(cfg)
	if err != nil {
		return nil, err
	}
	return &appRuntime{message: message, engine: engine, notifier: notifier, embedder: embedder, chat: chat, vision: vision, registry: registry, i18n: i18nBundle, suggest: suggestEngine}, nil
}

func newTranscriber(cfg *config.Config) pipeline.Transcriber {
	if !cfg.EnableSTT {
		return nil
	}
	slog.Info("STT enabled", "whisper_url", cfg.WhisperURL)
	return whisper.New(cfg.WhisperURL)
}

func loadI18nBundle() (*i18n.Bundle, error) {
	bundle := i18n.NewBundle()
	if err := bundle.LoadEmbedded(locales.FS); err != nil {
		return nil, fmt.Errorf("i18n: load embedded locales: %w", err)
	}
	slog.Info("i18n loaded", "locales", "en,pt-BR")
	return bundle, nil
}

func registerCoreCommands(registry *commands.Registry, st *store.Store, pend *pendingstore.Store, cfg *config.Config, bundle *i18n.Bundle) error {
	commandsToRegister := []ports.Command{
		commands.NewTargetCommand(st), commands.NewCancelCommand(pend), commands.NewTimezoneCommand(st), commands.NewHelpCommand(registry, bundle), commands.NewStartCommand(st),
		commands.NewLinkCommand(st, st, cfg.MessagingAdapter), commands.NewStatusCommand(st, cfg.Location), commands.NewWeightCommand(st), commands.NewProfileCommand(st), commands.NewFoodCommand(st),
		commands.NewWaterCommand(st), commands.NewWorkoutCommand(st), commands.NewSleepCommand(st), commands.NewFastCommand(st), commands.NewNudgeCommand(st),
	}
	for _, command := range commandsToRegister {
		if err := registry.Register(command); err != nil {
			return fmt.Errorf("register %s command: %w", command.Name()[1:], err)
		}
	}
	return nil
}

func registerPipelineCommands(registry *commands.Registry, st *store.Store, engine *pipeline.Engine, res *resolver.Resolver, suggestEngine *suggest.Engine) error {
	commandsToRegister := []ports.Command{commands.NewSuggestCommand(suggestEngine, st), commands.NewTemplateCommand(st, engine, engine), commands.NewLogMealCommand(engine), commands.NewCorrectCommand(st, res)}
	for _, command := range commandsToRegister {
		if err := registry.Register(command); err != nil {
			return fmt.Errorf("register %s command: %w", command.Name()[1:], err)
		}
	}
	return nil
}

func newNotifier(cfg *config.Config) (ports.Notifier, error) {
	if !cfg.EnableNotifications {
		return nil, nil
	}
	notifier, err := buildNotifier(cfg)
	if err != nil {
		return nil, err
	}
	slog.Info("notifier ready", "notifier", notifier.Name())
	return notifier, nil
}

func startBackgroundServices(ctx context.Context, cfg *config.Config, st *store.Store, runtime *appRuntime) (*backup.Runner, error) {
	sched := scheduler.New(st, st, runtime.notifier, scheduler.DefaultRules(), cfg.Location, cfg.NudgeInterval,
		scheduler.WithHealthRules(st, scheduler.DefaultHealthRules()), scheduler.WithRuleConfig(st), scheduler.WithDigestRules(st, scheduler.DefaultDigestRules()),
		scheduler.WithChatSender(st, runtime.message), scheduler.WithSentNudges(st), scheduler.WithWeeklyBudgetRules(st, scheduler.DefaultWeeklyBudgetRules()), scheduler.WithSmartMealRules(st, scheduler.DefaultSmartMealRules()),
	)
	go sched.Run(ctx)
	slog.Info("scheduler running", "interval", cfg.NudgeInterval.String())

	backupRunner, err := newBackupRunner(ctx, cfg, st)
	if err != nil {
		return nil, err
	}
	go backupRunner.Run(ctx)
	slog.Info("backup runner running", "check_interval", cfg.BackupCheckInterval.String())

	go assistant.NewPurgeRunner(st, 24*time.Hour).Run(ctx)
	slog.Info("chat session purge runner running", "retention", "30d")
	if err := startFoodImportRunner(ctx, cfg, st, runtime.embedder); err != nil {
		return nil, err
	}
	return backupRunner, nil
}

func newBackupRunner(ctx context.Context, cfg *config.Config, st *store.Store) (*backup.Runner, error) {
	var localDst backup.Destination
	if cfg.BackupLocalDir != "" {
		ld, err := localdisk.New(cfg.BackupLocalDir)
		if err != nil {
			return nil, fmt.Errorf("backup: local destination: %w", err)
		}
		localDst = ld
	}
	var s3Dst backup.Destination
	if sd, err := s3dest.New(ctx); err != nil {
		slog.Warn("backup: s3 destination unavailable", "err", err)
	} else {
		s3Dst = sd
	}
	return backup.New(st, localDst, s3Dst, cfg.BackupCheckInterval), nil
}

func startFoodImportRunner(ctx context.Context, cfg *config.Config, st *store.Store, embedder resolver.Embedder) error {
	if !cfg.FoodImportEnabled || len(cfg.FoodImportSources) == 0 {
		return nil
	}
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
	runner := foodimport.NewWithLocalPaths(st, srcs, filters, cfg.FoodImportInterval, slog.Default(), localPaths, refresh)
	if embedder, ok := embedder.(foodimport.Embedder); ok {
		runner = runner.WithEmbedder(embedder)
	}
	go runner.Run(ctx)
	slog.Info("food import runner running", "sources", cfg.FoodImportSources, "interval", cfg.FoodImportInterval.String())
	return nil
}

func startDashboard(ctx context.Context, cfg *config.Config, st *store.Store, runtime *appRuntime, backupRunner *backup.Runner) error {
	if !cfg.EnableDashboard {
		return nil
	}
	authCfg := api.AuthConfig{
		SessionCfg: auth.SessionConfig{IdleTTL: cfg.SessionIdleTTL, AbsoluteTTL: cfg.SessionAbsoluteTTL, RememberTTL: cfg.SessionRememberTTL},
		LockoutCfg: auth.DefaultLockoutConfig(), RegistrationMode: types.RegistrationMode(cfg.RegistrationMode), CookieSecure: cfg.CookieSecure, CookieDomain: cfg.CookieDomain, MultiUser: cfg.MultiUser,
	}
	oidcRegistry := oidc.BuildRegistry(oidcProviderConfigs(cfg))
	m, err := mailer.New(mailer.Config{Provider: cfg.EmailProvider, From: cfg.EmailFrom, ResendAPIKey: cfg.ResendAPIKey, SESRegion: cfg.SESRegion, SMTPHost: cfg.SMTPHost, SMTPPort: cfg.SMTPPort, SMTPUsername: cfg.SMTPUsername, SMTPPassword: cfg.SMTPPassword, SMTPTLS: cfg.SMTPTLS, PublicBaseURL: cfg.PublicBaseURL})
	if err != nil {
		return fmt.Errorf("mailer: %w", err)
	}
	wa, err := auth.NewWebAuthn(cfg.WebAuthnConfig())
	if err != nil {
		return fmt.Errorf("webauthn: %w", err)
	}
	assistantRouter, toolDescs := newAssistantRouter(runtime.chat, runtime.registry, runtime.i18n)
	handler := api.New(st, runtime.engine, cfg.Location, runtime.suggest, cfg,
		api.WithAuth(st, st, st, st, st, st, cfg.TOTPEncKey, cfg.TOTPIssuer, authCfg), api.WithOIDC(oidcRegistry), api.WithMailer(m, cfg.EmailProvider), api.WithPublicBaseURL(cfg.PublicBaseURL),
		api.WithWebAuthn(wa), api.WithBackupRunner(backupRunner), api.WithFoodImportRunner(&foodImportAdmin{store: st, cfg: cfg}), api.WithChat(runtime.chat, assistantRouter, runtime.registry.List(), toolDescs, st), api.WithI18n(runtime.i18n), api.WithOCR(runtime.vision),
	)
	handler.StartRateLimiterCleanup(ctx)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	if spa, err := web.Handler(); err != nil {
		slog.Error("dashboard assets", "err", err)
	} else {
		mux.Handle("/", spa)
	}
	startHTTPServer(ctx, cfg, mux)
	return nil
}

func oidcProviderConfigs(cfg *config.Config) []oidc.ProviderConfig {
	configs := make([]oidc.ProviderConfig, len(cfg.OIDCProviders))
	for i, provider := range cfg.OIDCProviders {
		configs[i] = oidc.ProviderConfig{ID: provider.ID, Name: provider.Name, Issuer: provider.Issuer, ClientID: provider.ClientID, ClientSecret: provider.ClientSecret, RedirectURL: provider.RedirectURL, Scopes: provider.Scopes, TrustEmail: provider.TrustEmail}
	}
	return configs
}

func newAssistantRouter(chat ports.ChatAdapter, registry *commands.Registry, bundle *i18n.Bundle) (*assistant.Router, map[string]string) {
	if chat == nil {
		return nil, nil
	}
	cmds := registry.List()
	descriptions := make(map[string]string, len(cmds))
	for _, command := range cmds {
		description := bundle.T("en", command.Help(), nil)
		if description == "" {
			description = command.Name()
		}
		descriptions[command.Name()] = description
	}
	return assistant.New(chat, cmds, descriptions), descriptions
}

func startHTTPServer(ctx context.Context, cfg *config.Config, mux *http.ServeMux) {
	srv := newHTTPServer(":"+cfg.Port, newHTTPHandler(mux, cfg))
	go func() {
		slog.Info("dashboard listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("dashboard server", "err", err)
		}
	}()
	// #nosec G118 -- graceful server shutdown needs its own timeout after ctx is cancelled.
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
}

func runMessageLoop(ctx context.Context, cfg *config.Config, message ports.MessagingAdapter, engine *pipeline.Engine) error {
	q := queue.NewMemory[types.InboundMessage](64)
	defer func() { _ = q.Close() }()
	in, err := message.Receive(ctx)
	if err != nil {
		return fmt.Errorf("messaging receive: %w", err)
	}
	go publishMessages(ctx, q, in)
	slog.Info("listening for messages")
	var wg sync.WaitGroup
	wg.Add(cfg.MessageWorkers)
	for range cfg.MessageWorkers {
		go consumeMessages(ctx, &wg, q, engine)
	}
	wg.Wait()
	slog.Info("shutdown complete")
	return nil
}

func publishMessages(ctx context.Context, q *queue.Memory[types.InboundMessage], in <-chan types.InboundMessage) {
	defer func() { _ = q.Close() }()
	for message := range in {
		if err := q.Publish(ctx, message); err != nil {
			return
		}
	}
}

func consumeMessages(ctx context.Context, wg *sync.WaitGroup, q *queue.Memory[types.InboundMessage], engine *pipeline.Engine) {
	defer wg.Done()
	for message := range q.Consume() {
		if err := engine.Handle(ctx, message); err != nil {
			slog.Error("handle message", "user", message.UserID, "err", err)
		}
	}
}

func buildParser(cfg *config.Config, st *store.Store, completion ports.ModelAdapter) (ports.Parser, resolver.Matcher, resolver.Embedder, error) {
	if cfg.ParserTier >= types.TierLLM {
		model, err := buildEmbedAdapter(cfg)
		if err != nil {
			return nil, nil, nil, err
		}
		emb := embedding.New(model, index.New(st.DB()), st, cfg.EmbedMatchThreshold)
		slog.Info("parser tier 2 (LLM + embedding)", "embed_adapter", cfg.EmbedAdapter, "completion_adapter", cfg.CompletionAdapter)
		return llm.New(completion, deterministic.New()), emb, emb, nil
	}
	if cfg.ParserTier >= types.TierEmbedding {
		model, err := buildEmbedAdapter(cfg)
		if err != nil {
			return nil, nil, nil, err
		}
		emb := embedding.New(model, index.New(st.DB()), st, cfg.EmbedMatchThreshold)
		slog.Info("parser tier 1 (deterministic + embedding)", "embed_adapter", cfg.EmbedAdapter)
		return deterministic.New(), emb, emb, nil
	}
	slog.Info("parser tier 0 (deterministic, no model)")
	return deterministic.New(), nil, nil, nil
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
	if cfg.OCRAdapter == "ollama" {
		models = append(models, cfg.OllamaVisionModel)
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

// buildOCRAdapter creates the VisionAdapter for OCR nutrition-label capture
// (issue #87). Returns nil, nil when OCR_ADAPTER is unset — the feature is
// opt-in, so an unset adapter is not an error; the endpoint returns 501.
func buildOCRAdapter(cfg *config.Config) (ports.VisionAdapter, error) {
	switch cfg.OCRAdapter {
	case "":
		return nil, nil
	case "ollama":
		a := ollama.New(cfg.OllamaURL, cfg.EmbedModel, cfg.LLMModel, cfg.ModelTimeout)
		a.SetVisionModel(cfg.OllamaVisionModel)
		return a, nil
	case "anthropic":
		return anthropic.New(cfg.AnthropicAPIKey, cfg.AnthropicModel, cfg.ModelTimeout), nil
	case "openai":
		return openai.New(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.ModelTimeout), nil
	default:
		return nil, fmt.Errorf("unsupported OCR_ADAPTER %q", cfg.OCRAdapter)
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
func touchHealthy(path string) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for range t.C {
		writeHealthy(path)
	}
}

// writeHealthy writes the current UTC timestamp to path, the single write
// touchHealthy performs on every tick. Split out so it's testable without
// waiting on the 5s ticker.
func writeHealthy(path string) {
	_ = os.WriteFile(path, []byte(time.Now().UTC().Format(time.RFC3339)), 0600)
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
