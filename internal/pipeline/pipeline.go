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
	"strconv"
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
	GetUser(ctx context.Context, userID string) (types.User, error)
	SaveMeal(ctx context.Context, m types.Meal) error
	GetTargets(ctx context.Context, userID string) (types.DailyTargets, error)
	SetTargets(ctx context.Context, t types.DailyTargets) error
	GetRollup(ctx context.Context, userID, localDate string) (types.DailyRollup, error)
	UpsertRollup(ctx context.Context, r types.DailyRollup) error
	// Channel mapping (multi-user).
	GetUserIDByChannel(ctx context.Context, channel, channelUserID string) (string, error)
	MapChannelUser(ctx context.Context, channel, channelUserID, userID string) error
}

// Replier sends an in-channel reply. Satisfied by any ports.MessagingAdapter.
type Replier interface {
	Send(ctx context.Context, reply types.Reply) error
}

// Transcriber converts audio to text. Optional; when nil, audio messages are
// replied to with a "text only" prompt. Satisfied by adapters/stt/whisper.Provider.
type Transcriber interface {
	Transcribe(ctx context.Context, audio []byte) (text string, locale string, err error)
}

// PendingStore holds short-lived per-user clarification state. Satisfied by
// internal/pending and any ports.PendingStore.
type PendingStore interface {
	Save(ctx context.Context, pm types.PendingMeal) error
	Get(ctx context.Context, userID string) (types.PendingMeal, error)
	Delete(ctx context.Context, userID string) error
}

// Engine processes one message at a time. It is safe for sequential use by a
// single consumer goroutine draining the queue.
type Engine struct {
	parser      Parser
	resolver    Resolver
	store       MealStore
	pending     PendingStore
	replier     Replier
	transcriber Transcriber // optional STT; nil = audio not supported
	loc         *time.Location
	threshold   float64 // replies flag low confidence when confidence < threshold
	channelName string  // e.g. "telegram", used for user_channels mapping

	now   func() time.Time
	idgen func() string
}

// New builds an Engine. loc is the default timezone used for daily rollup
// boundaries; threshold is the confidence below which a reply nudges the user
// to double-check amounts. pending holds the clarification loop's state.
// channelName is the messaging adapter identifier used for user_channels mapping.
// transcriber is optional (nil = audio messages receive a "text only" reply).
func New(p Parser, r Resolver, s MealStore, pending PendingStore, replier Replier, loc *time.Location, threshold float64, channelName string, transcriber Transcriber) *Engine {
	if loc == nil {
		loc = time.UTC
	}
	return &Engine{
		parser:      p,
		resolver:    r,
		store:       s,
		pending:     pending,
		replier:     replier,
		transcriber: transcriber,
		loc:         loc,
		threshold:   threshold,
		channelName: channelName,
		now:         time.Now,
		idgen:       randomID,
	}
}

// LogMeal directly persists a fully-resolved meal and updates the daily rollup,
// bypassing parsing and resolution. Used by template logging and meal duplication.
func (e *Engine) LogMeal(ctx context.Context, meal types.Meal) error {
	if err := e.store.SaveMeal(ctx, meal); err != nil {
		return fmt.Errorf("pipeline: log meal: %w", err)
	}
	return e.updateRollup(ctx, meal.UserID, meal.At, meal.Total(), e.userLoc(ctx, meal.UserID))
}

// Handle runs the full pipeline for one inbound message. Parsing/resolution
// problems are reported back to the user rather than returned as errors;
// non-nil errors indicate infrastructure failures (store, transport).
func (e *Engine) Handle(ctx context.Context, msg types.InboundMessage) error {
	// STT: transcribe audio before parsing. When audio arrives but STT is not
	// configured, reply with a prompt and return.
	if msg.Kind == types.MessageAudio {
		if e.transcriber == nil {
			return e.reply(ctx, msg, "Audio messages are not supported (STT is disabled). Send your meal as text, e.g. \"200g chicken, 2 eggs\".")
		}
		transcript, locale, err := e.transcriber.Transcribe(ctx, msg.Audio)
		if err != nil {
			return e.reply(ctx, msg, fmt.Sprintf("Couldn't transcribe audio: %v. Try sending your meal as text.", err))
		}
		msg.Text = transcript
		msg.Kind = types.MessageText
		if msg.Locale == "" && locale != "" {
			msg.Locale = locale
		}
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return e.reply(ctx, msg, "Send a meal as text, e.g. \"200g chicken, 2 eggs\".")
	}

	at := msg.At
	if at.IsZero() {
		at = e.now()
	}
	at = at.UTC()

	// Resolve the internal user ID through the channel mapping table.
	// In single-user mode this auto-registers any new channel to "default".
	userID := msg.UserID
	if mapped, err := e.store.GetUserIDByChannel(ctx, e.channelName, msg.UserID); err == nil {
		userID = mapped
	} else {
		// No mapping exists: auto-register this channel ID to the incoming userID.
		_ = e.store.MapChannelUser(ctx, e.channelName, msg.UserID, userID)
	}
	msg.UserID = userID

	// Ensure the user row exists (single-user today; keyed by user from day one).
	if err := e.store.UpsertUser(ctx, types.User{ID: msg.UserID, Timezone: e.loc.String(), CreatedAt: e.now().UTC()}); err != nil {
		return fmt.Errorf("pipeline: upsert user: %w", err)
	}

	// Commands take precedence over meal logging and the clarification loop.
	if strings.HasPrefix(text, "/target") {
		return e.handleTarget(ctx, msg, text)
	}
	if strings.HasPrefix(strings.ToLower(text), "/cancel") {
		return e.cancelPending(ctx, msg)
	}
	if strings.HasPrefix(text, "/timezone") {
		return e.handleTimezone(ctx, msg, text)
	}

	// A live pending meal turns the next message into a clarification answer.
	if pm, err := e.pending.Get(ctx, msg.UserID); err == nil {
		return e.handleClarification(ctx, msg, pm, text)
	} else if !isNotFound(err) {
		return fmt.Errorf("pipeline: get pending: %w", err)
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

	// Fully resolved: log it now.
	if needsClarification == 0 {
		return e.commitMeal(ctx, msg.UserID, msg.ChannelMeta, at, text, confidence, resolved)
	}

	// Some items need a portion or a correction. Hold the meal as pending state
	// and ask back through the channel rather than logging a guessed macro.
	good, open := splitResolved(resolved)
	pm := types.PendingMeal{
		UserID:      msg.UserID,
		At:          at,
		RawText:     text,
		Confidence:  confidence,
		ParserTier:  e.parser.Tier(),
		ChannelMeta: msg.ChannelMeta,
		Resolved:    good,
		Pending:     open,
		CreatedAt:   e.now().UTC(),
	}
	if err := e.pending.Save(ctx, pm); err != nil {
		return fmt.Errorf("pipeline: save pending: %w", err)
	}
	return e.reply(ctx, msg, askText(pm))
}

// handleClarification interprets the user's reply as an answer to the first open
// question of a pending meal: a portion (grams) for a known food, a corrected
// item phrase for an unrecognized food, "skip" to drop it, or "cancel" to
// discard the whole pending meal. When the last question is answered the meal is
// finalized and logged.
func (e *Engine) handleClarification(ctx context.Context, msg types.InboundMessage, pm types.PendingMeal, text string) error {
	lower := strings.ToLower(strings.TrimSpace(text))

	switch lower {
	case "cancel":
		return e.cancelPending(ctx, msg)
	case "skip":
		pm.Pending = pm.Pending[1:] // drop the current question
		return e.advance(ctx, msg, pm)
	}

	q := pm.Pending[0]
	if q.Match.FoodID == "" {
		// Unknown food: treat the reply as a corrected item and re-resolve it.
		items, _, err := e.parser.Extract(ctx, text, msg.Locale)
		if err != nil {
			return fmt.Errorf("pipeline: parse correction: %w", err)
		}
		if len(items) == 0 {
			return e.reply(ctx, msg, "Didn't catch a food there. "+questionText(q))
		}
		re, _ := e.resolver.Resolve(ctx, msg.UserID, items[:1])
		ri := re[0]
		switch {
		case ri.Match.FoodID == "":
			return e.reply(ctx, msg, "Still don't recognize that one. "+questionText(ri))
		case ri.Parsed.NormalizedGrams <= 0:
			// Now recognized, but still no weight: ask for the portion.
			pm.Pending[0] = ri
			if err := e.pending.Save(ctx, pm); err != nil {
				return fmt.Errorf("pipeline: save pending: %w", err)
			}
			return e.reply(ctx, msg, questionText(ri))
		default:
			pm.Resolved = append(pm.Resolved, ri)
			pm.Pending = pm.Pending[1:]
			return e.advance(ctx, msg, pm)
		}
	}

	// Known food, missing portion: expect grams.
	grams, ok := parseGrams(text, q.Parsed.Quantity)
	if !ok {
		return e.reply(ctx, msg, "Reply with a weight in grams. "+questionText(q))
	}
	q.Parsed.NormalizedGrams = grams
	q.Macros = q.Match.Per100g.Scale(grams / 100.0)
	pm.Resolved = append(pm.Resolved, q)
	pm.Pending = pm.Pending[1:]
	return e.advance(ctx, msg, pm)
}

// advance finalizes the meal when no questions remain, otherwise persists the
// updated pending state and asks the next question.
func (e *Engine) advance(ctx context.Context, msg types.InboundMessage, pm types.PendingMeal) error {
	if len(pm.Pending) == 0 {
		if err := e.pending.Delete(ctx, msg.UserID); err != nil {
			return fmt.Errorf("pipeline: delete pending: %w", err)
		}
		if len(pm.Resolved) == 0 {
			return e.reply(ctx, msg, "Nothing left to log — all items were skipped.")
		}
		return e.commitMeal(ctx, pm.UserID, pm.ChannelMeta, pm.At, pm.RawText, pm.Confidence, pm.Resolved)
	}
	if err := e.pending.Save(ctx, pm); err != nil {
		return fmt.Errorf("pipeline: save pending: %w", err)
	}
	return e.reply(ctx, msg, questionText(pm.Pending[0]))
}

// cancelPending discards any open pending meal for the user.
func (e *Engine) cancelPending(ctx context.Context, msg types.InboundMessage) error {
	_, err := e.pending.Get(ctx, msg.UserID)
	if isNotFound(err) {
		return e.reply(ctx, msg, "Nothing pending to cancel.")
	}
	if err != nil {
		return fmt.Errorf("pipeline: get pending: %w", err)
	}
	if err := e.pending.Delete(ctx, msg.UserID); err != nil {
		return fmt.Errorf("pipeline: delete pending: %w", err)
	}
	return e.reply(ctx, msg, "Discarded the pending meal.")
}

// userLoc returns the user's timezone from the database, falling back to the
// engine's default location when the user has no timezone set or on error.
func (e *Engine) userLoc(ctx context.Context, userID string) *time.Location {
	u, err := e.store.GetUser(ctx, userID)
	if err != nil || u.Timezone == "" {
		return e.loc
	}
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return e.loc
	}
	return loc
}

// commitMeal persists a fully resolved meal, folds it into the day's rollup, and
// acknowledges it. Shared by the direct path and the clarification finalize.
func (e *Engine) commitMeal(ctx context.Context, userID string, meta map[string]string, at time.Time, rawText string, confidence float64, items []types.ResolvedItem) error {
	meal := types.Meal{
		ID:         e.idgen(),
		UserID:     userID,
		At:         at,
		RawText:    rawText,
		Items:      items,
		Confidence: confidence,
		ParserTier: e.parser.Tier(),
		CreatedAt:  e.now().UTC(),
	}
	if err := e.store.SaveMeal(ctx, meal); err != nil {
		return fmt.Errorf("pipeline: save meal: %w", err)
	}
	if err := e.updateRollup(ctx, userID, at, meal.Total(), e.userLoc(ctx, userID)); err != nil {
		return fmt.Errorf("pipeline: update rollup: %w", err)
	}
	return e.replyMeta(ctx, userID, meta, e.summary(meal))
}

// splitResolved partitions resolved items into the ones ready to log and the
// ones still needing clarification (no food match, or unknown portion).
func splitResolved(items []types.ResolvedItem) (good, open []types.ResolvedItem) {
	for _, ri := range items {
		if ri.Match.FoodID == "" || ri.Parsed.NormalizedGrams <= 0 {
			open = append(open, ri)
		} else {
			good = append(good, ri)
		}
	}
	return good, open
}

// updateRollup adds the meal's macros to the user's rollup for its local day,
// creating the row (with current targets) on first meal of the day.
func (e *Engine) updateRollup(ctx context.Context, userID string, at time.Time, add types.Macros, loc *time.Location) error {
	localDate := at.In(loc).Format("2006-01-02")

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

// summary builds the acknowledgement reply for a logged meal: totals plus a
// low-confidence nudge. By the time a meal is committed every item is resolved,
// so clarification is handled before this, not here.
func (e *Engine) summary(meal types.Meal) string {
	t := meal.Total()
	var b strings.Builder
	fmt.Fprintf(&b, "Logged %d item(s).\n", len(meal.Items))
	fmt.Fprintf(&b, "~%.0f kcal | P %.0fg · C %.0fg · F %.0fg", t.Calories, t.Protein, t.Carbs, t.Fat)
	if meal.Confidence < e.threshold {
		b.WriteString("\n⚠ Low confidence — double-check the amounts.")
	}
	return b.String()
}

// askText is the first reply when a meal goes pending: a short status plus the
// first open question.
func askText(pm types.PendingMeal) string {
	var b strings.Builder
	if len(pm.Resolved) > 0 {
		fmt.Fprintf(&b, "Got %d item(s). ", len(pm.Resolved))
	}
	n := len(pm.Pending)
	fmt.Fprintf(&b, "%d need%s clarification before I log this.\n", n, plural(n))
	b.WriteString(questionText(pm.Pending[0]))
	if n > 1 {
		fmt.Fprintf(&b, "\n(%d more after this.)", n-1)
	}
	return b.String()
}

// questionText asks for the one piece of information a pending item is missing:
// a portion for a known food, or a correction for an unrecognized one.
func questionText(ri types.ResolvedItem) string {
	if ri.Match.FoodID == "" {
		return fmt.Sprintf("I don't recognize %q. Reply with the food and a weight (e.g. \"120g chicken\"), \"skip\", or \"cancel\".", ri.Parsed.RawPhrase)
	}
	if ri.Parsed.Quantity > 0 && ri.Parsed.Unit != "" && ri.Parsed.Unit != "unit" {
		return fmt.Sprintf("How many grams is %q (%g %s)? Reply e.g. \"100g\" or \"50g each\" — or \"skip\"/\"cancel\".",
			ri.Match.Name, ri.Parsed.Quantity, ri.Parsed.Unit)
	}
	return fmt.Sprintf("How many grams is %q? Reply e.g. \"100g\" — or \"skip\"/\"cancel\".", ri.Match.Name)
}

func plural(n int) string {
	if n == 1 {
		return "s"
	}
	return ""
}

// parseGrams reads a gram weight from a clarification reply. It accepts a bare
// number, a "g"/"grams"/"gramas" suffix, and a per-unit form ("50 each",
// "50g cada") which it multiplies by the item's quantity. ok is false when no
// positive weight can be read.
func parseGrams(text string, qty float64) (float64, bool) {
	t := strings.ToLower(strings.TrimSpace(text))
	perEach := strings.Contains(t, "each") || strings.Contains(t, "cada")

	var num strings.Builder
	for _, r := range t {
		if (r >= '0' && r <= '9') || r == '.' {
			num.WriteRune(r)
		} else if num.Len() > 0 {
			break
		}
	}
	if num.Len() == 0 {
		return 0, false
	}
	g, err := strconv.ParseFloat(num.String(), 64)
	if err != nil || g <= 0 {
		return 0, false
	}
	if perEach && qty > 0 {
		g *= qty
	}
	return g, true
}

// handleTarget sets the user's daily macro goals from a command such as
// "/target kcal=3000 protein=180 carbs=350 fat=90".
func (e *Engine) handleTarget(ctx context.Context, msg types.InboundMessage, text string) error {
	macros, ok := parseTargetCommand(text)
	if !ok {
		return e.reply(ctx, msg, "Usage: /target kcal=3000 protein=180 carbs=350 fat=90")
	}
	if err := e.store.SetTargets(ctx, types.DailyTargets{UserID: msg.UserID, Targets: macros}); err != nil {
		return fmt.Errorf("pipeline: set targets: %w", err)
	}
	return e.reply(ctx, msg, fmt.Sprintf("Targets set: %.0f kcal | P %.0fg · C %.0fg · F %.0fg",
		macros.Calories, macros.Protein, macros.Carbs, macros.Fat))
}

// parseTargetCommand reads "key=value" pairs after "/target" into a Macros. ok
// is false if no recognized key was provided.
func parseTargetCommand(text string) (types.Macros, bool) {
	var m types.Macros
	found := false
	for _, f := range strings.Fields(text)[1:] { // skip "/target"
		k, v, hasEq := strings.Cut(f, "=")
		if !hasEq {
			continue
		}
		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			continue
		}
		switch strings.ToLower(k) {
		case "kcal", "calories", "cal":
			m.Calories, found = val, true
		case "protein", "p":
			m.Protein, found = val, true
		case "carbs", "c":
			m.Carbs, found = val, true
		case "fat", "f":
			m.Fat, found = val, true
		case "fiber":
			m.Fiber, found = val, true
		}
	}
	return m, found
}

// handleTimezone sets the user's timezone from a command such as
// "/timezone America/Sao_Paulo". Validates the IANA name before saving.
func (e *Engine) handleTimezone(ctx context.Context, msg types.InboundMessage, text string) error {
	tz := parseTimezoneCommand(text)
	if tz == "" {
		return e.reply(ctx, msg, "Usage: /timezone <IANA name> (e.g. /timezone America/Sao_Paulo)")
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return e.reply(ctx, msg, fmt.Sprintf("%q is not a valid IANA timezone.", tz))
	}
	// Update the user record with the new timezone.
	u, err := e.store.GetUser(ctx, msg.UserID)
	if err != nil {
		u = types.User{ID: msg.UserID, CreatedAt: e.now().UTC()}
	}
	u.Timezone = loc.String()
	if err := e.store.UpsertUser(ctx, u); err != nil {
		return fmt.Errorf("pipeline: upsert user timezone: %w", err)
	}
	return e.reply(ctx, msg, fmt.Sprintf("Timezone set to %s.", loc.String()))
}

// parseTimezoneCommand extracts the first word after "/timezone". Returns ""
// when nothing follows the command.
func parseTimezoneCommand(text string) string {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func (e *Engine) reply(ctx context.Context, msg types.InboundMessage, text string) error {
	return e.replyMeta(ctx, msg.UserID, msg.ChannelMeta, text)
}

func (e *Engine) replyMeta(ctx context.Context, userID string, meta map[string]string, text string) error {
	return e.replier.Send(ctx, types.Reply{
		UserID:      userID,
		Text:        text,
		ChannelMeta: meta,
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
