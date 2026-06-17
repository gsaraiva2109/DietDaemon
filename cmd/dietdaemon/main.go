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
	"github.com/gsaraiva2109/dietdaemon/internal/config"
	"github.com/gsaraiva2109/dietdaemon/internal/index"
	"github.com/gsaraiva2109/dietdaemon/internal/parser/deterministic"
	"github.com/gsaraiva2109/dietdaemon/internal/parser/llm"
	"github.com/gsaraiva2109/dietdaemon/internal/pendingstore"
	"github.com/gsaraiva2109/dietdaemon/internal/pipeline"
	"github.com/gsaraiva2109/dietdaemon/internal/queue"
	"github.com/gsaraiva2109/dietdaemon/internal/resolver"
	"github.com/gsaraiva2109/dietdaemon/internal/resolver/embedding"
	"github.com/gsaraiva2109/dietdaemon/internal/scheduler"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
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
	defer st.Close()

	msg, err := buildMessaging(cfg)
	if err != nil {
		return err
	}
	sources, err := buildSources(cfg)
	if err != nil {
		return err
	}

	// --- Phase 5 wiring ---

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
		// Tier 0: deterministic splitter, exact-alias match (Phase 4 behaviour).
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

	engine := pipeline.New(parser, res, st, pend, msg, cfg.Location, confidenceThreshold, cfg.MessagingAdapter, transcriber)

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
		apiHandler := api.New(st, engine, cfg.Location, cfg.APIAuthToken, cfg.MultiUser)
		mux := http.NewServeMux()
		apiHandler.RegisterRoutes(mux)

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
		defer q.Close()
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
