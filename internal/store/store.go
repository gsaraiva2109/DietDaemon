// Package store implements ports.Store with SQLite via a pure-Go driver
// (modernc.org/sqlite). It is CGO-free so the Dockerfile static build works.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/normalize"
	"github.com/gsaraiva2109/dietdaemon/internal/scheduler"
	"github.com/gsaraiva2109/dietdaemon/migrations"
)

// Store implements ports.Store backed by SQLite.
type Store struct {
	db *sql.DB
}

// Compile-time guarantees that Store satisfies every interface boundary it must.
var (
	_ ports.Store      = (*Store)(nil)
	_ scheduler.Store    = (*Store)(nil)
	_ scheduler.NudgeStore = (*Store)(nil)
	_ ports.PendingStore   = (*Store)(nil)
)

// New opens the SQLite database at dbPath, enables foreign keys and WAL mode,
// runs migrations, and returns a ready Store.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store: open db: %w", err)
	}

	// Enable WAL mode for concurrent reads and writes.
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: enable WAL: %w", err)
	}

	// Enforce foreign keys at the connection level.
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: enable foreign keys: %w", err)
	}

	s := &Store{db: db}
	if err := s.runMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: migrate: %w", err)
	}

	return s, nil
}

func (s *Store) runMigrations() error {
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := migrations.FS.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		if _, err := s.db.Exec(string(content)); err != nil {
			return fmt.Errorf("exec %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

// UpsertUser inserts or replaces a user row.
func (s *Store) UpsertUser(ctx context.Context, u types.User) error {
	const q = `
		INSERT OR REPLACE INTO users (id, timezone, created_at)
		VALUES (?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, q, u.ID, u.Timezone, utcStr(u.CreatedAt))
	return err
}

// GetUser returns the user or types.ErrNotFound.
func (s *Store) GetUser(ctx context.Context, userID string) (types.User, error) {
	const q = `SELECT id, timezone, created_at FROM users WHERE id = ?`
	row := s.db.QueryRowContext(ctx, q, userID)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return types.User{}, types.ErrNotFound
	}
	return u, err
}

// ListUsers returns every user. Empty slice, nil error when there are none.
func (s *Store) ListUsers(ctx context.Context) ([]types.User, error) {
	const q = `SELECT id, timezone, created_at FROM users ORDER BY id`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store: list users: %w", err)
	}
	defer rows.Close()

	var users []types.User
	for rows.Next() {
		var u types.User
		var ca string
		if err := rows.Scan(&u.ID, &u.Timezone, &ca); err != nil {
			return nil, fmt.Errorf("store: scan user: %w", err)
		}
		u.CreatedAt = parseUTC(ca)
		users = append(users, u)
	}
	return users, rows.Err()
}

func scanUser(row *sql.Row) (types.User, error) {
	var u types.User
	var ca string
	if err := row.Scan(&u.ID, &u.Timezone, &ca); err != nil {
		return types.User{}, err
	}
	u.CreatedAt = parseUTC(ca)
	return u, nil
}

// ---------------------------------------------------------------------------
// Meals
// ---------------------------------------------------------------------------

// SaveMeal inserts a meal and all its resolved items inside a transaction.
func (s *Store) SaveMeal(ctx context.Context, m types.Meal) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	const mealQ = `
		INSERT INTO meals (id, user_id, at_utc, raw_text, confidence, parser_tier, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err = tx.ExecContext(ctx, mealQ,
		m.ID, m.UserID, utcStr(m.At), m.RawText, m.Confidence, int(m.ParserTier), utcStr(m.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("store: insert meal: %w", err)
	}

	const itemQ = `
		INSERT INTO resolved_items
			(id, meal_id, raw_phrase, quantity, unit, normalized_grams,
			 food_id, food_name, source, match_score,
			 kcal, protein, carbs, fat, fiber)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	for _, it := range m.Items {
		_, err = tx.ExecContext(ctx, itemQ,
			newID(), m.ID,
			it.Parsed.RawPhrase, it.Parsed.Quantity, it.Parsed.Unit, it.Parsed.NormalizedGrams,
			it.Match.FoodID, it.Match.Name, it.Match.Source, it.Match.MatchScore,
			it.Macros.Calories, it.Macros.Protein, it.Macros.Carbs, it.Macros.Fat, it.Macros.Fiber,
		)
		if err != nil {
			return fmt.Errorf("store: insert resolved_item: %w", err)
		}
	}

	return tx.Commit()
}

// RecentMeals returns the most recent meals for a user, each with its resolved
// items populated. Meals are ordered newest-first.
func (s *Store) RecentMeals(ctx context.Context, userID string, limit int) ([]types.Meal, error) {
	const mealQ = `
		SELECT id, user_id, at_utc, raw_text, confidence, parser_tier, created_at
		FROM meals
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, mealQ, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("store: query meals: %w", err)
	}
	defer rows.Close()

	var meals []types.Meal
	var mealIDs []string
	for rows.Next() {
		m, err := scanMeal(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan meal: %w", err)
		}
		meals = append(meals, m)
		mealIDs = append(mealIDs, m.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: meals rows: %w", err)
	}

	if len(meals) == 0 {
		return meals, nil
	}

	// Fetch all resolved items for the retrieved meals.
	itemsByMeal, err := s.loadItems(ctx, mealIDs)
	if err != nil {
		return nil, err
	}

	for i := range meals {
		meals[i].Items = itemsByMeal[meals[i].ID]
	}

	return meals, nil
}

func scanMeal(rows *sql.Rows) (types.Meal, error) {
	var m types.Meal
	var atUTC, ca string
	var tier int
	if err := rows.Scan(&m.ID, &m.UserID, &atUTC, &m.RawText, &m.Confidence, &tier, &ca); err != nil {
		return types.Meal{}, err
	}
	m.At = parseUTC(atUTC)
	m.ParserTier = types.ParserTier(tier)
	m.CreatedAt = parseUTC(ca)
	return m, nil
}

func (s *Store) loadItems(ctx context.Context, mealIDs []string) (map[string][]types.ResolvedItem, error) {
	if len(mealIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(mealIDs))
	args := make([]interface{}, len(mealIDs))
	for i, id := range mealIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	q := fmt.Sprintf(`
		SELECT meal_id, raw_phrase, quantity, unit, normalized_grams,
		       food_id, food_name, source, match_score,
		       kcal, protein, carbs, fat, fiber
		FROM resolved_items
		WHERE meal_id IN (%s)
		ORDER BY meal_id, rowid
	`, strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query items: %w", err)
	}
	defer rows.Close()

	out := make(map[string][]types.ResolvedItem)
	for rows.Next() {
		var mealID string
		var ri types.ResolvedItem
		var parsedNMG float64 // normalized grams — needed to back-calc Per100g
		err := rows.Scan(
			&mealID, &ri.Parsed.RawPhrase, &ri.Parsed.Quantity, &ri.Parsed.Unit, &parsedNMG,
			&ri.Match.FoodID, &ri.Match.Name, &ri.Match.Source, &ri.Match.MatchScore,
			&ri.Macros.Calories, &ri.Macros.Protein, &ri.Macros.Carbs, &ri.Macros.Fat, &ri.Macros.Fiber,
		)
		if err != nil {
			return nil, fmt.Errorf("store: scan item: %w", err)
		}
		ri.Parsed.NormalizedGrams = parsedNMG
		// Reconstruct Per100g from the absolute macros and portion grams.
		ri.Match.Per100g = macrosPer100g(ri.Macros, parsedNMG)
		out[mealID] = append(out[mealID], ri)
	}
	return out, rows.Err()
}

// macrosPer100g back-calculates per-100g macros from the absolute portion
// macros. If grams is zero or negative the absolute macros are returned as-is.
func macrosPer100g(m types.Macros, grams float64) types.Macros {
	if grams <= 0 {
		return m
	}
	return m.Scale(100.0 / grams)
}

// ---------------------------------------------------------------------------
// Personal food library
// ---------------------------------------------------------------------------

// LookupFood matches a phrase against the user's food aliases, joins to
// food_library, and returns the top match ordered by query_count DESC.
// Returns types.ErrNoMatch when no alias matches.
func (s *Store) LookupFood(ctx context.Context, userID, phrase string) (types.FoodMatch, error) {
	normalized := normalize.Normalize(phrase)

	const q = `
		SELECT fl.food_id, fl.name, fl.source, fl.kcal_100g, fl.protein_100g,
		       fl.carbs_100g, fl.fat_100g, fl.fiber_100g
		FROM food_aliases fa
		JOIN food_library fl ON fl.user_id = fa.user_id AND fl.food_id = fa.food_id
		WHERE fa.user_id = ? AND fa.alias_normalized = ?
		ORDER BY fl.query_count DESC
		LIMIT 1
	`
	row := s.db.QueryRowContext(ctx, q, userID, normalized)
	var fm types.FoodMatch
	err := row.Scan(&fm.FoodID, &fm.Name, &fm.Source,
		&fm.Per100g.Calories, &fm.Per100g.Protein, &fm.Per100g.Carbs, &fm.Per100g.Fat, &fm.Per100g.Fiber,
	)
	if err == sql.ErrNoRows {
		return types.FoodMatch{}, types.ErrNoMatch
	}
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("store: lookup food: %w", err)
	}
	// Exact alias match always scores 1.0.
	fm.MatchScore = 1.0
	return fm, nil
}

// UpsertFood inserts or replaces a food_library row and adds any new normalized
// aliases, all within a single transaction.
func (s *Store) UpsertFood(ctx context.Context, userID string, match types.FoodMatch, aliases []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	const foodQ = `
		INSERT INTO food_library
			(food_id, user_id, name, source, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g, query_count, last_used)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, '')
		ON CONFLICT(user_id, food_id) DO UPDATE SET
			name        = excluded.name,
			source      = excluded.source,
			kcal_100g   = excluded.kcal_100g,
			protein_100g= excluded.protein_100g,
			carbs_100g  = excluded.carbs_100g,
			fat_100g    = excluded.fat_100g,
			fiber_100g  = excluded.fiber_100g
	`
	_, err = tx.ExecContext(ctx, foodQ,
		match.FoodID, userID, match.Name, match.Source,
		match.Per100g.Calories, match.Per100g.Protein, match.Per100g.Carbs, match.Per100g.Fat, match.Per100g.Fiber,
	)
	if err != nil {
		return fmt.Errorf("store: upsert food: %w", err)
	}

	const aliasQ = `
		INSERT OR IGNORE INTO food_aliases (user_id, alias_normalized, food_id)
		VALUES (?, ?, ?)
	`
	for _, alias := range aliases {
		normalized := normalize.Normalize(alias)
		if normalized == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, aliasQ, userID, normalized, match.FoodID); err != nil {
			return fmt.Errorf("store: insert alias: %w", err)
		}
	}

	return tx.Commit()
}

// RecordFoodQuery bumps query_count and sets last_used to now.
func (s *Store) RecordFoodQuery(ctx context.Context, userID, foodID string) error {
	const q = `
		UPDATE food_library
		SET query_count = query_count + 1, last_used = ?
		WHERE user_id = ? AND food_id = ?
	`
	_, err := s.db.ExecContext(ctx, q, utcNow(), userID, foodID)
	return err
}

// ---------------------------------------------------------------------------
// Targets
// ---------------------------------------------------------------------------

// GetTargets returns the daily targets for a user, or types.ErrNotFound.
func (s *Store) GetTargets(ctx context.Context, userID string) (types.DailyTargets, error) {
	const q = `SELECT user_id, kcal, protein, carbs, fat, fiber FROM daily_targets WHERE user_id = ?`
	row := s.db.QueryRowContext(ctx, q, userID)
	dt, err := scanTargets(row)
	if err == sql.ErrNoRows {
		return types.DailyTargets{}, types.ErrNotFound
	}
	return dt, err
}

func scanTargets(row *sql.Row) (types.DailyTargets, error) {
	var dt types.DailyTargets
	err := row.Scan(&dt.UserID,
		&dt.Targets.Calories, &dt.Targets.Protein, &dt.Targets.Carbs, &dt.Targets.Fat, &dt.Targets.Fiber,
	)
	return dt, err
}

// SetTargets inserts or replaces the daily targets row.
func (s *Store) SetTargets(ctx context.Context, t types.DailyTargets) error {
	const q = `
		INSERT OR REPLACE INTO daily_targets (user_id, kcal, protein, carbs, fat, fiber)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, q, t.UserID,
		t.Targets.Calories, t.Targets.Protein, t.Targets.Carbs, t.Targets.Fat, t.Targets.Fiber,
	)
	return err
}

// ---------------------------------------------------------------------------
// Rollups
// ---------------------------------------------------------------------------

// GetRollup returns a daily rollup, or types.ErrNotFound.
func (s *Store) GetRollup(ctx context.Context, userID, localDate string) (types.DailyRollup, error) {
	const q = `
		SELECT user_id, date,
		       consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
		       target_kcal, target_protein, target_carbs, target_fat, target_fiber
		FROM daily_rollups
		WHERE user_id = ? AND date = ?
	`
	row := s.db.QueryRowContext(ctx, q, userID, localDate)
	r, err := scanRollup(row)
	if err == sql.ErrNoRows {
		return types.DailyRollup{}, types.ErrNotFound
	}
	return r, err
}

func scanRollup(row *sql.Row) (types.DailyRollup, error) {
	var r types.DailyRollup
	err := row.Scan(&r.UserID, &r.Date,
		&r.Consumed.Calories, &r.Consumed.Protein, &r.Consumed.Carbs, &r.Consumed.Fat, &r.Consumed.Fiber,
		&r.Targets.Calories, &r.Targets.Protein, &r.Targets.Carbs, &r.Targets.Fat, &r.Targets.Fiber,
	)
	return r, err
}

// UpsertRollup inserts or replaces a daily rollup row.
func (s *Store) UpsertRollup(ctx context.Context, r types.DailyRollup) error {
	const q = `
		INSERT OR REPLACE INTO daily_rollups
			(user_id, date,
			 consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
			 target_kcal, target_protein, target_carbs, target_fat, target_fiber)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, q,
		r.UserID, r.Date,
		r.Consumed.Calories, r.Consumed.Protein, r.Consumed.Carbs, r.Consumed.Fat, r.Consumed.Fiber,
		r.Targets.Calories, r.Targets.Protein, r.Targets.Carbs, r.Targets.Fat, r.Targets.Fiber,
	)
	return err
}

// ---------------------------------------------------------------------------
// Nudge dedupe
// ---------------------------------------------------------------------------

// WasNudged reports whether ruleID has already fired for this user on
// localDate. Satisfies scheduler.NudgeStore.
func (s *Store) WasNudged(ctx context.Context, userID, localDate, ruleID string) (bool, error) {
	const q = `SELECT 1 FROM nudge_log WHERE user_id = ? AND local_date = ? AND rule_id = ?`
	row := s.db.QueryRowContext(ctx, q, userID, localDate, ruleID)
	var v int
	err := row.Scan(&v)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: was-nudged: %w", err)
	}
	return true, nil
}

// MarkNudged records that ruleID fired for this user on localDate. Idempotent
// (INSERT OR IGNORE). Satisfies scheduler.NudgeStore.
func (s *Store) MarkNudged(ctx context.Context, userID, localDate, ruleID string) error {
	const q = `
		INSERT OR IGNORE INTO nudge_log (user_id, local_date, rule_id, sent_at)
		VALUES (?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, q, userID, localDate, ruleID, utcNow())
	return err
}

// ---------------------------------------------------------------------------
// Pending meals (durable — survives restart)
// ---------------------------------------------------------------------------

// Save inserts or replaces the pending meal for the given user.
func (s *Store) Save(ctx context.Context, pm types.PendingMeal) error {
	metaJSON, err := json.Marshal(pm.ChannelMeta)
	if err != nil {
		return fmt.Errorf("store: marshal channel_meta: %w", err)
	}
	resolvedJSON, err := json.Marshal(pm.Resolved)
	if err != nil {
		return fmt.Errorf("store: marshal resolved: %w", err)
	}
	pendingJSON, err := json.Marshal(pm.Pending)
	if err != nil {
		return fmt.Errorf("store: marshal pending: %w", err)
	}

	const q = `
		INSERT OR REPLACE INTO pending_meals
			(user_id, at_utc, raw_text, confidence, parser_tier, channel_meta, resolved, pending, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.ExecContext(ctx, q,
		pm.UserID, utcStr(pm.At), pm.RawText, pm.Confidence, int(pm.ParserTier),
		string(metaJSON), string(resolvedJSON), string(pendingJSON), utcStr(pm.CreatedAt),
	)
	return err
}

// Get returns the live pending meal for userID, or types.ErrNotFound.
func (s *Store) Get(ctx context.Context, userID string) (types.PendingMeal, error) {
	const q = `
		SELECT user_id, at_utc, raw_text, confidence, parser_tier,
		       channel_meta, resolved, pending, created_at
		FROM pending_meals
		WHERE user_id = ?
	`
	row := s.db.QueryRowContext(ctx, q, userID)

	var pm types.PendingMeal
	var atUTC, ca string
	var tier int
	var metaJSON, resolvedJSON, pendingJSON string
	err := row.Scan(&pm.UserID, &atUTC, &pm.RawText, &pm.Confidence, &tier,
		&metaJSON, &resolvedJSON, &pendingJSON, &ca,
	)
	if err == sql.ErrNoRows {
		return types.PendingMeal{}, types.ErrNotFound
	}
	if err != nil {
		return types.PendingMeal{}, fmt.Errorf("store: get pending: %w", err)
	}

	pm.At = parseUTC(atUTC)
	pm.ParserTier = types.ParserTier(tier)
	pm.CreatedAt = parseUTC(ca)

	if err := json.Unmarshal([]byte(metaJSON), &pm.ChannelMeta); err != nil {
		return types.PendingMeal{}, fmt.Errorf("store: unmarshal channel_meta: %w", err)
	}
	if err := json.Unmarshal([]byte(resolvedJSON), &pm.Resolved); err != nil {
		return types.PendingMeal{}, fmt.Errorf("store: unmarshal resolved: %w", err)
	}
	if err := json.Unmarshal([]byte(pendingJSON), &pm.Pending); err != nil {
		return types.PendingMeal{}, fmt.Errorf("store: unmarshal pending: %w", err)
	}

	return pm, nil
}

// Delete removes any pending meal for userID. Idempotent.
func (s *Store) Delete(ctx context.Context, userID string) error {
	const q = `DELETE FROM pending_meals WHERE user_id = ?`
	_, err := s.db.ExecContext(ctx, q, userID)
	return err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func utcStr(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func utcNow() string { return time.Now().UTC().Format(time.RFC3339) }

func parseUTC(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// newID returns a short pseudo-unique ID using a monotonic counter + timestamp
// fallback. Simple identifiers keep the embedded DB readable.
var idCounter int64

func newID() string {
	n := atomic.AddInt64(&idCounter, 1)
	return fmt.Sprintf("%d%x", time.Now().UnixNano(), n)
}
