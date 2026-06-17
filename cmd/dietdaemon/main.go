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
	"os"
	"os/signal"
	"syscall"

	"github.com/gsaraiva2109/dietdaemon/adapters/messaging/telegram"
	"github.com/gsaraiva2109/dietdaemon/adapters/notifier/ntfy"
	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/openfoodfacts"
	"github.com/gsaraiva2109/dietdaemon/adapters/nutrition/taco"
	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/config"
	"github.com/gsaraiva2109/dietdaemon/internal/parser/deterministic"
	"github.com/gsaraiva2109/dietdaemon/internal/pipeline"
	"github.com/gsaraiva2109/dietdaemon/internal/queue"
	"github.com/gsaraiva2109/dietdaemon/internal/resolver"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

// confidenceThreshold below which the pipeline nudges the user to double-check.
const confidenceThreshold = 0.6

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

	if cfg.ParserTier > types.TierDeterministic {
		slog.Warn("only the Tier-0 deterministic parser is implemented; falling back", "requested", cfg.ParserTier)
	}
	parser := deterministic.New()
	res := resolver.New(st, sources...)
	engine := pipeline.New(parser, res, st, msg, cfg.Location, confidenceThreshold)

	if cfg.EnableNotifications {
		if n, nerr := buildNotifier(cfg); nerr != nil {
			return nerr
		} else {
			// Held for the Phase-3 scheduler; logged so misconfig surfaces now.
			slog.Info("notifier ready", "notifier", n.Name())
		}
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

func buildMessaging(cfg *config.Config) (ports.MessagingAdapter, error) {
	switch cfg.MessagingAdapter {
	case "telegram":
		return telegram.New(cfg.TelegramBotToken), nil
	default:
		return nil, fmt.Errorf("unsupported MESSAGING_ADAPTER %q", cfg.MessagingAdapter)
	}
}

func buildNotifier(cfg *config.Config) (ports.Notifier, error) {
	switch cfg.Notifier {
	case "ntfy":
		return ntfy.New(cfg.NtfyURL, cfg.NtfyTopic, cfg.NtfyToken), nil
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
