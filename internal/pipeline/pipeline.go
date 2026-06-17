// Package pipeline wires the per-message flow of DietDaemon: an inbound message
// is parsed (Stage A), resolved to macros (Stage B), persisted as a Meal, folded
// into the day's rollup, and acknowledged with an in-channel reply. It depends
// only on narrow interfaces, so the concrete parser, resolver, store, and
// messaging adapter all plug in without the pipeline importing them.
package pipeline

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Parser is Stage A. Satisfied by internal/parser/deterministic.Parser.
type Parser interface {
	Extract(ctx context.Context, text, locale string) ([]types.ParsedItem, float64, error)
	Tier() types.ParserTier
}

// Resolver is Stage B. Satisfied by internal/resolver.Resolver.
type Resolver interface {
	Resolve(ctx context.Context, userID string, items []types.ParsedItem) ([]types.ResolvedItem, int)
}

// MealStore is the subset of ports.Store the pipeline needs.
type MealStore interface {
	UpsertUser(ctx context.Context, u types.User) error
	SaveMeal(ctx context.Context, m types.Meal) error
	GetTargets(ctx context.Context, userID string) (types.DailyTargets, error)
	GetRollup(ctx context.Context, userID, localDate string) (types.DailyRollup, error)
	UpsertRollup(ctx context.Context, r types.DailyRollup) error
}

// Replier sends an in-channel reply. Satisfied by any ports.MessagingAdapter.
type Replier interface {
	Send(ctx context.Context, reply types.Reply) error
}

// Engine processes one message at a time. It is safe for sequential use by a
// single consumer goroutine draining the queue.
type Engine struct {
	parser    Parser
	resolver  Resolver
	store     MealStore
	replier   Replier
	loc       *time.Location
	threshold float64 // replies flag clarification when confidence < threshold

	now   func() time.Time
	idgen func() string
}

// New builds an Engine. loc is the default timezone used for daily rollup
// boundaries; threshold is the confidence below which a reply nudges the user
// to clarify.
func New(p Parser, r Resolver, s MealStore, replier Replier, loc *time.Location, threshold float64) *Engine {
	if loc == nil {
		loc = time.UTC
	}
	return &Engine{
		parser:    p,
		resolver:  r,
		store:     s,
		replier:   replier,
		loc:       loc,
		threshold: threshold,
		now:       time.Now,
		idgen:     randomID,
	}
}

// Handle runs the full pipeline for one inbound message. Parsing/resolution
// problems are reported back to the user rather than returned as errors;
// non-nil errors indicate infrastructure failures (store, transport).
func (e *Engine) Handle(ctx context.Context, msg types.InboundMessage) error {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return e.reply(ctx, msg, "Send a meal as text, e.g. \"200g chicken, 2 eggs\".")
	}

	at := msg.At
	if at.IsZero() {
		at = e.now()
	}
	at = at.UTC()

	// Ensure the user row exists (single-user today; keyed by user from day one).
	if err := e.store.UpsertUser(ctx, types.User{ID: msg.UserID, Timezone: e.loc.String(), CreatedAt: e.now().UTC()}); err != nil {
		return fmt.Errorf("pipeline: upsert user: %w", err)
	}

	// Stage A.
	items, confidence, err := e.parser.Extract(ctx, text, msg.Locale)
	if err != nil {
		return fmt.Errorf("pipeline: parse: %w", err)
	}
	if len(items) == 0 {
		return e.reply(ctx, msg, "Couldn't read any food in that. Try \"200g rice, 100g beans\".")
	}

	// Stage B.
	resolved, needsClarification := e.resolver.Resolve(ctx, msg.UserID, items)

	// Persist the meal.
	meal := types.Meal{
		ID:         e.idgen(),
		UserID:     msg.UserID,
		At:         at,
		RawText:    text,
		Items:      resolved,
		Confidence: confidence,
		ParserTier: e.parser.Tier(),
		CreatedAt:  e.now().UTC(),
	}
	if err := e.store.SaveMeal(ctx, meal); err != nil {
		return fmt.Errorf("pipeline: save meal: %w", err)
	}

	// Fold into the day's rollup, in the user's local calendar day.
	if err := e.updateRollup(ctx, msg.UserID, at, meal.Total()); err != nil {
		return fmt.Errorf("pipeline: update rollup: %w", err)
	}

	return e.reply(ctx, msg, e.summary(meal, needsClarification, confidence))
}

// updateRollup adds the meal's macros to the user's rollup for its local day,
// creating the row (with current targets) on first meal of the day.
func (e *Engine) updateRollup(ctx context.Context, userID string, at time.Time, add types.Macros) error {
	localDate := at.In(e.loc).Format("2006-01-02")

	rollup, err := e.store.GetRollup(ctx, userID, localDate)
	if err != nil {
		if !isNotFound(err) {
			return err
		}
		rollup = types.DailyRollup{UserID: userID, Date: localDate}
		if t, terr := e.store.GetTargets(ctx, userID); terr == nil {
			rollup.Targets = t.Targets
		} else if !isNotFound(terr) {
			return terr
		}
	}
	rollup.Consumed = rollup.Consumed.Add(add)
	return e.store.UpsertRollup(ctx, rollup)
}

// summary builds the acknowledgement reply: totals plus a clarification nudge
// when items were unresolved or overall confidence is low.
func (e *Engine) summary(meal types.Meal, needsClarification int, confidence float64) string {
	t := meal.Total()
	var b strings.Builder
	fmt.Fprintf(&b, "Logged %d item(s).\n", len(meal.Items))
	fmt.Fprintf(&b, "~%.0f kcal | P %.0fg · C %.0fg · F %.0fg", t.Calories, t.Protein, t.Carbs, t.Fat)
	if needsClarification > 0 {
		fmt.Fprintf(&b, "\n⚠ %d item(s) need a portion or weren't recognized — reply with grams.", needsClarification)
	} else if confidence < e.threshold {
		b.WriteString("\n⚠ Low confidence — double-check the amounts.")
	}
	return b.String()
}

func (e *Engine) reply(ctx context.Context, msg types.InboundMessage, text string) error {
	return e.replier.Send(ctx, types.Reply{
		UserID:      msg.UserID,
		Text:        text,
		ChannelMeta: msg.ChannelMeta,
	})
}

func isNotFound(err error) bool {
	return errors.Is(err, types.ErrNotFound) || errors.Is(err, types.ErrNoMatch)
}

// randomID returns a 128-bit random hex id; no external dependency.
func randomID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
