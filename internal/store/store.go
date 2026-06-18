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
	_ ports.Store          = (*Store)(nil)
	_ scheduler.Store      = (*Store)(nil)
	_ scheduler.NudgeStore = (*Store)(nil)
)

// New opens the SQLite database at dbPath, enables foreign keys and WAL mode,
// runs migrations, and returns a ready Store.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store: open db: %w", err)
	}

	// SQLite is single-writer; a pool of 1 guarantees every PRAGMA and all
	// subsequent queries share the same connection, avoiding SQLITE_CANTOPEN
	// when a second connection races the EXCLUSIVE lock below.
	db.SetMaxOpenConns(1)

	// EXCLUSIVE locking before WAL avoids shared-memory (-shm) entirely.
	// SQLite keeps the wal-index in heap memory instead of mmap'ing a file.
	// Required for Docker Swarm / some overlay filesystems where the VFS
	// shared-memory primitives (xShmMap, xShmLock, etc.) don't work.
	if _, err := db.Exec("PRAGMA locking_mode = EXCLUSIVE"); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: exclusive lock: %w", err)
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
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (datetime('now')))`); err != nil {
		return fmt.Errorf("init migration tracking: %w", err)
	}

	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	// If the tracking table is empty but the DB already has tables (existing
	// install upgrading from before migration tracking existed), mark all
	// current migration files as already applied so they are not re-run.
	var tracked int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&tracked); err != nil {
		return fmt.Errorf("count tracked migrations: %w", err)
	}
	if tracked == 0 {
		var hasTables int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='meals'`).Scan(&hasTables); err != nil {
			return fmt.Errorf("check existing schema: %w", err)
		}
		if hasTables > 0 {
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
					continue
				}
				if _, err := s.db.Exec(`INSERT OR IGNORE INTO schema_migrations (name) VALUES (?)`, entry.Name()); err != nil {
					return fmt.Errorf("record legacy migration %s: %w", entry.Name(), err)
				}
			}
		}
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		var already int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, entry.Name()).Scan(&already); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if already > 0 {
			continue
		}

		content, err := migrations.FS.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		if _, err := s.db.Exec(string(content)); err != nil {
			return fmt.Errorf("exec %s: %w", entry.Name(), err)
		}
		if _, err := s.db.Exec(`INSERT INTO schema_migrations (name) VALUES (?)`, entry.Name()); err != nil {
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// DB returns the underlying *sql.DB so that callers (e.g. the embedding index)
// can operate on the same database connection without opening a second one.
func (s *Store) DB() *sql.DB { return s.db }

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

// ValidateToken looks up a Bearer token in the api_tokens table and returns the
// owning userID. Returns types.ErrNotFound when the token is invalid or expired.
// In single-user mode this method is not called; the static API_AUTH_TOKEN is
// checked directly.
func (s *Store) ValidateToken(ctx context.Context, token string) (string, error) {
	const q = `SELECT user_id FROM api_tokens WHERE token = ?`
	row := s.db.QueryRowContext(ctx, q, token)
	var userID string
	if err := row.Scan(&userID); err == sql.ErrNoRows {
		return "", types.ErrNotFound
	} else if err != nil {
		return "", fmt.Errorf("store: validate token: %w", err)
	}
	return userID, nil
}

// UpsertUserTimezone updates the users.timezone column for a user.
func (s *Store) UpsertUserTimezone(ctx context.Context, userID, timezone string) error {
	const q = `UPDATE users SET timezone = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, q, timezone, userID)
	return err
}

// MapChannelUser inserts a mapping from a messaging channel + channel_user_id
// to an internal user_id. It is idempotent (INSERT OR IGNORE).
func (s *Store) MapChannelUser(ctx context.Context, channel, channelUserID, userID string) error {
	const q = `
		INSERT OR IGNORE INTO user_channels (channel, channel_user_id, user_id)
		VALUES (?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, q, channel, channelUserID, userID)
	return err
}

// GetUserIDByChannel returns the internal user_id for a given
// (channel, channel_user_id) pair. Returns types.ErrNotFound when no mapping
// exists.
func (s *Store) GetUserIDByChannel(ctx context.Context, channel, channelUserID string) (string, error) {
	const q = `SELECT user_id FROM user_channels WHERE channel = ? AND channel_user_id = ?`
	row := s.db.QueryRowContext(ctx, q, channel, channelUserID)
	var userID string
	if err := row.Scan(&userID); err == sql.ErrNoRows {
		return "", types.ErrNotFound
	} else if err != nil {
		return "", fmt.Errorf("store: get user by channel: %w", err)
	}
	return userID, nil
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

// GetMeal returns a single meal by ID with its resolved items populated.
// Returns types.ErrNotFound when the meal does not exist.
func (s *Store) GetMeal(ctx context.Context, mealID string) (types.Meal, error) {
	const q = `
		SELECT id, user_id, at_utc, raw_text, confidence, parser_tier, created_at
		FROM meals WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, q, mealID)

	var m types.Meal
	var atUTC, ca string
	var tier int
	if err := row.Scan(&m.ID, &m.UserID, &atUTC, &m.RawText, &m.Confidence, &tier, &ca); err != nil {
		if err == sql.ErrNoRows {
			return types.Meal{}, types.ErrNotFound
		}
		return types.Meal{}, fmt.Errorf("store: get meal: %w", err)
	}
	m.At = parseUTC(atUTC)
	m.ParserTier = types.ParserTier(tier)
	m.CreatedAt = parseUTC(ca)

	itemsByMeal, err := s.loadItems(ctx, []string{m.ID})
	if err != nil {
		return types.Meal{}, err
	}
	m.Items = itemsByMeal[m.ID]
	return m, nil
}

// GetRollups returns daily rollups for a user between startDate and endDate
// (inclusive, "YYYY-MM-DD" format). Ordered by date ascending.
func (s *Store) GetRollups(ctx context.Context, userID, startDate, endDate string) ([]types.DailyRollup, error) {
	const q = `
		SELECT user_id, date,
		       consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
		       target_kcal, target_protein, target_carbs, target_fat, target_fiber
		FROM daily_rollups
		WHERE user_id = ? AND date >= ? AND date <= ?
		ORDER BY date ASC
	`
	rows, err := s.db.QueryContext(ctx, q, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("store: query rollups: %w", err)
	}
	defer rows.Close()

	var out []types.DailyRollup
	for rows.Next() {
		var r types.DailyRollup
		if err := rows.Scan(&r.UserID, &r.Date,
			&r.Consumed.Calories, &r.Consumed.Protein, &r.Consumed.Carbs, &r.Consumed.Fat, &r.Consumed.Fiber,
			&r.Targets.Calories, &r.Targets.Protein, &r.Targets.Carbs, &r.Targets.Fat, &r.Targets.Fiber,
		); err != nil {
			return nil, fmt.Errorf("store: scan rollup: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
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
	args := make([]any, len(mealIDs))
	for i, id := range mealIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	// #nosec G201 -- placeholder expansion is ? only, values are args
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

// GetFood loads a food by its (userID, foodID) primary key. Returns
// types.ErrNoMatch when the food does not exist in the library.
func (s *Store) GetFood(ctx context.Context, userID, foodID string) (types.FoodMatch, error) {
	const q = `
		SELECT food_id, name, source, kcal_100g, protein_100g,
		       carbs_100g, fat_100g, fiber_100g
		FROM food_library
		WHERE user_id = ? AND food_id = ?
	`
	row := s.db.QueryRowContext(ctx, q, userID, foodID)
	var fm types.FoodMatch
	err := row.Scan(&fm.FoodID, &fm.Name, &fm.Source,
		&fm.Per100g.Calories, &fm.Per100g.Protein, &fm.Per100g.Carbs, &fm.Per100g.Fat, &fm.Per100g.Fiber,
	)
	if err == sql.ErrNoRows {
		return types.FoodMatch{}, types.ErrNoMatch
	}
	if err != nil {
		return types.FoodMatch{}, fmt.Errorf("store: get food: %w", err)
	}
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

// CorrectMealItem updates one resolved item's macros for a meal, then
// recalculates the daily rollup and refreshes the food_library cache so future
// logs use the corrected values. itemIndex is the 0-based position of the item
// within the meal's items (ordered by rowid).
func (s *Store) CorrectMealItem(ctx context.Context, userID string, mealID string, itemIndex int, corrected types.ResolvedItem) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	// Load the meal to get the at time for rollup lookup and the original items.
	var atUTC string
	const mealQ = `SELECT at_utc FROM meals WHERE id = ?`
	if err := tx.QueryRowContext(ctx, mealQ, mealID).Scan(&atUTC); err != nil {
		if err == sql.ErrNoRows {
			return types.ErrNotFound
		}
		return fmt.Errorf("store: get meal: %w", err)
	}
	mealAt := parseUTC(atUTC)

	// Load items by rowid so we can find and update the target item.
	const itemsQ = `
		SELECT rowid, raw_phrase, quantity, unit, normalized_grams,
		       food_id, food_name, source, match_score,
		       kcal, protein, carbs, fat, fiber
		FROM resolved_items
		WHERE meal_id = ?
		ORDER BY rowid
	`
	rows, err := tx.QueryContext(ctx, itemsQ, mealID)
	if err != nil {
		return fmt.Errorf("store: query items: %w", err)
	}
	defer rows.Close()

	type itemRow struct {
		rowid int64
		ri    types.ResolvedItem
	}
	var items []itemRow
	var oldTotal types.Macros
	for rows.Next() {
		var ir itemRow
		var parsedNMG float64
		if err := rows.Scan(&ir.rowid,
			&ir.ri.Parsed.RawPhrase, &ir.ri.Parsed.Quantity, &ir.ri.Parsed.Unit, &parsedNMG,
			&ir.ri.Match.FoodID, &ir.ri.Match.Name, &ir.ri.Match.Source, &ir.ri.Match.MatchScore,
			&ir.ri.Macros.Calories, &ir.ri.Macros.Protein, &ir.ri.Macros.Carbs, &ir.ri.Macros.Fat, &ir.ri.Macros.Fiber,
		); err != nil {
			return fmt.Errorf("store: scan item: %w", err)
		}
		ir.ri.Parsed.NormalizedGrams = parsedNMG
		ir.ri.Match.Per100g = macrosPer100g(ir.ri.Macros, parsedNMG)
		items = append(items, ir)
		oldTotal = oldTotal.Add(ir.ri.Macros)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: items rows: %w", err)
	}
	if itemIndex < 0 || itemIndex >= len(items) {
		return fmt.Errorf("store: item index %d out of range [0, %d)", itemIndex, len(items))
	}

	// Replace the target item's macros and recalculate the new total.
	oldItemMacros := items[itemIndex].ri.Macros
	items[itemIndex].ri = corrected

	var newTotal types.Macros
	for _, ir := range items {
		newTotal = newTotal.Add(ir.ri.Macros)
	}

	// Update the resolved_items row.
	const updateQ = `
		UPDATE resolved_items SET
			normalized_grams = ?, food_id = ?, food_name = ?, source = ?, match_score = ?,
			kcal = ?, protein = ?, carbs = ?, fat = ?, fiber = ?
		WHERE rowid = ?
	`
	_, err = tx.ExecContext(ctx, updateQ,
		corrected.Parsed.NormalizedGrams,
		corrected.Match.FoodID, corrected.Match.Name, corrected.Match.Source, corrected.Match.MatchScore,
		corrected.Macros.Calories, corrected.Macros.Protein, corrected.Macros.Carbs, corrected.Macros.Fat, corrected.Macros.Fiber,
		items[itemIndex].rowid,
	)
	if err != nil {
		return fmt.Errorf("store: update item: %w", err)
	}

	// Update the daily rollup: remove old macros, add new ones.
	localDate := mealAt.Format("2006-01-02")
	const rollupQ = `
		INSERT INTO daily_rollups
			(user_id, date,
			 consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
			 target_kcal, target_protein, target_carbs, target_fat, target_fiber)
		VALUES (?, ?,
		        ?, ?, ?, ?, ?,
		        0, 0, 0, 0, 0)
		ON CONFLICT(user_id, date) DO UPDATE SET
			consumed_kcal   = consumed_kcal   - ? + ?,
			consumed_protein = consumed_protein - ? + ?,
			consumed_carbs  = consumed_carbs  - ? + ?,
			consumed_fat    = consumed_fat    - ? + ?,
			consumed_fiber  = consumed_fiber  - ? + ?
	`
	_, err = tx.ExecContext(ctx, rollupQ,
		userID, localDate,
		newTotal.Calories, newTotal.Protein, newTotal.Carbs, newTotal.Fat, newTotal.Fiber,
		oldItemMacros.Calories, corrected.Macros.Calories,
		oldItemMacros.Protein, corrected.Macros.Protein,
		oldItemMacros.Carbs, corrected.Macros.Carbs,
		oldItemMacros.Fat, corrected.Macros.Fat,
		oldItemMacros.Fiber, corrected.Macros.Fiber,
	)
	if err != nil {
		return fmt.Errorf("store: update rollup: %w", err)
	}

	// Refresh the food_library cache: upsert the corrected food so future
	// alias lookups use the corrected macros.
	if corrected.Match.FoodID != "" {
		const foodQ = `
			INSERT INTO food_library
				(food_id, user_id, name, source, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g, query_count, last_used)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, '')
			ON CONFLICT(user_id, food_id) DO UPDATE SET
				kcal_100g   = excluded.kcal_100g,
				protein_100g= excluded.protein_100g,
				carbs_100g  = excluded.carbs_100g,
				fat_100g    = excluded.fat_100g,
				fiber_100g  = excluded.fiber_100g
		`
		_, err = tx.ExecContext(ctx, foodQ,
			corrected.Match.FoodID, userID, corrected.Match.Name, corrected.Match.Source,
			corrected.Match.Per100g.Calories, corrected.Match.Per100g.Protein,
			corrected.Match.Per100g.Carbs, corrected.Match.Per100g.Fat, corrected.Match.Per100g.Fiber,
		)
		if err != nil {
			return fmt.Errorf("store: upsert food library: %w", err)
		}
	}

	return tx.Commit()
}

// AddMealItem appends a resolved item to an existing meal and adds its macros
// to that day's rollup. Mirrors CorrectMealItem's delta approach.
func (s *Store) AddMealItem(ctx context.Context, userID, mealID string, item types.ResolvedItem) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	var atUTC, mealUser string
	const mealQ = `SELECT at_utc, user_id FROM meals WHERE id = ?`
	if err := tx.QueryRowContext(ctx, mealQ, mealID).Scan(&atUTC, &mealUser); err != nil {
		if err == sql.ErrNoRows {
			return types.ErrNotFound
		}
		return fmt.Errorf("store: get meal: %w", err)
	}
	if mealUser != userID {
		return types.ErrNotFound
	}

	const itemQ = `
		INSERT INTO resolved_items
			(id, meal_id, raw_phrase, quantity, unit, normalized_grams,
			 food_id, food_name, source, match_score,
			 kcal, protein, carbs, fat, fiber)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	if _, err := tx.ExecContext(ctx, itemQ,
		newID(), mealID,
		item.Parsed.RawPhrase, item.Parsed.Quantity, item.Parsed.Unit, item.Parsed.NormalizedGrams,
		item.Match.FoodID, item.Match.Name, item.Match.Source, item.Match.MatchScore,
		item.Macros.Calories, item.Macros.Protein, item.Macros.Carbs, item.Macros.Fat, item.Macros.Fiber,
	); err != nil {
		return fmt.Errorf("store: insert resolved_item: %w", err)
	}

	localDate := parseUTC(atUTC).Format("2006-01-02")
	const rollupQ = `
		INSERT INTO daily_rollups
			(user_id, date,
			 consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
			 target_kcal, target_protein, target_carbs, target_fat, target_fiber)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0, 0, 0, 0)
		ON CONFLICT(user_id, date) DO UPDATE SET
			consumed_kcal    = consumed_kcal    + ?,
			consumed_protein = consumed_protein + ?,
			consumed_carbs   = consumed_carbs   + ?,
			consumed_fat     = consumed_fat     + ?,
			consumed_fiber   = consumed_fiber   + ?
	`
	m := item.Macros
	if _, err := tx.ExecContext(ctx, rollupQ,
		userID, localDate,
		m.Calories, m.Protein, m.Carbs, m.Fat, m.Fiber,
		m.Calories, m.Protein, m.Carbs, m.Fat, m.Fiber,
	); err != nil {
		return fmt.Errorf("store: update rollup: %w", err)
	}

	return tx.Commit()
}

// DeleteMealItem removes the item at itemIndex (zero-based, rowid order) from a
// meal and subtracts its macros from that day's rollup.
func (s *Store) DeleteMealItem(ctx context.Context, userID, mealID string, itemIndex int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	var atUTC, mealUser string
	const mealQ = `SELECT at_utc, user_id FROM meals WHERE id = ?`
	if err := tx.QueryRowContext(ctx, mealQ, mealID).Scan(&atUTC, &mealUser); err != nil {
		if err == sql.ErrNoRows {
			return types.ErrNotFound
		}
		return fmt.Errorf("store: get meal: %w", err)
	}
	if mealUser != userID {
		return types.ErrNotFound
	}

	const itemsQ = `
		SELECT rowid, kcal, protein, carbs, fat, fiber
		FROM resolved_items WHERE meal_id = ? ORDER BY rowid
	`
	rows, err := tx.QueryContext(ctx, itemsQ, mealID)
	if err != nil {
		return fmt.Errorf("store: query items: %w", err)
	}
	type row struct {
		rowid int64
		m     types.Macros
	}
	var items []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.rowid, &r.m.Calories, &r.m.Protein, &r.m.Carbs, &r.m.Fat, &r.m.Fiber); err != nil {
			rows.Close()
			return fmt.Errorf("store: scan item: %w", err)
		}
		items = append(items, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: items rows: %w", err)
	}
	if itemIndex < 0 || itemIndex >= len(items) {
		return fmt.Errorf("store: item index %d out of range [0, %d): %w", itemIndex, len(items), types.ErrNotFound)
	}

	target := items[itemIndex]
	if _, err := tx.ExecContext(ctx, `DELETE FROM resolved_items WHERE rowid = ?`, target.rowid); err != nil {
		return fmt.Errorf("store: delete item: %w", err)
	}

	localDate := parseUTC(atUTC).Format("2006-01-02")
	const rollupQ = `
		UPDATE daily_rollups SET
			consumed_kcal    = consumed_kcal    - ?,
			consumed_protein = consumed_protein - ?,
			consumed_carbs   = consumed_carbs   - ?,
			consumed_fat     = consumed_fat     - ?,
			consumed_fiber   = consumed_fiber   - ?
		WHERE user_id = ? AND date = ?
	`
	m := target.m
	if _, err := tx.ExecContext(ctx, rollupQ,
		m.Calories, m.Protein, m.Carbs, m.Fat, m.Fiber, userID, localDate,
	); err != nil {
		return fmt.Errorf("store: update rollup: %w", err)
	}

	return tx.Commit()
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

// UpdateRollupTargets writes the target columns of a day's rollup (creating the
// row with zero consumption if absent) so a targets change shows immediately on
// the dashboard, which reads targets from the rollup.
func (s *Store) UpdateRollupTargets(ctx context.Context, userID, localDate string, t types.Macros) error {
	const q = `
		INSERT INTO daily_rollups
			(user_id, date,
			 consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
			 target_kcal, target_protein, target_carbs, target_fat, target_fiber)
		VALUES (?, ?, 0, 0, 0, 0, 0, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, date) DO UPDATE SET
			target_kcal    = ?,
			target_protein = ?,
			target_carbs   = ?,
			target_fat     = ?,
			target_fiber   = ?
	`
	_, err := s.db.ExecContext(ctx, q, userID, localDate,
		t.Calories, t.Protein, t.Carbs, t.Fat, t.Fiber,
		t.Calories, t.Protein, t.Carbs, t.Fat, t.Fiber,
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

// ---------------------------------------------------------------------------
// Phase 1 — Latest Meal
// ---------------------------------------------------------------------------

// LatestMealTime returns the most recent meal timestamp for a user, or
// types.ErrNotFound when no meals exist.
func (s *Store) LatestMealTime(ctx context.Context, userID string) (string, error) {
	const q = `SELECT at_utc FROM meals WHERE user_id = ? ORDER BY created_at DESC LIMIT 1`
	row := s.db.QueryRowContext(ctx, q, userID)
	var at string
	if err := row.Scan(&at); err == sql.ErrNoRows {
		return "", types.ErrNotFound
	} else if err != nil {
		return "", fmt.Errorf("store: latest meal time: %w", err)
	}
	return at, nil
}

// ---------------------------------------------------------------------------
// Phase 2 — Food Discovery
// ---------------------------------------------------------------------------

// ListFoods returns paginated food library entries, optionally filtered by source.
func (s *Store) ListFoods(ctx context.Context, userID, source string, limit, offset int) ([]types.FoodDetail, error) {
	args := []any{userID}
	q := `SELECT food_id, user_id, name, source, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g,
	       category, brand, barcode, image_url, serving_size, serving_unit, query_count, last_used
		FROM food_library WHERE user_id = ?`
	if source != "" {
		q += ` AND source = ?`
		args = append(args, source)
	}
	q += ` ORDER BY last_used DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list foods: %w", err)
	}
	defer rows.Close()
	return scanFoodDetails(rows)
}

// SearchFoods searches food_library by name and alias using FTS5 full-text search.
func (s *Store) SearchFoods(ctx context.Context, userID, query string) ([]types.FoodDetail, error) {
	// Build FTS5 prefix query: each token gets a * suffix.
	tokens := strings.Fields(query)
	ftsQuery := strings.Join(tokens, "* ") + "*"

	const q = `
		SELECT fl.food_id, fl.user_id, fl.name, fl.source,
		       fl.kcal_100g, fl.protein_100g, fl.carbs_100g, fl.fat_100g, fl.fiber_100g,
		       fl.category, fl.brand, fl.barcode, fl.image_url, fl.serving_size, fl.serving_unit,
		       fl.query_count, fl.last_used
		FROM food_library fl
		INNER JOIN food_search fs ON fs.food_id = fl.food_id AND fs.user_id = fl.user_id
		WHERE food_search MATCH ? AND fl.user_id = ?
		ORDER BY fl.query_count DESC
		LIMIT 20
	`
	rows, err := s.db.QueryContext(ctx, q, ftsQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("store: search foods: %w", err)
	}
	defer rows.Close()
	return scanFoodDetails(rows)
}

// FrequentFoods returns the most frequently logged foods.
func (s *Store) FrequentFoods(ctx context.Context, userID string, limit int) ([]types.FoodDetail, error) {
	const q = `
		SELECT food_id, user_id, name, source,
		       kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g,
		       category, brand, barcode, image_url, serving_size, serving_unit,
		       query_count, last_used
		FROM food_library
		WHERE user_id = ? AND query_count > 0
		ORDER BY query_count DESC, last_used DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("store: frequent foods: %w", err)
	}
	defer rows.Close()
	return scanFoodDetails(rows)
}

// GetFoodDetail returns a single food with its aliases.
func (s *Store) GetFoodDetail(ctx context.Context, userID, foodID string) (types.FoodDetail, error) {
	const foodQ = `
		SELECT food_id, user_id, name, source,
		       kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g,
		       category, brand, barcode, image_url, serving_size, serving_unit,
		       query_count, last_used
		FROM food_library
		WHERE user_id = ? AND food_id = ?
	`
	row := s.db.QueryRowContext(ctx, foodQ, userID, foodID)
	fd, err := scanFoodDetail(row)
	if err == sql.ErrNoRows {
		return types.FoodDetail{}, types.ErrNotFound
	}
	if err != nil {
		return types.FoodDetail{}, fmt.Errorf("store: get food detail: %w", err)
	}

	// Load aliases separately.
	const aliasQ = `
		SELECT food_id, alias_normalized FROM food_aliases
		WHERE user_id = ? AND food_id = ?
	`
	arows, err := s.db.QueryContext(ctx, aliasQ, userID, foodID)
	if err != nil {
		return types.FoodDetail{}, fmt.Errorf("store: get food aliases: %w", err)
	}
	defer arows.Close()
	for arows.Next() {
		var a types.FoodAlias
		if err := arows.Scan(&a.FoodID, &a.Normalized); err != nil {
			return types.FoodDetail{}, fmt.Errorf("store: scan alias: %w", err)
		}
		fd.Aliases = append(fd.Aliases, a)
	}
	return fd, arows.Err()
}

// AddFoodAlias inserts a normalized alias for a food.
func (s *Store) AddFoodAlias(ctx context.Context, userID, foodID, alias string) error {
	normalized := normalize.Normalize(alias)
	const q = `INSERT OR IGNORE INTO food_aliases (user_id, alias_normalized, food_id) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, userID, normalized, foodID)
	return err
}

// DeleteFoodAlias removes a normalized alias for a food. Returns ErrNotFound if
// no row was deleted.
func (s *Store) DeleteFoodAlias(ctx context.Context, userID, foodID, alias string) error {
	normalized := normalize.Normalize(alias)
	const q = `DELETE FROM food_aliases WHERE user_id = ? AND food_id = ? AND alias_normalized = ?`
	res, err := s.db.ExecContext(ctx, q, userID, foodID, normalized)
	if err != nil {
		return fmt.Errorf("store: delete food alias: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func scanFoodDetail(row *sql.Row) (types.FoodDetail, error) {
	var fd types.FoodDetail
	var lastUsed string
	err := row.Scan(&fd.FoodID, &fd.UserID, &fd.Name, &fd.Source,
		&fd.Per100g.Calories, &fd.Per100g.Protein, &fd.Per100g.Carbs, &fd.Per100g.Fat, &fd.Per100g.Fiber,
		&fd.Category, &fd.Brand, &fd.Barcode, &fd.ImageURL, &fd.ServingSize, &fd.ServingUnit,
		&fd.QueryCount, &lastUsed,
	)
	if err != nil {
		return types.FoodDetail{}, err
	}
	fd.LastUsed = lastUsed
	fd.Aliases = []types.FoodAlias{}
	return fd, nil
}

func scanFoodDetails(rows *sql.Rows) ([]types.FoodDetail, error) {
	var out []types.FoodDetail
	for rows.Next() {
		var fd types.FoodDetail
		var lastUsed string
		if err := rows.Scan(&fd.FoodID, &fd.UserID, &fd.Name, &fd.Source,
			&fd.Per100g.Calories, &fd.Per100g.Protein, &fd.Per100g.Carbs, &fd.Per100g.Fat, &fd.Per100g.Fiber,
			&fd.Category, &fd.Brand, &fd.Barcode, &fd.ImageURL, &fd.ServingSize, &fd.ServingUnit,
			&fd.QueryCount, &lastUsed,
		); err != nil {
			return nil, fmt.Errorf("store: scan food detail: %w", err)
		}
		fd.LastUsed = lastUsed
		fd.Aliases = []types.FoodAlias{}
		out = append(out, fd)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Phase 3 — Meal Templates
// ---------------------------------------------------------------------------

// SaveTemplate inserts or upserts a meal template.
func (s *Store) SaveTemplate(ctx context.Context, t types.MealTemplate) error {
	itemsJSON, err := json.Marshal(t.Items)
	if err != nil {
		return fmt.Errorf("store: marshal template items: %w", err)
	}
	const q = `
		INSERT INTO meal_templates (id, user_id, name, items_json, created_at, last_used)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name       = excluded.name,
			items_json = excluded.items_json,
			last_used  = excluded.last_used
	`
	_, err = s.db.ExecContext(ctx, q, t.ID, t.UserID, t.Name, string(itemsJSON), utcStr(t.CreatedAt), utcStr(t.LastUsed))
	return err
}

// GetTemplates returns all templates for a user, newest first.
func (s *Store) GetTemplates(ctx context.Context, userID string) ([]types.MealTemplate, error) {
	const q = `
		SELECT id, user_id, name, items_json, created_at, last_used
		FROM meal_templates WHERE user_id = ?
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("store: get templates: %w", err)
	}
	defer rows.Close()
	return scanTemplates(rows)
}

// GetTemplate returns a single template by ID.
func (s *Store) GetTemplate(ctx context.Context, templateID string) (types.MealTemplate, error) {
	const q = `
		SELECT id, user_id, name, items_json, created_at, last_used
		FROM meal_templates WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, q, templateID)
	t, err := scanTemplate(row)
	if err == sql.ErrNoRows {
		return types.MealTemplate{}, types.ErrNotFound
	}
	return t, err
}

// DeleteTemplate deletes a template by user + ID. Returns ErrNotFound if 0 rows.
func (s *Store) DeleteTemplate(ctx context.Context, userID, templateID string) error {
	const q = `DELETE FROM meal_templates WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, q, templateID, userID)
	if err != nil {
		return fmt.Errorf("store: delete template: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// LogTemplateUse records a template usage event.
func (s *Store) LogTemplateUse(ctx context.Context, tl types.TemplateLog) error {
	const q = `INSERT INTO template_logs (id, user_id, template_id, logged_at) VALUES (?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, tl.ID, tl.UserID, tl.TemplateID, utcStr(tl.LoggedAt))
	return err
}

func scanTemplate(row *sql.Row) (types.MealTemplate, error) {
	var t types.MealTemplate
	var itemsJSON, ca, lu string
	if err := row.Scan(&t.ID, &t.UserID, &t.Name, &itemsJSON, &ca, &lu); err != nil {
		return types.MealTemplate{}, err
	}
	if err := json.Unmarshal([]byte(itemsJSON), &t.Items); err != nil {
		return types.MealTemplate{}, fmt.Errorf("store: unmarshal template items: %w", err)
	}
	t.CreatedAt = parseUTC(ca)
	t.LastUsed = parseUTC(lu)
	if t.Items == nil {
		t.Items = []types.ResolvedItem{}
	}
	return t, nil
}

func scanTemplates(rows *sql.Rows) ([]types.MealTemplate, error) {
	var out []types.MealTemplate
	for rows.Next() {
		var t types.MealTemplate
		var itemsJSON, ca, lu string
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &itemsJSON, &ca, &lu); err != nil {
			return nil, fmt.Errorf("store: scan template: %w", err)
		}
		if err := json.Unmarshal([]byte(itemsJSON), &t.Items); err != nil {
			return nil, fmt.Errorf("store: unmarshal template items: %w", err)
		}
		t.CreatedAt = parseUTC(ca)
		t.LastUsed = parseUTC(lu)
		if t.Items == nil {
			t.Items = []types.ResolvedItem{}
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Phase 4 — Body Tracking
// ---------------------------------------------------------------------------

// ListWeight returns weight entries for the last N days.
func (s *Store) ListWeight(ctx context.Context, userID string, days int) ([]types.WeightEntry, error) {
	const q = `
		SELECT id, user_id, date, weight_kg, note, created_at
		FROM weight_log
		WHERE user_id = ? AND date >= date('now', ? || ' days', 'localtime')
		ORDER BY date ASC
	`
	rows, err := s.db.QueryContext(ctx, q, userID, fmt.Sprintf("-%d", days))
	if err != nil {
		return nil, fmt.Errorf("store: list weight: %w", err)
	}
	defer rows.Close()
	return scanWeightEntries(rows)
}

// LogWeight inserts or updates a weight entry.
func (s *Store) LogWeight(ctx context.Context, w types.WeightEntry) error {
	const q = `
		INSERT INTO weight_log (id, user_id, date, weight_kg, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			date      = excluded.date,
			weight_kg = excluded.weight_kg,
			note      = excluded.note
	`
	_, err := s.db.ExecContext(ctx, q, w.ID, w.UserID, w.Date, w.WeightKg, w.Note, utcStr(w.CreatedAt))
	return err
}

// DeleteWeight deletes a weight entry by user + ID. Returns ErrNotFound if absent.
func (s *Store) DeleteWeight(ctx context.Context, userID, entryID string) error {
	const q = `DELETE FROM weight_log WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, q, entryID, userID)
	if err != nil {
		return fmt.Errorf("store: delete weight: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// WeightTrend returns weight entries with 7-day rolling average for the last N days.
func (s *Store) WeightTrend(ctx context.Context, userID string, days int) ([]types.WeightTrend, error) {
	entries, err := s.ListWeight(ctx, userID, days)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return []types.WeightTrend{}, nil
	}

	var trend []types.WeightTrend
	for i, e := range entries {
		wt := types.WeightTrend{Date: e.Date, WeightKg: e.WeightKg}

		// 7-day rolling average.
		start := i - 6
		if start < 0 {
			start = 0
		}
		sum := 0.0
		count := 0
		for j := start; j <= i; j++ {
			sum += entries[j].WeightKg
			count++
		}
		if count > 0 {
			wt.RollingAvg = sum / float64(count)
		}
		trend = append(trend, wt)
	}
	return trend, nil
}

func scanWeightEntries(rows *sql.Rows) ([]types.WeightEntry, error) {
	var out []types.WeightEntry
	for rows.Next() {
		var w types.WeightEntry
		var ca string
		if err := rows.Scan(&w.ID, &w.UserID, &w.Date, &w.WeightKg, &w.Note, &ca); err != nil {
			return nil, fmt.Errorf("store: scan weight: %w", err)
		}
		w.CreatedAt = parseUTC(ca)
		out = append(out, w)
	}
	return out, rows.Err()
}

// ListMeasurements returns measurement entries for the last N days.
func (s *Store) ListMeasurements(ctx context.Context, userID string, days int) ([]types.MeasurementEntry, error) {
	const q = `
		SELECT id, user_id, date, waist_cm, hips_cm, chest_cm, left_arm_cm, right_arm_cm,
		       left_thigh_cm, right_thigh_cm, note, created_at
		FROM measurement_log
		WHERE user_id = ? AND date >= date('now', ? || ' days', 'localtime')
		ORDER BY date ASC
	`
	rows, err := s.db.QueryContext(ctx, q, userID, fmt.Sprintf("-%d", days))
	if err != nil {
		return nil, fmt.Errorf("store: list measurements: %w", err)
	}
	defer rows.Close()
	return scanMeasurementEntries(rows)
}

// LogMeasurement inserts or updates a measurement entry.
func (s *Store) LogMeasurement(ctx context.Context, m types.MeasurementEntry) error {
	const q = `
		INSERT INTO measurement_log
			(id, user_id, date, waist_cm, hips_cm, chest_cm, left_arm_cm, right_arm_cm,
			 left_thigh_cm, right_thigh_cm, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			date           = excluded.date,
			waist_cm       = excluded.waist_cm,
			hips_cm        = excluded.hips_cm,
			chest_cm       = excluded.chest_cm,
			left_arm_cm    = excluded.left_arm_cm,
			right_arm_cm   = excluded.right_arm_cm,
			left_thigh_cm  = excluded.left_thigh_cm,
			right_thigh_cm = excluded.right_thigh_cm,
			note           = excluded.note
	`
	_, err := s.db.ExecContext(ctx, q,
		m.ID, m.UserID, m.Date,
		m.WaistCm, m.HipsCm, m.ChestCm,
		m.LeftArmCm, m.RightArmCm, m.LeftThighCm, m.RightThighCm,
		m.Note, utcStr(m.CreatedAt),
	)
	return err
}

// DeleteMeasurement deletes a measurement entry by user + ID. Returns ErrNotFound.
func (s *Store) DeleteMeasurement(ctx context.Context, userID, entryID string) error {
	const q = `DELETE FROM measurement_log WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, q, entryID, userID)
	if err != nil {
		return fmt.Errorf("store: delete measurement: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

func scanMeasurementEntries(rows *sql.Rows) ([]types.MeasurementEntry, error) {
	var out []types.MeasurementEntry
	for rows.Next() {
		var m types.MeasurementEntry
		var ca string
		if err := rows.Scan(&m.ID, &m.UserID, &m.Date,
			&m.WaistCm, &m.HipsCm, &m.ChestCm,
			&m.LeftArmCm, &m.RightArmCm, &m.LeftThighCm, &m.RightThighCm,
			&m.Note, &ca,
		); err != nil {
			return nil, fmt.Errorf("store: scan measurement: %w", err)
		}
		m.CreatedAt = parseUTC(ca)
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListPhotoMetadata returns progress photo records without the BLOB data.
func (s *Store) ListPhotoMetadata(ctx context.Context, userID string) ([]types.ProgressPhoto, error) {
	const q = `
		SELECT id, user_id, date, view, mime_type, created_at
		FROM progress_photos WHERE user_id = ?
		ORDER BY date DESC
	`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("store: list photo metadata: %w", err)
	}
	defer rows.Close()

	var out []types.ProgressPhoto
	for rows.Next() {
		var p types.ProgressPhoto
		var ca string
		if err := rows.Scan(&p.ID, &p.UserID, &p.Date, &p.View, &p.MimeType, &ca); err != nil {
			return nil, fmt.Errorf("store: scan photo meta: %w", err)
		}
		p.CreatedAt = parseUTC(ca)
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPhotoData returns a single progress photo including BLOB data.
func (s *Store) GetPhotoData(ctx context.Context, photoID string) (types.ProgressPhoto, error) {
	const q = `
		SELECT id, user_id, date, view, mime_type, data, created_at
		FROM progress_photos WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, q, photoID)
	var p types.ProgressPhoto
	var ca string
	if err := row.Scan(&p.ID, &p.UserID, &p.Date, &p.View, &p.MimeType, &p.Data, &ca); err == sql.ErrNoRows {
		return types.ProgressPhoto{}, types.ErrNotFound
	} else if err != nil {
		return types.ProgressPhoto{}, fmt.Errorf("store: get photo data: %w", err)
	}
	p.CreatedAt = parseUTC(ca)
	return p, nil
}

// UploadPhoto inserts a progress photo with BLOB data.
func (s *Store) UploadPhoto(ctx context.Context, p types.ProgressPhoto) error {
	const q = `
		INSERT INTO progress_photos (id, user_id, date, view, mime_type, data, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, q, p.ID, p.UserID, p.Date, p.View, p.MimeType, p.Data, utcStr(p.CreatedAt))
	return err
}

// DeletePhoto deletes a progress photo by user + ID. Returns ErrNotFound if absent.
func (s *Store) DeletePhoto(ctx context.Context, userID, photoID string) error {
	const q = `DELETE FROM progress_photos WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, q, photoID, userID)
	if err != nil {
		return fmt.Errorf("store: delete photo: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// GetMealsInRange returns meals for a user within a date range (inclusive).
func (s *Store) GetMealsInRange(ctx context.Context, userID, startDate, endDate string) ([]types.Meal, error) {
	const mealQ = `
		SELECT id, user_id, at_utc, raw_text, confidence, parser_tier, created_at
		FROM meals
		WHERE user_id = ? AND date(at_utc) >= ? AND date(at_utc) <= ?
		ORDER BY at_utc ASC
	`
	rows, err := s.db.QueryContext(ctx, mealQ, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("store: query meals in range: %w", err)
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
		return nil, fmt.Errorf("store: meals range rows: %w", err)
	}

	if len(meals) == 0 {
		return []types.Meal{}, nil
	}

	itemsByMeal, err := s.loadItems(ctx, mealIDs)
	if err != nil {
		return nil, err
	}
	for i := range meals {
		meals[i].Items = itemsByMeal[meals[i].ID]
	}
	return meals, nil
}

// ---------------------------------------------------------------------------
// Phase 5 — Goals & Profile
// ---------------------------------------------------------------------------

// GetProfile returns the user profile, or ErrNotFound.
func (s *Store) GetProfile(ctx context.Context, userID string) (types.UserProfile, error) {
	const q = `
		SELECT user_id, height_cm, birth_date, gender, activity_level, goal,
		       target_weight_kg, weekly_rate, onboarded, created_at, updated_at
		FROM user_profiles WHERE user_id = ?
	`
	row := s.db.QueryRowContext(ctx, q, userID)
	var p types.UserProfile
	var bd, ca, ua string
	var onboarded int
	if err := row.Scan(&p.UserID, &p.HeightCm, &bd, &p.Gender, &p.ActivityLevel, &p.Goal,
		&p.TargetWeightKg, &p.WeeklyRate, &onboarded, &ca, &ua,
	); err == sql.ErrNoRows {
		return types.UserProfile{}, types.ErrNotFound
	} else if err != nil {
		return types.UserProfile{}, fmt.Errorf("store: get profile: %w", err)
	}
	p.BirthDate = bd
	p.Onboarded = onboarded != 0
	p.CreatedAt = parseUTC(ca)
	p.UpdatedAt = parseUTC(ua)
	return p, nil
}

// UpsertProfile inserts or updates the user profile.
func (s *Store) UpsertProfile(ctx context.Context, p types.UserProfile) error {
	onboarded := 0
	if p.Onboarded {
		onboarded = 1
	}
	const q = `
		INSERT INTO user_profiles
			(user_id, height_cm, birth_date, gender, activity_level, goal,
			 target_weight_kg, weekly_rate, onboarded, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			height_cm        = excluded.height_cm,
			birth_date       = excluded.birth_date,
			gender           = excluded.gender,
			activity_level   = excluded.activity_level,
			goal             = excluded.goal,
			target_weight_kg = excluded.target_weight_kg,
			weekly_rate      = excluded.weekly_rate,
			onboarded        = excluded.onboarded,
			updated_at       = excluded.updated_at
	`
	_, err := s.db.ExecContext(ctx, q,
		p.UserID, p.HeightCm, p.BirthDate, p.Gender, p.ActivityLevel, p.Goal,
		p.TargetWeightKg, p.WeeklyRate, onboarded,
		utcStr(p.CreatedAt), utcStr(p.UpdatedAt),
	)
	return err
}
