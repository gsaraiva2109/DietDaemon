// Package store implements ports.Store. Supports SQLite (modernc.org/sqlite, pure
// Go, CGO-free) and Postgres (lib/pq) via a Dialect abstraction.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/backup"
	"github.com/gsaraiva2109/dietdaemon/internal/normalize"
	"github.com/gsaraiva2109/dietdaemon/internal/scheduler"
	"github.com/gsaraiva2109/dietdaemon/migrations"
)

// Store implements ports.Store backed by SQLite or Postgres.
type Store struct {
	db      *sqlx.DB
	dialect Dialect
	driver  string // "sqlite" or "postgres"
}

// Compile-time guarantees that Store satisfies every interface boundary it must.
var (
	_ ports.Store                 = (*Store)(nil)
	_ scheduler.Store             = (*Store)(nil)
	_ scheduler.NudgeStore        = (*Store)(nil)
	_ scheduler.RuleConfigStore   = (*Store)(nil)
	_ scheduler.DigestStore       = (*Store)(nil)
	_ scheduler.ChatRouteStore    = (*Store)(nil)
	_ scheduler.SentNudgeStore    = (*Store)(nil)
	_ scheduler.WeeklyBudgetStore = (*Store)(nil)
	_ backup.Store                = (*Store)(nil)
)

// New opens a database, applies driver-specific setup, runs migrations, and
// returns a ready Store.
//
// driver is "sqlite" or "postgres". dsn is the file path for SQLite or a
// connection URL for Postgres (e.g. "postgres://user:pass@host/db?sslmode=disable").
func New(driver, dsn string, dialect Dialect) (*Store, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open db: %w", err)
	}

	switch driver {
	case "sqlite":
		// SQLite is single-writer; a pool of 1 guarantees every PRAGMA and all
		// subsequent queries share the same connection, avoiding SQLITE_CANTOPEN
		// when a second connection races the EXCLUSIVE lock below.
		db.SetMaxOpenConns(1)

		// EXCLUSIVE locking before WAL avoids shared-memory (-shm) entirely.
		// SQLite keeps the wal-index in heap memory instead of mmap'ing a file.
		// Required for Docker Swarm / some overlay filesystems where the VFS
		// shared-memory primitives (xShmMap, xShmLock, etc.) don't work.
		if _, err := db.Exec("PRAGMA locking_mode = EXCLUSIVE"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("store: exclusive lock: %w", err)
		}
		if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("store: enable WAL: %w", err)
		}
		if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("store: enable foreign keys: %w", err)
		}
	case "postgres":
		// Postgres manages its own connection pool; 25 is a sensible default
		// for modest deployments. The operator can tune via PG* env vars or
		// connection URL parameters.
		db.SetMaxOpenConns(25)
	}

	s := &Store{db: sqlx.NewDb(db, driver), dialect: dialect, driver: driver}
	if err := s.runMigrations(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: migrate: %w", err)
	}

	return s, nil
}

// rewrite applies dialect-specific placeholder conversion to a SQL query.
// For SQLite this is a no-op; for Postgres it replaces ? with $1, $2, ...
func (s *Store) rewrite(sql string) string {
	return s.dialect.RewritePlaceholders(sql)
}

func (s *Store) runMigrations() error {
	if _, err := s.db.Exec(s.rewrite(`CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (` + s.dialect.Now() + `))`)); err != nil {
		return fmt.Errorf("init migration tracking: %w", err)
	}

	entries, err := migrations.FS(s.driver).ReadDir(s.driver)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	// Run the migration loop up to twice. A second pass is only needed
	// when a legacy "mark all as applied" bug recorded migrations that
	// never actually ran — the first pass detects the inconsistency,
	// removes the bogus tracking entries, and the second pass applies
	// them for real.
	for pass := range 2 {
		applied := 0
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
				continue
			}

			var already int
			if err := s.db.Get(&already, s.rewrite(`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`), entry.Name()); err != nil {
				return fmt.Errorf("check migration %s: %w", entry.Name(), err)
			}
			if already > 0 {
				continue
			}

			content, err := migrations.FS(s.driver).ReadFile(s.driver + "/" + entry.Name())
			if err != nil {
				return fmt.Errorf("read %s: %w", entry.Name(), err)
			}
			if _, err := s.db.Exec(s.rewrite(string(content))); err != nil {
				// Idempotency: databases that predate migration tracking may
				// already have tables/columns/indexes from manual or older
				// migration paths. Treat "already exists" errors as success
				// so the migration is tracked and skipped on next start.
				if isBenignMigrationErr(err) {
					if _, recErr := s.db.Exec(s.rewrite(`INSERT INTO schema_migrations (name) VALUES (?)`), entry.Name()); recErr != nil {
						return fmt.Errorf("record migration %s after benign error: %w", entry.Name(), recErr)
					}
					continue
				}
				return fmt.Errorf("exec %s: %w", entry.Name(), err)
			}
			if _, err := s.db.Exec(s.rewrite(`INSERT INTO schema_migrations (name) VALUES (?)`), entry.Name()); err != nil {
				return fmt.Errorf("record migration %s: %w", entry.Name(), err)
			}
			applied++
		}

		// Self-heal: detect migrations that were tracked as applied by a
		// buggy legacy path but whose DDL effects are actually missing.
		// If found, delete the bogus entries so the next pass applies them.
		if applied == 0 && pass == 0 {
			if healed := s.healMissingColumns(); healed > 0 {
				continue // re-run loop with cleaned tracking
			}
		}
		break
	}

	return nil
}

// healMissingColumns detects migrations that are tracked as applied in
// schema_migrations but whose key columns never materialised (legacy
// "mark all as applied" bug). It removes the bogus tracking entries so
// the next migration pass runs them for real.
func (s *Store) healMissingColumns() int {
	// healMissingColumns is a SQLite-only legacy fix. Postgres databases don't
	// have the "mark all as applied" bug this addresses.
	if s.driver != "sqlite" {
		return 0
	}
	checks := []struct {
		migration string
		table     string
		column    string
	}{
		// 006_food_metadata adds category, brand, barcode + more to food_library.
		{"006_food_metadata.sql", "food_library", "category"},
		// 008_body_tracking adds weight_log, measurement_log, progress_photos.
		{"008_body_tracking.sql", "weight_log", "id"},
		// 009_user_profile adds the user_profiles table.
		{"009_user_profile.sql", "user_profiles", "user_id"},
		// 011_auth_foundation adds account_id, email, status + more to users.
		{"011_auth_foundation.sql", "users", "account_id"},
		// 012_totp adds totp_secrets, recovery_codes.
		{"012_totp.sql", "totp_secrets", "user_id"},
	}
	healed := 0
	for _, c := range checks {
		var tracked int
		if err := s.db.Get(&tracked, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, c.migration); err != nil {
			continue
		}
		if tracked == 0 {
			continue
		}
		// Check if the column/table actually exists.
		var exists int
		if err := s.db.Get(&exists, `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, c.table, c.column); err != nil {
			// Table might not exist either — that's also a sign of missing migration.
			_, _ = s.db.Exec(`DELETE FROM schema_migrations WHERE name = ?`, c.migration)
			healed++
			continue
		}
		if exists == 0 {
			_, _ = s.db.Exec(`DELETE FROM schema_migrations WHERE name = ?`, c.migration)
			healed++
		}
	}
	return healed
}

// isBenignMigrationErr returns true when err indicates the DDL/DML operation
// was already applied — a duplicate column, table, index, or constraint.
// This lets the migration runner treat pre-existing schema as success instead
// of aborting startup.
func isBenignMigrationErr(err error) bool {
	msg := err.Error()
	// SQLite error patterns for "already exists":
	//   - "duplicate column name: X"
	//   - "table X already exists"
	//   - "index X already exists"
	//   - "trigger X already exists"
	//   - "UNIQUE constraint failed: schema_migrations.name" (harmless)
	return strings.Contains(msg, "duplicate column name") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "UNIQUE constraint failed")
}

// DB returns the underlying *sql.DB so that callers (e.g. the embedding index)
// can operate on the same database connection without opening a second one.
func (s *Store) DB() *sql.DB { return s.db.DB }

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

// UpsertUser inserts or updates a user row. New auth columns are set via
// separate auth-dedicated methods (CreateUserWithPassword); this method
// preserves the existing id/timezone/created_at contract for the pipeline.
func (s *Store) UpsertUser(ctx context.Context, u types.User) error {
	const q = `
		INSERT INTO users (id, account_id, email, email_verified_at, status, display_name, timezone, created_at)
		VALUES (:id, :account_id, :email, :email_verified_at, :status, :display_name, :timezone, :created_at)
		ON CONFLICT(id) DO UPDATE SET timezone = excluded.timezone
	`
	var emailVerifiedAt any
	if u.EmailVerifiedAt != nil {
		emailVerifiedAt = utcStr(*u.EmailVerifiedAt)
	}
	query, args, err := sqlx.Named(q, map[string]any{
		"id":                u.ID,
		"account_id":        nullStr(u.AccountID),
		"email":             nullStr(u.Email),
		"email_verified_at": emailVerifiedAt,
		"status":            u.Status,
		"display_name":      nullStr(u.DisplayName),
		"timezone":          u.Timezone,
		"created_at":        utcStr(u.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("store: bind upsert user: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}

// GetUser returns the user or types.ErrNotFound.
func (s *Store) GetUser(ctx context.Context, userID string) (types.User, error) {
	const q = `SELECT id, account_id, email, email_verified_at, status, display_name, timezone, created_at, webauthn_handle FROM users WHERE id = ?`
	var row userRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.User{}, types.ErrNotFound
		}
		return types.User{}, err
	}
	return row.toUser(), nil
}

// ListUsers returns every user. Empty slice, nil error when there are none.
func (s *Store) ListUsers(ctx context.Context) ([]types.User, error) {
	const q = `SELECT id, account_id, email, email_verified_at, status, display_name, timezone, created_at, webauthn_handle FROM users ORDER BY id`
	var rows []userRow
	if err := s.db.SelectContext(ctx, &rows, q); err != nil {
		return nil, fmt.Errorf("store: list users: %w", err)
	}
	var users []types.User
	for _, r := range rows {
		users = append(users, r.toUser())
	}
	return users, nil
}

// userRow is the flat DB shape of the users table; the public types.User
// nests EmailVerifiedAt as *time.Time and applies a default status, neither
// of which maps 1:1 onto a column.
type userRow struct {
	ID              string         `db:"id"`
	AccountID       sql.NullString `db:"account_id"`
	Email           sql.NullString `db:"email"`
	EmailVerifiedAt sql.NullString `db:"email_verified_at"`
	Status          sql.NullString `db:"status"`
	DisplayName     sql.NullString `db:"display_name"`
	Timezone        string         `db:"timezone"`
	CreatedAt       string         `db:"created_at"`
	WebAuthnHandle  sql.NullString `db:"webauthn_handle"`
}

func (r userRow) toUser() types.User {
	u := types.User{
		ID:             r.ID,
		AccountID:      r.AccountID.String,
		Email:          r.Email.String,
		DisplayName:    r.DisplayName.String,
		Status:         r.Status.String,
		Timezone:       r.Timezone,
		CreatedAt:      parseUTC(r.CreatedAt),
		WebAuthnHandle: r.WebAuthnHandle.String,
	}
	if r.EmailVerifiedAt.Valid {
		u.EmailVerifiedAt = new(parseUTC(r.EmailVerifiedAt.String))
	}
	if !r.Status.Valid {
		u.Status = "active"
	}
	return u
}

// scanUser scans a single *sql.Row with the same column order as userRow.
// Kept for store_auth.go call sites (CreateUserWithPassword, GetUserByEmail,
// GetUserByAPIKey, GetSession-adjacent lookups) that build the row via a
// custom SELECT elsewhere in that file.
func scanUser(row *sql.Row) (types.User, error) {
	var r userRow
	if err := row.Scan(&r.ID, &r.AccountID, &r.Email, &r.EmailVerifiedAt, &r.Status, &r.DisplayName, &r.Timezone, &r.CreatedAt, &r.WebAuthnHandle); err != nil {
		return types.User{}, err
	}
	return r.toUser(), nil
}

// ValidateToken looks up a Bearer token in the api_tokens table and returns the
// owning userID. Returns types.ErrNotFound when the token is invalid or expired.
// In single-user mode this method is not called; the static API_AUTH_TOKEN is
// checked directly.

// UpsertUserTimezone updates the users.timezone column for a user.
func (s *Store) UpsertUserTimezone(ctx context.Context, userID, timezone string) error {
	const q = `UPDATE users SET timezone = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), timezone, userID)
	return err
}

// MapChannelUser inserts a mapping from a messaging channel + channel_user_id
// to an internal user_id. It is idempotent (INSERT OR IGNORE).
func (s *Store) MapChannelUser(ctx context.Context, channel, channelUserID, userID string) error {
	const q = `
		INSERT INTO user_channels (channel, channel_user_id, user_id)
		VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), channel, channelUserID, userID)
	return err
}

// GetUserIDByChannel returns the internal user_id for a given
// (channel, channel_user_id) pair. Returns types.ErrNotFound when no mapping
// exists.
func (s *Store) GetUserIDByChannel(ctx context.Context, channel, channelUserID string) (string, error) {
	const q = `SELECT user_id FROM user_channels WHERE channel = ? AND channel_user_id = ?`
	var userID string
	if err := s.db.GetContext(ctx, &userID, s.rewrite(q), channel, channelUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", types.ErrNotFound
		}
		return "", fmt.Errorf("store: get user by channel: %w", err)
	}
	return userID, nil
}

// UpsertChatRoute records the chat metadata needed to reach a user
// proactively (e.g. from the scheduler), refreshed on every inbound message.
// One row per (user, channel).
func (s *Store) UpsertChatRoute(ctx context.Context, userID, channel string, meta map[string]string) error {
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("store: marshal chat route meta: %w", err)
	}
	const q = `
		INSERT INTO chat_routes (user_id, channel, meta_json, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, channel) DO UPDATE SET
			meta_json  = excluded.meta_json,
			updated_at = excluded.updated_at
	`
	_, err = s.db.ExecContext(ctx, s.rewrite(q), userID, channel, string(metaJSON), utcNow())
	if err != nil {
		return fmt.Errorf("store: upsert chat route: %w", err)
	}
	return nil
}

// GetChatRoute returns the most recently seen channel + delivery metadata for
// a user, so the scheduler can send a message through a MessagingAdapter
// instead of only the plain-text Notifier. Returns types.ErrNotFound when the
// user has never been seen on any channel.
func (s *Store) GetChatRoute(ctx context.Context, userID string) (string, map[string]string, error) {
	const q = `SELECT channel, meta_json FROM chat_routes WHERE user_id = ? ORDER BY updated_at DESC LIMIT 1`
	var row struct {
		Channel  string `db:"channel"`
		MetaJSON string `db:"meta_json"`
	}
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil, types.ErrNotFound
		}
		return "", nil, fmt.Errorf("store: get chat route: %w", err)
	}
	var meta map[string]string
	if err := json.Unmarshal([]byte(row.MetaJSON), &meta); err != nil {
		return "", nil, fmt.Errorf("store: unmarshal chat route meta: %w", err)
	}
	return row.Channel, meta, nil
}

// ---------------------------------------------------------------------------
// Meals
// ---------------------------------------------------------------------------

// SaveMeal inserts a meal and all its resolved items inside a transaction.
func (s *Store) SaveMeal(ctx context.Context, m types.Meal) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const mealQ = `
		INSERT INTO meals (id, user_id, at_utc, raw_text, confidence, parser_tier, created_at)
		VALUES (:id, :user_id, :at_utc, :raw_text, :confidence, :parser_tier, :created_at)
	`
	mealQuery, mealArgs, err := sqlx.Named(mealQ, map[string]any{
		"id":          m.ID,
		"user_id":     m.UserID,
		"at_utc":      utcStr(m.At),
		"raw_text":    m.RawText,
		"confidence":  m.Confidence,
		"parser_tier": int(m.ParserTier),
		"created_at":  utcStr(m.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("store: bind meal: %w", err)
	}
	if _, err = tx.ExecContext(ctx, s.rewrite(mealQuery), mealArgs...); err != nil {
		return fmt.Errorf("store: insert meal: %w", err)
	}

	const itemQ = `
		INSERT INTO resolved_items
			(id, meal_id, raw_phrase, quantity, unit, normalized_grams,
			 food_id, food_name, source, match_score,
			 kcal, protein, carbs, fat, fiber)
		VALUES (:id, :meal_id, :raw_phrase, :quantity, :unit, :normalized_grams,
			:food_id, :food_name, :source, :match_score,
			:kcal, :protein, :carbs, :fat, :fiber)
	`
	for _, it := range m.Items {
		itemQuery, itemArgs, err := sqlx.Named(itemQ, resolvedItemNamedArgs(newID(), m.ID, it))
		if err != nil {
			return fmt.Errorf("store: bind resolved_item: %w", err)
		}
		if _, err := tx.ExecContext(ctx, s.rewrite(itemQuery), itemArgs...); err != nil {
			return fmt.Errorf("store: insert resolved_item: %w", err)
		}
	}

	return tx.Commit()
}

// resolvedItemNamedArgs builds the named-parameter map shared by every insert
// of a resolved_items row (SaveMeal, AddMealItem).
func resolvedItemNamedArgs(id, mealID string, it types.ResolvedItem) map[string]any {
	return map[string]any{
		"id":               id,
		"meal_id":          mealID,
		"raw_phrase":       it.Parsed.RawPhrase,
		"quantity":         it.Parsed.Quantity,
		"unit":             it.Parsed.Unit,
		"normalized_grams": it.Parsed.NormalizedGrams,
		"food_id":          it.Match.FoodID,
		"food_name":        it.Match.Name,
		"source":           it.Match.Source,
		"match_score":      it.Match.MatchScore,
		"kcal":             it.Macros.Calories,
		"protein":          it.Macros.Protein,
		"carbs":            it.Macros.Carbs,
		"fat":              it.Macros.Fat,
		"fiber":            it.Macros.Fiber,
	}
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
	var rows []mealRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(mealQ), userID, limit); err != nil {
		return nil, fmt.Errorf("store: query meals: %w", err)
	}

	var meals []types.Meal
	var mealIDs []string
	for _, r := range rows {
		m := r.toMeal()
		meals = append(meals, m)
		mealIDs = append(mealIDs, m.ID)
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
	var row mealRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), mealID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Meal{}, types.ErrNotFound
		}
		return types.Meal{}, fmt.Errorf("store: get meal: %w", err)
	}
	m := row.toMeal()

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
	var rows []rollupRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("store: query rollups: %w", err)
	}
	out := make([]types.DailyRollup, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toRollup())
	}
	return out, nil
}

// mealRow is the flat DB shape of the meals table; types.Meal additionally
// carries a non-column Items slice populated by loadItems.
type mealRow struct {
	ID         string  `db:"id"`
	UserID     string  `db:"user_id"`
	AtUTC      string  `db:"at_utc"`
	RawText    string  `db:"raw_text"`
	Confidence float64 `db:"confidence"`
	ParserTier int     `db:"parser_tier"`
	CreatedAt  string  `db:"created_at"`
}

func (r mealRow) toMeal() types.Meal {
	return types.Meal{
		ID:         r.ID,
		UserID:     r.UserID,
		At:         parseUTC(r.AtUTC),
		RawText:    r.RawText,
		Confidence: r.Confidence,
		ParserTier: types.ParserTier(r.ParserTier),
		CreatedAt:  parseUTC(r.CreatedAt),
	}
}

// rollupRow is the flat DB shape of daily_rollups; types.DailyRollup groups
// the consumed/target columns into nested Macros structs.
type rollupRow struct {
	UserID          string  `db:"user_id"`
	Date            string  `db:"date"`
	ConsumedKcal    float64 `db:"consumed_kcal"`
	ConsumedProtein float64 `db:"consumed_protein"`
	ConsumedCarbs   float64 `db:"consumed_carbs"`
	ConsumedFat     float64 `db:"consumed_fat"`
	ConsumedFiber   float64 `db:"consumed_fiber"`
	TargetKcal      float64 `db:"target_kcal"`
	TargetProtein   float64 `db:"target_protein"`
	TargetCarbs     float64 `db:"target_carbs"`
	TargetFat       float64 `db:"target_fat"`
	TargetFiber     float64 `db:"target_fiber"`
}

func (r rollupRow) toRollup() types.DailyRollup {
	return types.DailyRollup{
		UserID: r.UserID,
		Date:   r.Date,
		Consumed: types.Macros{
			Calories: r.ConsumedKcal, Protein: r.ConsumedProtein, Carbs: r.ConsumedCarbs,
			Fat: r.ConsumedFat, Fiber: r.ConsumedFiber,
		},
		Targets: types.Macros{
			Calories: r.TargetKcal, Protein: r.TargetProtein, Carbs: r.TargetCarbs,
			Fat: r.TargetFat, Fiber: r.TargetFiber,
		},
	}
}

// resolvedItemRow is the flat DB shape of resolved_items; types.ResolvedItem
// groups these columns into nested Parsed/Match/Macros structs, and Per100g
// is reconstructed rather than stored.
type resolvedItemRow struct {
	MealID          string  `db:"meal_id"`
	RawPhrase       string  `db:"raw_phrase"`
	Quantity        float64 `db:"quantity"`
	Unit            string  `db:"unit"`
	NormalizedGrams float64 `db:"normalized_grams"`
	FoodID          string  `db:"food_id"`
	FoodName        string  `db:"food_name"`
	Source          string  `db:"source"`
	MatchScore      float64 `db:"match_score"`
	Kcal            float64 `db:"kcal"`
	Protein         float64 `db:"protein"`
	Carbs           float64 `db:"carbs"`
	Fat             float64 `db:"fat"`
	Fiber           float64 `db:"fiber"`
}

func (r resolvedItemRow) toResolvedItem() types.ResolvedItem {
	macros := types.Macros{Calories: r.Kcal, Protein: r.Protein, Carbs: r.Carbs, Fat: r.Fat, Fiber: r.Fiber}
	return types.ResolvedItem{
		Parsed: types.ParsedItem{
			RawPhrase: r.RawPhrase, Quantity: r.Quantity, Unit: r.Unit, NormalizedGrams: r.NormalizedGrams,
		},
		Match: types.FoodMatch{
			FoodID: r.FoodID, Name: r.FoodName, Source: r.Source, MatchScore: r.MatchScore,
			// Reconstruct Per100g from the absolute macros and portion grams.
			Per100g: macrosPer100g(macros, r.NormalizedGrams),
		},
		Macros: macros,
	}
}

func (s *Store) loadItems(ctx context.Context, mealIDs []string) (map[string][]types.ResolvedItem, error) {
	if len(mealIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(mealIDs))
	args := make([]any, len(mealIDs))
	for i, id := range mealIDs {
		placeholders[i] = s.dialect.Placeholder(i + 1)
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

	var rows []resolvedItemRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), args...); err != nil {
		return nil, fmt.Errorf("store: query items: %w", err)
	}

	out := make(map[string][]types.ResolvedItem)
	for _, r := range rows {
		out[r.MealID] = append(out[r.MealID], r.toResolvedItem())
	}
	return out, nil
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
	var row foodMatchRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID, normalized); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.FoodMatch{}, types.ErrNoMatch
		}
		return types.FoodMatch{}, fmt.Errorf("store: lookup food: %w", err)
	}
	fm := row.toFoodMatch()
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
	var row foodMatchRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID, foodID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.FoodMatch{}, types.ErrNoMatch
		}
		return types.FoodMatch{}, fmt.Errorf("store: get food: %w", err)
	}
	return row.toFoodMatch(), nil
}

// foodMatchRow is the flat DB shape shared by LookupFood and GetFood;
// types.FoodMatch nests the macro columns into Per100g.
type foodMatchRow struct {
	FoodID  string  `db:"food_id"`
	Name    string  `db:"name"`
	Source  string  `db:"source"`
	Kcal    float64 `db:"kcal_100g"`
	Protein float64 `db:"protein_100g"`
	Carbs   float64 `db:"carbs_100g"`
	Fat     float64 `db:"fat_100g"`
	Fiber   float64 `db:"fiber_100g"`
}

func (r foodMatchRow) toFoodMatch() types.FoodMatch {
	return types.FoodMatch{
		FoodID: r.FoodID,
		Name:   r.Name,
		Source: r.Source,
		Per100g: types.Macros{
			Calories: r.Kcal, Protein: r.Protein, Carbs: r.Carbs, Fat: r.Fat, Fiber: r.Fiber,
		},
	}
}

// UpsertFood inserts or replaces a food_library row and adds any new normalized
// aliases, all within a single transaction.
func (s *Store) UpsertFood(ctx context.Context, userID string, match types.FoodMatch, aliases []string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const foodQ = `
		INSERT INTO food_library
			(food_id, user_id, name, source, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g, query_count, last_used)
		VALUES (:food_id, :user_id, :name, :source, :kcal_100g, :protein_100g, :carbs_100g, :fat_100g, :fiber_100g, 0, '')
		ON CONFLICT(user_id, food_id) DO UPDATE SET
			name        = excluded.name,
			source      = excluded.source,
			kcal_100g   = excluded.kcal_100g,
			protein_100g= excluded.protein_100g,
			carbs_100g  = excluded.carbs_100g,
			fat_100g    = excluded.fat_100g,
			fiber_100g  = excluded.fiber_100g
	`
	foodQuery, foodArgs, err := sqlx.Named(foodQ, foodLibraryNamedArgs(userID, match))
	if err != nil {
		return fmt.Errorf("store: bind upsert food: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(foodQuery), foodArgs...); err != nil {
		return fmt.Errorf("store: upsert food: %w", err)
	}

	const aliasQ = `
		INSERT INTO food_aliases (user_id, alias_normalized, food_id)
		VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING
	`
	for _, alias := range aliases {
		normalized := normalize.Normalize(alias)
		if normalized == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, s.rewrite(aliasQ), userID, normalized, match.FoodID); err != nil {
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
	_, err := s.db.ExecContext(ctx, s.rewrite(q), utcNow(), userID, foodID)
	return err
}

// foodLibraryNamedArgs builds the named-parameter map shared by every upsert
// of a food_library row (UpsertFood, CorrectMealItem's cache refresh).
func foodLibraryNamedArgs(userID string, match types.FoodMatch) map[string]any {
	return map[string]any{
		"food_id":      match.FoodID,
		"user_id":      userID,
		"name":         match.Name,
		"source":       match.Source,
		"kcal_100g":    match.Per100g.Calories,
		"protein_100g": match.Per100g.Protein,
		"carbs_100g":   match.Per100g.Carbs,
		"fat_100g":     match.Per100g.Fat,
		"fiber_100g":   match.Per100g.Fiber,
	}
}

// mealOwnerRow is the flat DB shape used to fetch a meal's timestamp while
// checking that it belongs to the calling user.
type mealOwnerRow struct {
	AtUTC  string `db:"at_utc"`
	UserID string `db:"user_id"`
}

// mealOwner loads mealID's at_utc/user_id within tx and confirms it belongs
// to userID, returning types.ErrNotFound otherwise. Shared by CorrectMealItem,
// AddMealItem, and DeleteMealItem, which all gate a rollup mutation on the
// same ownership check.
func (s *Store) mealOwner(ctx context.Context, tx *sqlx.Tx, mealID, userID string) (mealOwnerRow, error) {
	const q = `SELECT at_utc, user_id FROM meals WHERE id = ?`
	var row mealOwnerRow
	if err := tx.GetContext(ctx, &row, s.rewrite(q), mealID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mealOwnerRow{}, types.ErrNotFound
		}
		return mealOwnerRow{}, fmt.Errorf("store: get meal: %w", err)
	}
	if row.UserID != userID {
		return mealOwnerRow{}, types.ErrNotFound
	}
	return row, nil
}

// CorrectMealItem updates one resolved item's macros for a meal, then
// recalculates the daily rollup and refreshes the food_library cache so future
// logs use the corrected values. itemIndex is the 0-based position of the item
// within the meal's items (ordered by rowid).
func (s *Store) CorrectMealItem(ctx context.Context, userID string, mealID string, itemIndex int, corrected types.ResolvedItem) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Load the meal to get the at time for rollup lookup and the original items.
	meal, err := s.mealOwner(ctx, tx, mealID, userID)
	if err != nil {
		return err
	}
	mealAt := parseUTC(meal.AtUTC)

	// Load items by rowid so we can find and update the target item.
	const itemsQ = `
		SELECT rowid, raw_phrase, quantity, unit, normalized_grams,
		       food_id, food_name, source, match_score,
		       kcal, protein, carbs, fat, fiber
		FROM resolved_items
		WHERE meal_id = ?
		ORDER BY rowid
	`
	var itemRows []mealItemRow
	if err := tx.SelectContext(ctx, &itemRows, s.rewrite(itemsQ), mealID); err != nil {
		return fmt.Errorf("store: query items: %w", err)
	}

	type item struct {
		rowid int64
		ri    types.ResolvedItem
	}
	items := make([]item, len(itemRows))
	var oldTotal types.Macros
	for i, r := range itemRows {
		items[i] = item{rowid: r.Rowid, ri: r.resolvedItemRow.toResolvedItem()}
		oldTotal = oldTotal.Add(items[i].ri.Macros)
	}
	if itemIndex < 0 || itemIndex >= len(items) {
		return fmt.Errorf("store: item index %d out of range [0, %d)", itemIndex, len(items))
	}

	// Replace the target item's macros and recalculate the new total.
	oldItemMacros := items[itemIndex].ri.Macros
	items[itemIndex].ri = corrected

	var newTotal types.Macros
	for _, it := range items {
		newTotal = newTotal.Add(it.ri.Macros)
	}

	// Update the resolved_items row.
	const updateQ = `
		UPDATE resolved_items SET
			normalized_grams = :normalized_grams, food_id = :food_id, food_name = :food_name,
			source = :source, match_score = :match_score,
			kcal = :kcal, protein = :protein, carbs = :carbs, fat = :fat, fiber = :fiber
		WHERE rowid = :rowid
	`
	updateQuery, updateArgs, err := sqlx.Named(updateQ, map[string]any{
		"normalized_grams": corrected.Parsed.NormalizedGrams,
		"food_id":          corrected.Match.FoodID,
		"food_name":        corrected.Match.Name,
		"source":           corrected.Match.Source,
		"match_score":      corrected.Match.MatchScore,
		"kcal":             corrected.Macros.Calories,
		"protein":          corrected.Macros.Protein,
		"carbs":            corrected.Macros.Carbs,
		"fat":              corrected.Macros.Fat,
		"fiber":            corrected.Macros.Fiber,
		"rowid":            items[itemIndex].rowid,
	})
	if err != nil {
		return fmt.Errorf("store: bind update item: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(updateQuery), updateArgs...); err != nil {
		return fmt.Errorf("store: update item: %w", err)
	}

	// Update the daily rollup: remove old macros, add new ones.
	localDate := mealAt.Format("2006-01-02")
	const rollupQ = `
		INSERT INTO daily_rollups
			(user_id, date,
			 consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
			 target_kcal, target_protein, target_carbs, target_fat, target_fiber)
		VALUES (:user_id, :date,
		        :new_total_kcal, :new_total_protein, :new_total_carbs, :new_total_fat, :new_total_fiber,
		        0, 0, 0, 0, 0)
		ON CONFLICT(user_id, date) DO UPDATE SET
			consumed_kcal    = consumed_kcal    - :old_kcal    + :corrected_kcal,
			consumed_protein = consumed_protein - :old_protein + :corrected_protein,
			consumed_carbs   = consumed_carbs   - :old_carbs   + :corrected_carbs,
			consumed_fat     = consumed_fat     - :old_fat     + :corrected_fat,
			consumed_fiber   = consumed_fiber   - :old_fiber   + :corrected_fiber
	`
	rollupQuery, rollupArgs, err := sqlx.Named(rollupQ, map[string]any{
		"user_id":           userID,
		"date":              localDate,
		"new_total_kcal":    newTotal.Calories,
		"new_total_protein": newTotal.Protein,
		"new_total_carbs":   newTotal.Carbs,
		"new_total_fat":     newTotal.Fat,
		"new_total_fiber":   newTotal.Fiber,
		"old_kcal":          oldItemMacros.Calories,
		"old_protein":       oldItemMacros.Protein,
		"old_carbs":         oldItemMacros.Carbs,
		"old_fat":           oldItemMacros.Fat,
		"old_fiber":         oldItemMacros.Fiber,
		"corrected_kcal":    corrected.Macros.Calories,
		"corrected_protein": corrected.Macros.Protein,
		"corrected_carbs":   corrected.Macros.Carbs,
		"corrected_fat":     corrected.Macros.Fat,
		"corrected_fiber":   corrected.Macros.Fiber,
	})
	if err != nil {
		return fmt.Errorf("store: bind update rollup: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(rollupQuery), rollupArgs...); err != nil {
		return fmt.Errorf("store: update rollup: %w", err)
	}

	// Refresh the food_library cache: upsert the corrected food so future
	// alias lookups use the corrected macros.
	if corrected.Match.FoodID != "" {
		const foodQ = `
			INSERT INTO food_library
				(food_id, user_id, name, source, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g, query_count, last_used)
			VALUES (:food_id, :user_id, :name, :source, :kcal_100g, :protein_100g, :carbs_100g, :fat_100g, :fiber_100g, 0, '')
			ON CONFLICT(user_id, food_id) DO UPDATE SET
				kcal_100g   = excluded.kcal_100g,
				protein_100g= excluded.protein_100g,
				carbs_100g  = excluded.carbs_100g,
				fat_100g    = excluded.fat_100g,
				fiber_100g  = excluded.fiber_100g
		`
		foodQuery, foodArgs, err := sqlx.Named(foodQ, foodLibraryNamedArgs(userID, corrected.Match))
		if err != nil {
			return fmt.Errorf("store: bind upsert food library: %w", err)
		}
		if _, err := tx.ExecContext(ctx, s.rewrite(foodQuery), foodArgs...); err != nil {
			return fmt.Errorf("store: upsert food library: %w", err)
		}
	}

	return tx.Commit()
}

// mealItemRow is resolvedItemRow plus the SQLite rowid, used where the
// caller must locate and later update or delete a specific resolved_items
// row (CorrectMealItem, DeleteMealItem).
type mealItemRow struct {
	Rowid int64 `db:"rowid"`
	resolvedItemRow
}

// AddMealItem appends a resolved item to an existing meal and adds its macros
// to that day's rollup. Mirrors CorrectMealItem's delta approach.
func (s *Store) AddMealItem(ctx context.Context, userID, mealID string, item types.ResolvedItem) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	meal, err := s.mealOwner(ctx, tx, mealID, userID)
	if err != nil {
		return err
	}

	const itemQ = `
		INSERT INTO resolved_items
			(id, meal_id, raw_phrase, quantity, unit, normalized_grams,
			 food_id, food_name, source, match_score,
			 kcal, protein, carbs, fat, fiber)
		VALUES (:id, :meal_id, :raw_phrase, :quantity, :unit, :normalized_grams,
			:food_id, :food_name, :source, :match_score,
			:kcal, :protein, :carbs, :fat, :fiber)
	`
	itemQuery, itemArgs, err := sqlx.Named(itemQ, resolvedItemNamedArgs(newID(), mealID, item))
	if err != nil {
		return fmt.Errorf("store: bind resolved_item: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(itemQuery), itemArgs...); err != nil {
		return fmt.Errorf("store: insert resolved_item: %w", err)
	}

	localDate := parseUTC(meal.AtUTC).Format("2006-01-02")
	const rollupQ = `
		INSERT INTO daily_rollups
			(user_id, date,
			 consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
			 target_kcal, target_protein, target_carbs, target_fat, target_fiber)
		VALUES (:user_id, :date, :kcal, :protein, :carbs, :fat, :fiber, 0, 0, 0, 0, 0)
		ON CONFLICT(user_id, date) DO UPDATE SET
			consumed_kcal    = consumed_kcal    + :kcal,
			consumed_protein = consumed_protein + :protein,
			consumed_carbs   = consumed_carbs   + :carbs,
			consumed_fat     = consumed_fat     + :fat,
			consumed_fiber   = consumed_fiber   + :fiber
	`
	m := item.Macros
	rollupQuery, rollupArgs, err := sqlx.Named(rollupQ, map[string]any{
		"user_id": userID, "date": localDate,
		"kcal": m.Calories, "protein": m.Protein, "carbs": m.Carbs, "fat": m.Fat, "fiber": m.Fiber,
	})
	if err != nil {
		return fmt.Errorf("store: bind update rollup: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(rollupQuery), rollupArgs...); err != nil {
		return fmt.Errorf("store: update rollup: %w", err)
	}

	return tx.Commit()
}

// DeleteMealItem removes the item at itemIndex (zero-based, rowid order) from a
// meal and subtracts its macros from that day's rollup.
func (s *Store) DeleteMealItem(ctx context.Context, userID, mealID string, itemIndex int) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	meal, err := s.mealOwner(ctx, tx, mealID, userID)
	if err != nil {
		return err
	}

	const itemsQ = `
		SELECT rowid, kcal, protein, carbs, fat, fiber
		FROM resolved_items WHERE meal_id = ? ORDER BY rowid
	`
	var items []mealItemRow
	if err := tx.SelectContext(ctx, &items, s.rewrite(itemsQ), mealID); err != nil {
		return fmt.Errorf("store: query items: %w", err)
	}
	if itemIndex < 0 || itemIndex >= len(items) {
		return fmt.Errorf("store: item index %d out of range [0, %d): %w", itemIndex, len(items), types.ErrNotFound)
	}

	target := items[itemIndex]
	if _, err := tx.ExecContext(ctx, s.rewrite(`DELETE FROM resolved_items WHERE rowid = ?`), target.Rowid); err != nil {
		return fmt.Errorf("store: delete item: %w", err)
	}

	localDate := parseUTC(meal.AtUTC).Format("2006-01-02")
	const rollupQ = `
		UPDATE daily_rollups SET
			consumed_kcal    = consumed_kcal    - ?,
			consumed_protein = consumed_protein - ?,
			consumed_carbs   = consumed_carbs   - ?,
			consumed_fat     = consumed_fat     - ?,
			consumed_fiber   = consumed_fiber   - ?
		WHERE user_id = ? AND date = ?
	`
	m := types.Macros{Calories: target.Kcal, Protein: target.Protein, Carbs: target.Carbs, Fat: target.Fat, Fiber: target.Fiber}
	if _, err := tx.ExecContext(ctx, s.rewrite(rollupQ),
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
	var row targetsRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.DailyTargets{}, types.ErrNotFound
		}
		return types.DailyTargets{}, err
	}
	return row.toTargets(), nil
}

// targetsRow is the flat DB shape of daily_targets; types.DailyTargets groups
// the macro columns into a nested Macros struct.
type targetsRow struct {
	UserID  string  `db:"user_id"`
	Kcal    float64 `db:"kcal"`
	Protein float64 `db:"protein"`
	Carbs   float64 `db:"carbs"`
	Fat     float64 `db:"fat"`
	Fiber   float64 `db:"fiber"`
}

func (r targetsRow) toTargets() types.DailyTargets {
	return types.DailyTargets{
		UserID:  r.UserID,
		Targets: types.Macros{Calories: r.Kcal, Protein: r.Protein, Carbs: r.Carbs, Fat: r.Fat, Fiber: r.Fiber},
	}
}

// SetTargets inserts or replaces the daily targets row.
func (s *Store) SetTargets(ctx context.Context, t types.DailyTargets) error {
	const q = `
		INSERT INTO daily_targets (user_id, kcal, protein, carbs, fat, fiber)
		VALUES (:user_id, :kcal, :protein, :carbs, :fat, :fiber)
		ON CONFLICT(user_id) DO UPDATE SET
			kcal    = excluded.kcal,
			protein = excluded.protein,
			carbs   = excluded.carbs,
			fat     = excluded.fat,
			fiber   = excluded.fiber
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"user_id": t.UserID,
		"kcal":    t.Targets.Calories, "protein": t.Targets.Protein, "carbs": t.Targets.Carbs,
		"fat": t.Targets.Fat, "fiber": t.Targets.Fiber,
	})
	if err != nil {
		return fmt.Errorf("store: bind set targets: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
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
		VALUES (:user_id, :date, 0, 0, 0, 0, 0, :kcal, :protein, :carbs, :fat, :fiber)
		ON CONFLICT(user_id, date) DO UPDATE SET
			target_kcal    = :kcal,
			target_protein = :protein,
			target_carbs   = :carbs,
			target_fat     = :fat,
			target_fiber   = :fiber
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"user_id": userID, "date": localDate,
		"kcal": t.Calories, "protein": t.Protein, "carbs": t.Carbs, "fat": t.Fat, "fiber": t.Fiber,
	})
	if err != nil {
		return fmt.Errorf("store: bind update rollup targets: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}

// ---------------------------------------------------------------------------
// Backup / scheduled export
// ---------------------------------------------------------------------------

// GetBackupConfig returns a user's backup settings, or types.ErrNotFound when
// none has been configured (callers treat "not found" as "disabled").
func (s *Store) GetBackupConfig(ctx context.Context, userID string) (types.BackupConfig, error) {
	const q = `
		SELECT user_id, enabled, destination, local_subdir, s3_bucket, s3_prefix, s3_region, s3_endpoint, interval_hrs, last_run_at
		FROM backup_config WHERE user_id = ?
	`
	var row backupConfigRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.BackupConfig{}, types.ErrNotFound
		}
		return types.BackupConfig{}, fmt.Errorf("store: get backup config: %w", err)
	}
	return row.toBackupConfig(), nil
}

// backupConfigRow is the flat DB shape of backup_config; types.BackupConfig
// stores Enabled as bool (DB: int) and LastRunAt as time.Time (DB: nullable
// RFC3339 string).
type backupConfigRow struct {
	UserID      string         `db:"user_id"`
	Enabled     int            `db:"enabled"`
	Destination string         `db:"destination"`
	LocalSubdir sql.NullString `db:"local_subdir"`
	S3Bucket    sql.NullString `db:"s3_bucket"`
	S3Prefix    sql.NullString `db:"s3_prefix"`
	S3Region    sql.NullString `db:"s3_region"`
	S3Endpoint  sql.NullString `db:"s3_endpoint"`
	IntervalHrs int            `db:"interval_hrs"`
	LastRunAt   sql.NullString `db:"last_run_at"`
}

func (r backupConfigRow) toBackupConfig() types.BackupConfig {
	cfg := types.BackupConfig{
		UserID:      r.UserID,
		Enabled:     r.Enabled != 0,
		Destination: r.Destination,
		LocalSubdir: r.LocalSubdir.String,
		S3Bucket:    r.S3Bucket.String,
		S3Prefix:    r.S3Prefix.String,
		S3Region:    r.S3Region.String,
		S3Endpoint:  r.S3Endpoint.String,
		IntervalHrs: r.IntervalHrs,
	}
	if r.LastRunAt.Valid && r.LastRunAt.String != "" {
		cfg.LastRunAt = parseUTC(r.LastRunAt.String)
	}
	return cfg
}

// SetBackupConfig inserts or replaces a user's backup settings.
func (s *Store) SetBackupConfig(ctx context.Context, cfg types.BackupConfig) error {
	enabled := 0
	if cfg.Enabled {
		enabled = 1
	}
	const q = `
		INSERT INTO backup_config
			(user_id, enabled, destination, local_subdir, s3_bucket, s3_prefix, s3_region, s3_endpoint, interval_hrs)
		VALUES (:user_id, :enabled, :destination, :local_subdir, :s3_bucket, :s3_prefix, :s3_region, :s3_endpoint, :interval_hrs)
		ON CONFLICT(user_id) DO UPDATE SET
			enabled      = excluded.enabled,
			destination  = excluded.destination,
			local_subdir = excluded.local_subdir,
			s3_bucket    = excluded.s3_bucket,
			s3_prefix    = excluded.s3_prefix,
			s3_region    = excluded.s3_region,
			s3_endpoint  = excluded.s3_endpoint,
			interval_hrs = excluded.interval_hrs
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"user_id": cfg.UserID, "enabled": enabled, "destination": cfg.Destination,
		"local_subdir": cfg.LocalSubdir, "s3_bucket": cfg.S3Bucket, "s3_prefix": cfg.S3Prefix,
		"s3_region": cfg.S3Region, "s3_endpoint": cfg.S3Endpoint, "interval_hrs": cfg.IntervalHrs,
	})
	if err != nil {
		return fmt.Errorf("store: bind set backup config: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}

// SetBackupLastRun records when a user's backup last completed.
func (s *Store) SetBackupLastRun(ctx context.Context, userID string, t time.Time) error {
	const q = `UPDATE backup_config SET last_run_at = ? WHERE user_id = ?`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), utcStr(t), userID)
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
	var row rollupRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID, localDate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.DailyRollup{}, types.ErrNotFound
		}
		return types.DailyRollup{}, err
	}
	return row.toRollup(), nil
}

// UpsertRollup inserts or replaces a daily rollup row.
func (s *Store) UpsertRollup(ctx context.Context, r types.DailyRollup) error {
	const q = `
		INSERT INTO daily_rollups
			(user_id, date,
			 consumed_kcal, consumed_protein, consumed_carbs, consumed_fat, consumed_fiber,
			 target_kcal, target_protein, target_carbs, target_fat, target_fiber)
		VALUES (:user_id, :date,
		        :consumed_kcal, :consumed_protein, :consumed_carbs, :consumed_fat, :consumed_fiber,
		        :target_kcal, :target_protein, :target_carbs, :target_fat, :target_fiber)
		ON CONFLICT(user_id, date) DO UPDATE SET
			consumed_kcal    = excluded.consumed_kcal,
			consumed_protein = excluded.consumed_protein,
			consumed_carbs   = excluded.consumed_carbs,
			consumed_fat     = excluded.consumed_fat,
			consumed_fiber   = excluded.consumed_fiber,
			target_kcal      = excluded.target_kcal,
			target_protein   = excluded.target_protein,
			target_carbs     = excluded.target_carbs,
			target_fat       = excluded.target_fat,
			target_fiber     = excluded.target_fiber
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"user_id": r.UserID, "date": r.Date,
		"consumed_kcal": r.Consumed.Calories, "consumed_protein": r.Consumed.Protein,
		"consumed_carbs": r.Consumed.Carbs, "consumed_fat": r.Consumed.Fat, "consumed_fiber": r.Consumed.Fiber,
		"target_kcal": r.Targets.Calories, "target_protein": r.Targets.Protein,
		"target_carbs": r.Targets.Carbs, "target_fat": r.Targets.Fat, "target_fiber": r.Targets.Fiber,
	})
	if err != nil {
		return fmt.Errorf("store: bind upsert rollup: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}

// ---------------------------------------------------------------------------
// Nudge dedupe
// ---------------------------------------------------------------------------

// WasNudged reports whether ruleID has already fired for this user on
// localDate. Satisfies scheduler.NudgeStore.
func (s *Store) WasNudged(ctx context.Context, userID, localDate, ruleID string) (bool, error) {
	const q = `SELECT 1 FROM nudge_log WHERE user_id = ? AND local_date = ? AND rule_id = ?`
	var v int
	err := s.db.GetContext(ctx, &v, s.rewrite(q), userID, localDate, ruleID)
	if errors.Is(err, sql.ErrNoRows) {
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
		INSERT INTO nudge_log (user_id, local_date, rule_id, sent_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT DO NOTHING
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID, localDate, ruleID, utcNow())
	return err
}

// ---------------------------------------------------------------------------
// Sent nudge tracking (undo / edit support)
// ---------------------------------------------------------------------------

// RecordSentNudge inserts a sent nudge row for later undo.
func (s *Store) RecordSentNudge(ctx context.Context, n types.SentNudge) error {
	snap, err := json.Marshal(n.Snapshot)
	if err != nil {
		return fmt.Errorf("store: marshal snapshot: %w", err)
	}
	const q = `
		INSERT INTO sent_nudges (id, user_id, rule_id, sent_at, body, snapshot_json, status)
		VALUES (:id, :user_id, :rule_id, :sent_at, :body, :snapshot_json, :status)
	`
	query, args, bindErr := sqlx.Named(q, map[string]any{
		"id": n.ID, "user_id": n.UserID, "rule_id": n.RuleID, "sent_at": utcStr(n.SentAt),
		"body": n.Body, "snapshot_json": string(snap), "status": n.Status,
	})
	if bindErr != nil {
		return fmt.Errorf("store: bind record sent nudge: %w", bindErr)
	}
	if _, err = s.db.ExecContext(ctx, s.rewrite(query), args...); err != nil {
		return fmt.Errorf("store: record sent nudge: %w", err)
	}
	return nil
}

// sentNudgeRow is the flat DB shape of sent_nudges; types.SentNudge nests
// Snapshot as a decoded Macros (DB: JSON string) and ResolvedAt as *time.Time
// (DB: nullable RFC3339 string).
type sentNudgeRow struct {
	ID           string         `db:"id"`
	UserID       string         `db:"user_id"`
	RuleID       string         `db:"rule_id"`
	SentAt       string         `db:"sent_at"`
	Body         string         `db:"body"`
	SnapshotJSON string         `db:"snapshot_json"`
	Status       string         `db:"status"`
	ResolvedAt   sql.NullString `db:"resolved_at"`
}

// GetSentNudge returns a sent nudge by id, or types.ErrNotFound.
func (s *Store) GetSentNudge(ctx context.Context, id string) (types.SentNudge, error) {
	const q = `SELECT id, user_id, rule_id, sent_at, body, snapshot_json, status, resolved_at FROM sent_nudges WHERE id = ?`
	var row sentNudgeRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.SentNudge{}, types.ErrNotFound
		}
		return types.SentNudge{}, fmt.Errorf("store: get sent nudge: %w", err)
	}
	n := types.SentNudge{
		ID: row.ID, UserID: row.UserID, RuleID: row.RuleID,
		SentAt: parseUTC(row.SentAt), Body: row.Body, Status: row.Status,
	}
	if row.ResolvedAt.Valid {
		n.ResolvedAt = new(time.Time)
		*n.ResolvedAt = parseUTC(row.ResolvedAt.String)
	}
	if err := json.Unmarshal([]byte(row.SnapshotJSON), &n.Snapshot); err != nil {
		return types.SentNudge{}, fmt.Errorf("store: unmarshal snapshot: %w", err)
	}
	return n, nil
}

// UpdateSentNudgeStatus marks a sent nudge with a terminal status and resolved_at.
func (s *Store) UpdateSentNudgeStatus(ctx context.Context, id, status string) error {
	const q = `UPDATE sent_nudges SET status = ?, resolved_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), status, utcNow(), id)
	if err != nil {
		return fmt.Errorf("store: update sent nudge status: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Nudge rule config (per-user overrides)
// ---------------------------------------------------------------------------

// GetNudgeRuleConfig returns every rule override a user has stored. Rules with
// no row here run with their hardcoded defaults. Satisfies
// scheduler.RuleConfigStore.
func (s *Store) GetNudgeRuleConfig(ctx context.Context, userID string) ([]types.NudgeRuleConfig, error) {
	const q = `SELECT user_id, rule_id, enabled, params_json FROM nudge_rule_config WHERE user_id = ?`
	type ruleConfigRow struct {
		UserID     string `db:"user_id"`
		RuleID     string `db:"rule_id"`
		Enabled    int    `db:"enabled"`
		ParamsJSON string `db:"params_json"`
	}
	var rows []ruleConfigRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID); err != nil {
		return nil, fmt.Errorf("store: query nudge rule config: %w", err)
	}

	var out []types.NudgeRuleConfig
	for _, r := range rows {
		out = append(out, types.NudgeRuleConfig{
			UserID:  r.UserID,
			RuleID:  r.RuleID,
			Enabled: r.Enabled != 0,
			Params:  json.RawMessage(r.ParamsJSON),
		})
	}
	return out, nil
}

// SetNudgeRuleConfig upserts a per-user override for one rule.
func (s *Store) SetNudgeRuleConfig(ctx context.Context, userID, ruleID string, enabled bool, params json.RawMessage) error {
	if len(params) == 0 {
		params = json.RawMessage("{}")
	}
	const q = `
		INSERT INTO nudge_rule_config (user_id, rule_id, enabled, params_json)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, rule_id) DO UPDATE SET
			enabled     = excluded.enabled,
			params_json = excluded.params_json
	`
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID, ruleID, enabledInt, string(params))
	if err != nil {
		return fmt.Errorf("store: set nudge rule config: %w", err)
	}
	return nil
}

// DeleteNudgeRuleConfig resets a rule to its hardcoded default by removing the
// override row. No error if nothing existed.
func (s *Store) DeleteNudgeRuleConfig(ctx context.Context, userID, ruleID string) error {
	const q = `DELETE FROM nudge_rule_config WHERE user_id = ? AND rule_id = ?`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID, ruleID)
	if err != nil {
		return fmt.Errorf("store: delete nudge rule config: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func utcStr(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func utcNow() string { return time.Now().UTC().Format(time.RFC3339) }

// nullStr returns nil for an empty string, otherwise returns the string.
// Used to store nullable TEXT columns as SQL NULL instead of "".
func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

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

// ptrTime returns a pointer to t.
func ptrTime(t time.Time) *time.Time {
	p := new(time.Time)
	*p = t
	return p
}

// isUniqueViolation reports whether err is a SQL UNIQUE constraint violation.
// Works with modernc.org/sqlite; kept simple and portable.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// modernc.org/sqlite surfaces this in the error string.
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// newID returns a short pseudo-unique ID using a monotonic counter + timestamp
// fallback. Simple identifiers keep the embedded DB readable.
var idCounter int64

func newID() string {
	n := atomic.AddInt64(&idCounter, 1)
	return fmt.Sprintf("%d%x", time.Now().UnixNano(), n)
}

// ---------------------------------------------------------------------------
// Latest meal
// ---------------------------------------------------------------------------

// LatestMealTime returns the most recent meal timestamp for a user, or
// types.ErrNotFound when no meals exist.
func (s *Store) LatestMealTime(ctx context.Context, userID string) (string, error) {
	const q = `SELECT at_utc FROM meals WHERE user_id = ? ORDER BY created_at DESC LIMIT 1`
	var at string
	if err := s.db.GetContext(ctx, &at, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", types.ErrNotFound
		}
		return "", fmt.Errorf("store: latest meal time: %w", err)
	}
	return at, nil
}

// ---------------------------------------------------------------------------
// Food discovery
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

	var rows []foodDetailRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), args...); err != nil {
		return nil, fmt.Errorf("store: list foods: %w", err)
	}
	return foodDetailRows(rows), nil
}

// SearchFoods searches food_library by name and alias using full-text search.
func (s *Store) SearchFoods(ctx context.Context, userID, query string) ([]types.FoodDetail, error) {
	searchParam := s.dialect.SearchQuery(query)

	var q string
	if s.driver == "postgres" {
		q = `
		SELECT fl.food_id, fl.user_id, fl.name, fl.source,
		       fl.kcal_100g, fl.protein_100g, fl.carbs_100g, fl.fat_100g, fl.fiber_100g,
		       fl.category, fl.brand, fl.barcode, fl.image_url, fl.serving_size, fl.serving_unit,
		       fl.query_count, fl.last_used
		FROM food_library fl
		WHERE fl.user_id = ? AND fl.food_id IN (
			SELECT fs.food_id FROM food_search fs WHERE fs.tsv @@ to_tsquery('simple', ?) AND fs.user_id = ?
		)
		ORDER BY fl.query_count DESC
		LIMIT 20
	`
	} else {
		q = `
		SELECT fl.food_id, fl.user_id, fl.name, fl.source,
		       fl.kcal_100g, fl.protein_100g, fl.carbs_100g, fl.fat_100g, fl.fiber_100g,
		       fl.category, fl.brand, fl.barcode, fl.image_url, fl.serving_size, fl.serving_unit,
		       fl.query_count, fl.last_used
		FROM food_library fl
		WHERE fl.user_id = ? AND fl.food_id IN (
			SELECT fs.food_id FROM food_search fs WHERE food_search MATCH ? AND fs.user_id = ?
		)
		ORDER BY fl.query_count DESC
		LIMIT 20
	`
	}
	var rows []foodDetailRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID, searchParam, userID); err != nil {
		return nil, fmt.Errorf("store: search foods: %w", err)
	}
	return foodDetailRows(rows), nil
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
	var rows []foodDetailRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID, limit); err != nil {
		return nil, fmt.Errorf("store: frequent foods: %w", err)
	}
	return foodDetailRows(rows), nil
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
	var row foodDetailRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(foodQ), userID, foodID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.FoodDetail{}, types.ErrNotFound
		}
		return types.FoodDetail{}, fmt.Errorf("store: get food detail: %w", err)
	}
	fd := row.toFoodDetail()

	// Load aliases separately.
	const aliasQ = `
		SELECT food_id, alias_normalized FROM food_aliases
		WHERE user_id = ? AND food_id = ?
	`
	type aliasRow struct {
		FoodID     string `db:"food_id"`
		Normalized string `db:"alias_normalized"`
	}
	var aliases []aliasRow
	if err := s.db.SelectContext(ctx, &aliases, s.rewrite(aliasQ), userID, foodID); err != nil {
		return types.FoodDetail{}, fmt.Errorf("store: get food aliases: %w", err)
	}
	for _, a := range aliases {
		fd.Aliases = append(fd.Aliases, types.FoodAlias{FoodID: a.FoodID, Normalized: a.Normalized})
	}
	return fd, nil
}

// AddFoodAlias inserts a normalized alias for a food.
func (s *Store) AddFoodAlias(ctx context.Context, userID, foodID, alias string) error {
	normalized := normalize.Normalize(alias)
	const q = `INSERT INTO food_aliases (user_id, alias_normalized, food_id) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID, normalized, foodID)
	return err
}

// DeleteFoodAlias removes a normalized alias for a food. Returns ErrNotFound if
// no row was deleted.
func (s *Store) DeleteFoodAlias(ctx context.Context, userID, foodID, alias string) error {
	normalized := normalize.Normalize(alias)
	const q = `DELETE FROM food_aliases WHERE user_id = ? AND food_id = ? AND alias_normalized = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), userID, foodID, normalized)
	if err != nil {
		return fmt.Errorf("store: delete food alias: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// AddPendingAlias queues an embedding-matched phrase for user confirmation
// instead of writing it straight into food_aliases.
func (s *Store) AddPendingAlias(ctx context.Context, userID, phrase, foodID string, matchScore float64) error {
	const q = `
		INSERT INTO pending_aliases (id, user_id, phrase, food_id, match_score, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), newID(), userID, phrase, foodID, matchScore, utcStr(time.Now()))
	if err != nil {
		return fmt.Errorf("store: add pending alias: %w", err)
	}
	return nil
}

// ListPendingAliases returns every alias candidate awaiting confirmation for a user.
func (s *Store) ListPendingAliases(ctx context.Context, userID string) ([]types.PendingAlias, error) {
	const q = `
		SELECT id, user_id, phrase, food_id, match_score, created_at
		FROM pending_aliases
		WHERE user_id = ?
		ORDER BY created_at DESC
	`
	type pendingAliasRow struct {
		ID         string  `db:"id"`
		UserID     string  `db:"user_id"`
		Phrase     string  `db:"phrase"`
		FoodID     string  `db:"food_id"`
		MatchScore float64 `db:"match_score"`
		CreatedAt  string  `db:"created_at"`
	}
	var rows []pendingAliasRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID); err != nil {
		return nil, fmt.Errorf("store: list pending aliases: %w", err)
	}

	var out []types.PendingAlias
	for _, r := range rows {
		out = append(out, types.PendingAlias{
			ID: r.ID, UserID: r.UserID, Phrase: r.Phrase, FoodID: r.FoodID,
			MatchScore: r.MatchScore, CreatedAt: parseUTC(r.CreatedAt),
		})
	}
	return out, nil
}

// ConfirmPendingAlias promotes a pending alias into food_aliases and removes
// the pending row, in one transaction. Returns types.ErrNotFound if the
// pending row doesn't exist or doesn't belong to userID.
func (s *Store) ConfirmPendingAlias(ctx context.Context, userID, id string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const selectQ = `SELECT phrase, food_id FROM pending_aliases WHERE id = ? AND user_id = ?`
	var pending struct {
		Phrase string `db:"phrase"`
		FoodID string `db:"food_id"`
	}
	if err := tx.GetContext(ctx, &pending, s.rewrite(selectQ), id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrNotFound
		}
		return fmt.Errorf("store: load pending alias: %w", err)
	}

	const aliasQ = `INSERT INTO food_aliases (user_id, alias_normalized, food_id) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`
	normalized := normalize.Normalize(pending.Phrase)
	if _, err := tx.ExecContext(ctx, s.rewrite(aliasQ), userID, normalized, pending.FoodID); err != nil {
		return fmt.Errorf("store: insert alias: %w", err)
	}

	const deleteQ = `DELETE FROM pending_aliases WHERE id = ? AND user_id = ?`
	if _, err := tx.ExecContext(ctx, s.rewrite(deleteQ), id, userID); err != nil {
		return fmt.Errorf("store: delete pending alias: %w", err)
	}

	return tx.Commit()
}

// RejectPendingAlias discards a pending alias without promoting it. Returns
// types.ErrNotFound if no row matched.
func (s *Store) RejectPendingAlias(ctx context.Context, userID, id string) error {
	const q = `DELETE FROM pending_aliases WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), id, userID)
	if err != nil {
		return fmt.Errorf("store: reject pending alias: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// GetSourcePrecedence returns a user's customized nutrition-source order, or
// an empty slice (not an error) if they have none — the resolver falls back
// to its startup-configured default order in that case.
func (s *Store) GetSourcePrecedence(ctx context.Context, userID string) ([]string, error) {
	const q = `SELECT source FROM source_precedence WHERE user_id = ? ORDER BY rank ASC`
	out := []string{}
	if err := s.db.SelectContext(ctx, &out, s.rewrite(q), userID); err != nil {
		return nil, fmt.Errorf("store: get source precedence: %w", err)
	}
	return out, nil
}

// SetSourcePrecedence replaces a user's nutrition-source order with the given
// list, ranked by position.
func (s *Store) SetSourcePrecedence(ctx context.Context, userID string, order []string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const deleteQ = `DELETE FROM source_precedence WHERE user_id = ?`
	if _, err := tx.ExecContext(ctx, s.rewrite(deleteQ), userID); err != nil {
		return fmt.Errorf("store: clear source precedence: %w", err)
	}

	const insertQ = `INSERT INTO source_precedence (user_id, source, rank) VALUES (?, ?, ?)`
	for i, source := range order {
		if _, err := tx.ExecContext(ctx, s.rewrite(insertQ), userID, source, i); err != nil {
			return fmt.Errorf("store: insert source precedence: %w", err)
		}
	}

	return tx.Commit()
}

// foodDetailRow is the flat DB shape of food_library; types.FoodDetail nests
// the macro columns into Per100g and carries a non-column Aliases slice
// populated separately (GetFoodDetail).
type foodDetailRow struct {
	FoodID      string  `db:"food_id"`
	UserID      string  `db:"user_id"`
	Name        string  `db:"name"`
	Source      string  `db:"source"`
	Kcal        float64 `db:"kcal_100g"`
	Protein     float64 `db:"protein_100g"`
	Carbs       float64 `db:"carbs_100g"`
	Fat         float64 `db:"fat_100g"`
	Fiber       float64 `db:"fiber_100g"`
	Category    string  `db:"category"`
	Brand       string  `db:"brand"`
	Barcode     string  `db:"barcode"`
	ImageURL    string  `db:"image_url"`
	ServingSize float64 `db:"serving_size"`
	ServingUnit string  `db:"serving_unit"`
	QueryCount  int     `db:"query_count"`
	LastUsed    string  `db:"last_used"`
}

func (r foodDetailRow) toFoodDetail() types.FoodDetail {
	return types.FoodDetail{
		FoodID: r.FoodID,
		UserID: r.UserID,
		Name:   r.Name,
		Source: r.Source,
		Per100g: types.Macros{
			Calories: r.Kcal, Protein: r.Protein, Carbs: r.Carbs, Fat: r.Fat, Fiber: r.Fiber,
		},
		Category:    r.Category,
		Brand:       r.Brand,
		Barcode:     r.Barcode,
		ImageURL:    r.ImageURL,
		ServingSize: r.ServingSize,
		ServingUnit: r.ServingUnit,
		QueryCount:  r.QueryCount,
		LastUsed:    r.LastUsed,
		Aliases:     []types.FoodAlias{},
	}
}

func foodDetailRows(rows []foodDetailRow) []types.FoodDetail {
	out := make([]types.FoodDetail, len(rows))
	for i, r := range rows {
		out[i] = r.toFoodDetail()
	}
	return out
}

// ---------------------------------------------------------------------------
// Meal templates
// ---------------------------------------------------------------------------

// SaveTemplate inserts or upserts a meal template.
func (s *Store) SaveTemplate(ctx context.Context, t types.MealTemplate) error {
	itemsJSON, err := json.Marshal(t.Items)
	if err != nil {
		return fmt.Errorf("store: marshal template items: %w", err)
	}
	const q = `
		INSERT INTO meal_templates (id, user_id, name, items_json, created_at, last_used)
		VALUES (:id, :user_id, :name, :items_json, :created_at, :last_used)
		ON CONFLICT(id) DO UPDATE SET
			name       = excluded.name,
			items_json = excluded.items_json,
			last_used  = excluded.last_used
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"id": t.ID, "user_id": t.UserID, "name": t.Name, "items_json": string(itemsJSON),
		"created_at": utcStr(t.CreatedAt), "last_used": utcStr(t.LastUsed),
	})
	if err != nil {
		return fmt.Errorf("store: bind save template: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}

// GetTemplates returns all templates for a user, newest first.
func (s *Store) GetTemplates(ctx context.Context, userID string) ([]types.MealTemplate, error) {
	const q = `
		SELECT id, user_id, name, items_json, created_at, last_used
		FROM meal_templates WHERE user_id = ?
		ORDER BY created_at DESC
	`
	var rows []templateRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID); err != nil {
		return nil, fmt.Errorf("store: get templates: %w", err)
	}
	out := make([]types.MealTemplate, 0, len(rows))
	for _, r := range rows {
		t, err := r.toTemplate()
		if err != nil {
			return nil, fmt.Errorf("store: unmarshal template items: %w", err)
		}
		out = append(out, t)
	}
	return out, nil
}

// GetTemplate returns a single template by ID.
func (s *Store) GetTemplate(ctx context.Context, templateID string) (types.MealTemplate, error) {
	const q = `
		SELECT id, user_id, name, items_json, created_at, last_used
		FROM meal_templates WHERE id = ?
	`
	var row templateRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), templateID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.MealTemplate{}, types.ErrNotFound
		}
		return types.MealTemplate{}, err
	}
	t, err := row.toTemplate()
	if err != nil {
		return types.MealTemplate{}, fmt.Errorf("store: unmarshal template items: %w", err)
	}
	return t, nil
}

// DeleteTemplate deletes a template by user + ID. Returns ErrNotFound if 0 rows.
func (s *Store) DeleteTemplate(ctx context.Context, userID, templateID string) error {
	const q = `DELETE FROM meal_templates WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), templateID, userID)
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
	_, err := s.db.ExecContext(ctx, s.rewrite(q), tl.ID, tl.UserID, tl.TemplateID, utcStr(tl.LoggedAt))
	return err
}

// templateRow is the flat DB shape of meal_templates; types.MealTemplate
// decodes ItemsJSON into Items and parses the RFC3339 timestamp columns.
type templateRow struct {
	ID        string `db:"id"`
	UserID    string `db:"user_id"`
	Name      string `db:"name"`
	ItemsJSON string `db:"items_json"`
	CreatedAt string `db:"created_at"`
	LastUsed  string `db:"last_used"`
}

func (r templateRow) toTemplate() (types.MealTemplate, error) {
	t := types.MealTemplate{
		ID: r.ID, UserID: r.UserID, Name: r.Name,
		CreatedAt: parseUTC(r.CreatedAt), LastUsed: parseUTC(r.LastUsed),
	}
	if err := json.Unmarshal([]byte(r.ItemsJSON), &t.Items); err != nil {
		return types.MealTemplate{}, err
	}
	if t.Items == nil {
		t.Items = []types.ResolvedItem{}
	}
	return t, nil
}

// ---------------------------------------------------------------------------
// Body tracking
// ---------------------------------------------------------------------------

// ListWeight returns weight entries for the last N days.
func (s *Store) ListWeight(ctx context.Context, userID string, days int) ([]types.WeightEntry, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	const q = `
		SELECT id, user_id, date, weight_kg, note, created_at
		FROM weight_log
		WHERE user_id = ? AND date >= ?
		ORDER BY date ASC
	`
	var rows []weightRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID, cutoff); err != nil {
		return nil, fmt.Errorf("store: list weight: %w", err)
	}
	out := make([]types.WeightEntry, len(rows))
	for i, r := range rows {
		out[i] = r.toWeightEntry()
	}
	return out, nil
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
	_, err := s.db.ExecContext(ctx, s.rewrite(q), w.ID, w.UserID, w.Date, w.WeightKg, w.Note, utcStr(w.CreatedAt))
	return err
}

// DeleteWeight deletes a weight entry by user + ID. Returns ErrNotFound if absent.
func (s *Store) DeleteWeight(ctx context.Context, userID, entryID string) error {
	const q = `DELETE FROM weight_log WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), entryID, userID)
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
		start := max(i-6, 0)
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

// weightRow is the flat DB shape of weight_log; types.WeightEntry parses
// CreatedAt from the stored RFC3339 string.
type weightRow struct {
	ID        string  `db:"id"`
	UserID    string  `db:"user_id"`
	Date      string  `db:"date"`
	WeightKg  float64 `db:"weight_kg"`
	Note      string  `db:"note"`
	CreatedAt string  `db:"created_at"`
}

func (r weightRow) toWeightEntry() types.WeightEntry {
	return types.WeightEntry{
		ID: r.ID, UserID: r.UserID, Date: r.Date, WeightKg: r.WeightKg,
		Note: r.Note, CreatedAt: parseUTC(r.CreatedAt),
	}
}

// --- Fasting ---

// StartFast inserts a new fasting window. Callers should ensure no active fast
// exists first (see GetActiveFast).
func (s *Store) StartFast(ctx context.Context, f types.Fast) error {
	const q = `
		INSERT INTO fasts (id, user_id, start_at, end_at, target_hours, completed, created_at)
		VALUES (?, ?, ?, NULL, ?, 0, ?)
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), f.ID, f.UserID, utcStr(f.StartAt), f.TargetHours, utcStr(f.CreatedAt))
	if err != nil {
		return fmt.Errorf("store: start fast: %w", err)
	}
	return nil
}

// GetActiveFast returns the user's in-progress fast (end_at IS NULL), or
// ErrNotFound if none is active.
func (s *Store) GetActiveFast(ctx context.Context, userID string) (types.Fast, error) {
	const q = `
		SELECT id, user_id, start_at, end_at, target_hours, completed, created_at
		FROM fasts
		WHERE user_id = ? AND end_at IS NULL
		ORDER BY start_at DESC
		LIMIT 1
	`
	var row fastRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Fast{}, types.ErrNotFound
		}
		return types.Fast{}, err
	}
	return row.toFast(), nil
}

// EndFast closes a fasting window by id, marking its end time and completion.
// Returns the updated fast, or ErrNotFound if no matching active fast exists.
func (s *Store) EndFast(ctx context.Context, userID, fastID string, endAt time.Time, completed bool) (types.Fast, error) {
	const q = `
		UPDATE fasts SET end_at = ?, completed = ?
		WHERE id = ? AND user_id = ? AND end_at IS NULL
	`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), utcStr(endAt), boolToInt(completed), fastID, userID)
	if err != nil {
		return types.Fast{}, fmt.Errorf("store: end fast: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return types.Fast{}, types.ErrNotFound
	}
	const sel = `
		SELECT id, user_id, start_at, end_at, target_hours, completed, created_at
		FROM fasts WHERE id = ? AND user_id = ?
	`
	var row fastRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(sel), fastID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Fast{}, types.ErrNotFound
		}
		return types.Fast{}, err
	}
	return row.toFast(), nil
}

// ListFasts returns the user's most recent fasting windows, newest first.
func (s *Store) ListFasts(ctx context.Context, userID string, limit int) ([]types.Fast, error) {
	const q = `
		SELECT id, user_id, start_at, end_at, target_hours, completed, created_at
		FROM fasts
		WHERE user_id = ?
		ORDER BY start_at DESC
		LIMIT ?
	`
	var rows []fastRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID, limit); err != nil {
		return nil, fmt.Errorf("store: list fasts: %w", err)
	}
	out := make([]types.Fast, len(rows))
	for i, r := range rows {
		out[i] = r.toFast()
	}
	return out, nil
}

// fastRow is the flat DB shape of fasts; types.Fast nests EndAt as *time.Time
// (DB: nullable RFC3339 string) and Completed as bool (DB: int).
type fastRow struct {
	ID          string         `db:"id"`
	UserID      string         `db:"user_id"`
	StartAt     string         `db:"start_at"`
	EndAt       sql.NullString `db:"end_at"`
	TargetHours float64        `db:"target_hours"`
	Completed   int            `db:"completed"`
	CreatedAt   string         `db:"created_at"`
}

func (r fastRow) toFast() types.Fast {
	f := types.Fast{
		ID: r.ID, UserID: r.UserID, StartAt: parseUTC(r.StartAt),
		TargetHours: r.TargetHours, Completed: r.Completed != 0, CreatedAt: parseUTC(r.CreatedAt),
	}
	if r.EndAt.Valid && r.EndAt.String != "" {
		f.EndAt = new(parseUTC(r.EndAt.String))
	}
	return f
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ListMeasurements returns measurement entries for the last N days.
func (s *Store) ListMeasurements(ctx context.Context, userID string, days int) ([]types.MeasurementEntry, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	const q = `
		SELECT id, user_id, date, waist_cm, hips_cm, chest_cm, left_arm_cm, right_arm_cm,
		       left_thigh_cm, right_thigh_cm, note, created_at
		FROM measurement_log
		WHERE user_id = ? AND date >= ?
		ORDER BY date ASC
	`
	var rows []measurementRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID, cutoff); err != nil {
		return nil, fmt.Errorf("store: list measurements: %w", err)
	}
	out := make([]types.MeasurementEntry, len(rows))
	for i, r := range rows {
		out[i] = r.toMeasurementEntry()
	}
	return out, nil
}

// LogMeasurement inserts or updates a measurement entry.
func (s *Store) LogMeasurement(ctx context.Context, m types.MeasurementEntry) error {
	const q = `
		INSERT INTO measurement_log
			(id, user_id, date, waist_cm, hips_cm, chest_cm, left_arm_cm, right_arm_cm,
			 left_thigh_cm, right_thigh_cm, note, created_at)
		VALUES (:id, :user_id, :date, :waist_cm, :hips_cm, :chest_cm, :left_arm_cm, :right_arm_cm,
			:left_thigh_cm, :right_thigh_cm, :note, :created_at)
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
	query, args, err := sqlx.Named(q, map[string]any{
		"id": m.ID, "user_id": m.UserID, "date": m.Date,
		"waist_cm": m.WaistCm, "hips_cm": m.HipsCm, "chest_cm": m.ChestCm,
		"left_arm_cm": m.LeftArmCm, "right_arm_cm": m.RightArmCm,
		"left_thigh_cm": m.LeftThighCm, "right_thigh_cm": m.RightThighCm,
		"note": m.Note, "created_at": utcStr(m.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("store: bind log measurement: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}

// DeleteMeasurement deletes a measurement entry by user + ID. Returns ErrNotFound.
func (s *Store) DeleteMeasurement(ctx context.Context, userID, entryID string) error {
	const q = `DELETE FROM measurement_log WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), entryID, userID)
	if err != nil {
		return fmt.Errorf("store: delete measurement: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// measurementRow is the flat DB shape of measurement_log; types.MeasurementEntry
// parses CreatedAt from the stored RFC3339 string.
type measurementRow struct {
	ID           string  `db:"id"`
	UserID       string  `db:"user_id"`
	Date         string  `db:"date"`
	WaistCm      float64 `db:"waist_cm"`
	HipsCm       float64 `db:"hips_cm"`
	ChestCm      float64 `db:"chest_cm"`
	LeftArmCm    float64 `db:"left_arm_cm"`
	RightArmCm   float64 `db:"right_arm_cm"`
	LeftThighCm  float64 `db:"left_thigh_cm"`
	RightThighCm float64 `db:"right_thigh_cm"`
	Note         string  `db:"note"`
	CreatedAt    string  `db:"created_at"`
}

func (r measurementRow) toMeasurementEntry() types.MeasurementEntry {
	return types.MeasurementEntry{
		ID: r.ID, UserID: r.UserID, Date: r.Date,
		WaistCm: r.WaistCm, HipsCm: r.HipsCm, ChestCm: r.ChestCm,
		LeftArmCm: r.LeftArmCm, RightArmCm: r.RightArmCm,
		LeftThighCm: r.LeftThighCm, RightThighCm: r.RightThighCm,
		Note: r.Note, CreatedAt: parseUTC(r.CreatedAt),
	}
}

// ListPhotoMetadata returns progress photo records without the BLOB data.
func (s *Store) ListPhotoMetadata(ctx context.Context, userID string) ([]types.ProgressPhoto, error) {
	const q = `
		SELECT id, user_id, date, view, mime_type, created_at
		FROM progress_photos WHERE user_id = ?
		ORDER BY date DESC
	`
	var rows []photoRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID); err != nil {
		return nil, fmt.Errorf("store: list photo metadata: %w", err)
	}
	out := make([]types.ProgressPhoto, len(rows))
	for i, r := range rows {
		out[i] = r.toProgressPhoto()
	}
	return out, nil
}

// photoRow is the flat DB shape of progress_photos; types.ProgressPhoto parses
// CreatedAt from the stored RFC3339 string. Data is left zero-value when the
// query (ListPhotoMetadata) doesn't select the BLOB column.
type photoRow struct {
	ID        string `db:"id"`
	UserID    string `db:"user_id"`
	Date      string `db:"date"`
	View      string `db:"view"`
	MimeType  string `db:"mime_type"`
	Data      []byte `db:"data"`
	CreatedAt string `db:"created_at"`
}

func (r photoRow) toProgressPhoto() types.ProgressPhoto {
	return types.ProgressPhoto{
		ID: r.ID, UserID: r.UserID, Date: r.Date, View: r.View, MimeType: r.MimeType,
		Data: r.Data, CreatedAt: parseUTC(r.CreatedAt),
	}
}

// GetPhotoData returns a single progress photo including BLOB data.
func (s *Store) GetPhotoData(ctx context.Context, photoID string) (types.ProgressPhoto, error) {
	const q = `
		SELECT id, user_id, date, view, mime_type, data, created_at
		FROM progress_photos WHERE id = ?
	`
	var row photoRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), photoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ProgressPhoto{}, types.ErrNotFound
		}
		return types.ProgressPhoto{}, fmt.Errorf("store: get photo data: %w", err)
	}
	return row.toProgressPhoto(), nil
}

// UploadPhoto inserts a progress photo with BLOB data.
func (s *Store) UploadPhoto(ctx context.Context, p types.ProgressPhoto) error {
	const q = `
		INSERT INTO progress_photos (id, user_id, date, view, mime_type, data, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), p.ID, p.UserID, p.Date, p.View, p.MimeType, p.Data, utcStr(p.CreatedAt))
	return err
}

// DeletePhoto deletes a progress photo by user + ID. Returns ErrNotFound if absent.
func (s *Store) DeletePhoto(ctx context.Context, userID, photoID string) error {
	const q = `DELETE FROM progress_photos WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), photoID, userID)
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
	var rows []mealRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(mealQ), userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("store: query meals in range: %w", err)
	}

	var meals []types.Meal
	var mealIDs []string
	for _, r := range rows {
		m := r.toMeal()
		meals = append(meals, m)
		mealIDs = append(mealIDs, m.ID)
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
// Goals & profile
// ---------------------------------------------------------------------------

// GetProfile returns the user profile, or ErrNotFound.
func (s *Store) GetProfile(ctx context.Context, userID string) (types.UserProfile, error) {
	const q = `
		SELECT user_id, height_cm, birth_date, gender, activity_level, goal,
		       target_weight_kg, weekly_rate, onboarded, created_at, updated_at
		FROM user_profiles WHERE user_id = ?
	`
	var row profileRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.UserProfile{}, types.ErrNotFound
		}
		return types.UserProfile{}, fmt.Errorf("store: get profile: %w", err)
	}
	return row.toUserProfile(), nil
}

// profileRow is the flat DB shape of user_profiles; types.UserProfile stores
// Onboarded as bool (DB: int) and CreatedAt/UpdatedAt as time.Time (DB:
// RFC3339 strings).
type profileRow struct {
	UserID         string  `db:"user_id"`
	HeightCm       float64 `db:"height_cm"`
	BirthDate      string  `db:"birth_date"`
	Gender         string  `db:"gender"`
	ActivityLevel  string  `db:"activity_level"`
	Goal           string  `db:"goal"`
	TargetWeightKg float64 `db:"target_weight_kg"`
	WeeklyRate     float64 `db:"weekly_rate"`
	Onboarded      int     `db:"onboarded"`
	CreatedAt      string  `db:"created_at"`
	UpdatedAt      string  `db:"updated_at"`
}

func (r profileRow) toUserProfile() types.UserProfile {
	return types.UserProfile{
		UserID: r.UserID, HeightCm: r.HeightCm, BirthDate: r.BirthDate, Gender: r.Gender,
		ActivityLevel: r.ActivityLevel, Goal: r.Goal, TargetWeightKg: r.TargetWeightKg,
		WeeklyRate: r.WeeklyRate, Onboarded: r.Onboarded != 0,
		CreatedAt: parseUTC(r.CreatedAt), UpdatedAt: parseUTC(r.UpdatedAt),
	}
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
		VALUES (:user_id, :height_cm, :birth_date, :gender, :activity_level, :goal,
			:target_weight_kg, :weekly_rate, :onboarded, :created_at, :updated_at)
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
	query, args, err := sqlx.Named(q, map[string]any{
		"user_id": p.UserID, "height_cm": p.HeightCm, "birth_date": p.BirthDate,
		"gender": p.Gender, "activity_level": p.ActivityLevel, "goal": p.Goal,
		"target_weight_kg": p.TargetWeightKg, "weekly_rate": p.WeeklyRate, "onboarded": onboarded,
		"created_at": utcStr(p.CreatedAt), "updated_at": utcStr(p.UpdatedAt),
	})
	if err != nil {
		return fmt.Errorf("store: bind upsert profile: %w", err)
	}
	_, err = s.db.ExecContext(ctx, s.rewrite(query), args...)
	return err
}

// ---------------------------------------------------------------------------
// Linking codes
// ---------------------------------------------------------------------------

// CreateLinkingCode inserts a new one-time linking code. The code expires after
// 10 minutes. The caller is responsible for generating the 6-char code.
func (s *Store) CreateLinkingCode(ctx context.Context, userID, platform, code string) error {
	expiresAt := time.Now().UTC().Add(10 * time.Minute).Format("2006-01-02 15:04:05")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO linking_codes (code, user_id, platform, expires_at) VALUES (?, ?, ?, ?)`,
		code, userID, platform, expiresAt,
	)
	return err
}

// LookupLinkingCode returns an unused linking code by its code string.
func (s *Store) LookupLinkingCode(ctx context.Context, code string) (types.LinkingCode, error) {
	var lc types.LinkingCode
	err := s.db.GetContext(ctx, &lc,
		`SELECT code, user_id, platform, expires_at, COALESCE(used_at, '') AS used_at FROM linking_codes WHERE code = ? AND used_at IS NULL`,
		code,
	)
	return lc, err
}

// LookupLinkingCodeAny returns a linking code regardless of whether it has been
// used. The SSE stream uses this to detect the transition from unused → used
// (LookupLinkingCode filters used_at IS NULL and would miss the transition).
func (s *Store) LookupLinkingCodeAny(ctx context.Context, code string) (types.LinkingCode, error) {
	var lc types.LinkingCode
	err := s.db.GetContext(ctx, &lc,
		`SELECT code, user_id, platform, expires_at, COALESCE(used_at, '') AS used_at FROM linking_codes WHERE code = ?`,
		code,
	)
	return lc, err
}

// ConsumeLinkingCode marks a linking code as used.
func (s *Store) ConsumeLinkingCode(ctx context.Context, code string) error {
	_, err := s.db.ExecContext(ctx,
		s.rewrite(`UPDATE linking_codes SET used_at = `+s.dialect.Now()+` WHERE code = ? AND used_at IS NULL`),
		code,
	)
	return err
}

// ---------------------------------------------------------------------------
// Water tracking
// ---------------------------------------------------------------------------

// LogWater inserts a water consumption entry.
func (s *Store) LogWater(ctx context.Context, w types.WaterLog) error {
	const q = `
		INSERT INTO water_logs (id, user_id, amount_ml, logged_at, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), w.ID, w.UserID, w.AmountML, w.LoggedAt, nullStr(w.Note), utcNow())
	if err != nil {
		return fmt.Errorf("store: log water: %w", err)
	}
	return nil
}

// GetWaterToday returns water logs for a specific local date, along with the
// total ml consumed that day.
func (s *Store) GetWaterToday(ctx context.Context, userID, localDate string) ([]types.WaterLog, int, error) {
	const q = `
		SELECT id, user_id, amount_ml, logged_at, COALESCE(note, '') AS note
		FROM water_logs
		WHERE user_id = ? AND date(logged_at) = ?
		ORDER BY logged_at DESC
	`
	var logs []types.WaterLog
	if err := s.db.SelectContext(ctx, &logs, s.rewrite(q), userID, localDate); err != nil {
		return nil, 0, fmt.Errorf("store: get water today: %w", err)
	}
	total := 0
	for _, w := range logs {
		total += w.AmountML
	}
	return logs, total, nil
}

// GetWaterDailyTotals returns per-day water totals between startDate and endDate
// (inclusive, "YYYY-MM-DD" format). Days with no water logs are not returned.
func (s *Store) GetWaterDailyTotals(ctx context.Context, userID, startDate, endDate string) ([]types.WaterDayTotal, error) {
	const q = `
		SELECT date(logged_at) AS date, SUM(amount_ml) AS total_ml
		FROM water_logs
		WHERE user_id = ? AND date(logged_at) >= ? AND date(logged_at) <= ?
		GROUP BY date(logged_at)
		ORDER BY date(logged_at) ASC
	`
	var out []types.WaterDayTotal
	if err := s.db.SelectContext(ctx, &out, s.rewrite(q), userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("store: get water daily totals: %w", err)
	}
	return out, nil
}

// DeleteWater deletes a water log entry by user + ID. Returns ErrNotFound if absent.
func (s *Store) DeleteWater(ctx context.Context, userID, id string) error {
	const q = `DELETE FROM water_logs WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), id, userID)
	if err != nil {
		return fmt.Errorf("store: delete water: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Workout tracking
// ---------------------------------------------------------------------------

// LogWorkout inserts a workout and its exercises inside a transaction.
func (s *Store) LogWorkout(ctx context.Context, w types.Workout) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const workoutQ = `
		INSERT INTO workouts (id, user_id, name, duration_min, intensity, calories_burned, note, logged_at, created_at)
		VALUES (:id, :user_id, :name, :duration_min, :intensity, :calories_burned, :note, :logged_at, :created_at)
	`
	workoutQuery, workoutArgs, err := sqlx.Named(workoutQ, map[string]any{
		"id": w.ID, "user_id": w.UserID, "name": w.Name, "duration_min": w.DurationMin,
		"intensity": w.Intensity, "calories_burned": w.CaloriesBurned, "note": nullStr(w.Note),
		"logged_at": w.LoggedAt, "created_at": utcNow(),
	})
	if err != nil {
		return fmt.Errorf("store: bind insert workout: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(workoutQuery), workoutArgs...); err != nil {
		return fmt.Errorf("store: insert workout: %w", err)
	}

	const exerciseQ = `
		INSERT INTO workout_exercises (id, workout_id, name, sets, reps, weight_kg, note)
		VALUES (:id, :workout_id, :name, :sets, :reps, :weight_kg, :note)
	`
	for _, e := range w.Exercises {
		exID := e.ID
		if exID == "" {
			exID = newID()
		}
		exerciseQuery, exerciseArgs, err := sqlx.Named(exerciseQ, map[string]any{
			"id": exID, "workout_id": w.ID, "name": e.Name,
			"sets": e.Sets, "reps": e.Reps, "weight_kg": e.WeightKg, "note": nullStr(e.Note),
		})
		if err != nil {
			return fmt.Errorf("store: bind insert exercise: %w", err)
		}
		if _, err := tx.ExecContext(ctx, s.rewrite(exerciseQuery), exerciseArgs...); err != nil {
			return fmt.Errorf("store: insert exercise: %w", err)
		}
	}

	return tx.Commit()
}

// GetWorkout returns a single workout by ID with its exercises populated.
// Returns types.ErrNotFound when the workout does not exist.
func (s *Store) GetWorkout(ctx context.Context, id string) (types.Workout, error) {
	const q = `
		SELECT id, user_id, name, duration_min, intensity, calories_burned, COALESCE(note, '') AS note, logged_at
		FROM workouts WHERE id = ?
	`
	var w types.Workout
	if err := s.db.GetContext(ctx, &w, s.rewrite(q), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.Workout{}, types.ErrNotFound
		}
		return types.Workout{}, fmt.Errorf("store: get workout: %w", err)
	}

	exercises, err := s.loadWorkoutExercises(ctx, id)
	if err != nil {
		return types.Workout{}, err
	}
	w.Exercises = exercises
	return w, nil
}

// ListWorkouts returns the user's most recent workouts without exercises.
func (s *Store) ListWorkouts(ctx context.Context, userID string, limit int) ([]types.Workout, error) {
	const q = `
		SELECT id, user_id, name, duration_min, intensity, calories_burned, COALESCE(note, '') AS note, logged_at
		FROM workouts
		WHERE user_id = ?
		ORDER BY logged_at DESC
		LIMIT ?
	`
	var out []types.Workout
	if err := s.db.SelectContext(ctx, &out, s.rewrite(q), userID, limit); err != nil {
		return nil, fmt.Errorf("store: list workouts: %w", err)
	}
	return out, nil
}

// DeleteWorkout deletes a workout by user + ID. Exercises are cascade-deleted.
// Returns ErrNotFound if absent.
func (s *Store) DeleteWorkout(ctx context.Context, userID, id string) error {
	const q = `DELETE FROM workouts WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), id, userID)
	if err != nil {
		return fmt.Errorf("store: delete workout: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ListWorkoutsInRange returns every workout between startDate and endDate
// (inclusive, "YYYY-MM-DD" format), ordered newest first, with no limit.
func (s *Store) ListWorkoutsInRange(ctx context.Context, userID, startDate, endDate string) ([]types.Workout, error) {
	const q = `
		SELECT id, user_id, name, duration_min, intensity, calories_burned, COALESCE(note, '') AS note, logged_at
		FROM workouts
		WHERE user_id = ? AND date(logged_at) >= ? AND date(logged_at) <= ?
		ORDER BY logged_at DESC
	`
	var out []types.Workout
	if err := s.db.SelectContext(ctx, &out, s.rewrite(q), userID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("store: list workouts in range: %w", err)
	}
	return out, nil
}

func (s *Store) loadWorkoutExercises(ctx context.Context, workoutID string) ([]types.WorkoutExercise, error) {
	const q = `
		SELECT id, workout_id, name, sets, reps, weight_kg, COALESCE(note, '') AS note
		FROM workout_exercises
		WHERE workout_id = ?
		ORDER BY rowid
	`
	var out []types.WorkoutExercise
	if err := s.db.SelectContext(ctx, &out, s.rewrite(q), workoutID); err != nil {
		return nil, fmt.Errorf("store: query exercises: %w", err)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Sleep tracking
// ---------------------------------------------------------------------------

// LogSleep inserts a new sleep log entry.
func (s *Store) LogSleep(ctx context.Context, sl types.SleepLog) error {
	const q = `
		INSERT INTO sleep_logs (id, user_id, sleep_at, wake_at, quality, note, created_at)
		VALUES (:id, :user_id, :sleep_at, :wake_at, :quality, :note, :created_at)
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"id": sl.ID, "user_id": sl.UserID, "sleep_at": sl.SleepAt, "wake_at": sl.WakeAt,
		"quality": sl.Quality, "note": nullStr(sl.Note), "created_at": utcNow(),
	})
	if err != nil {
		return fmt.Errorf("store: bind log sleep: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, s.rewrite(query), args...); err != nil {
		return fmt.Errorf("store: log sleep: %w", err)
	}
	return nil
}

// GetActiveSleep returns the user's in-progress sleep (wake_at IS NULL), or
// ErrNotFound if none is active.
func (s *Store) GetActiveSleep(ctx context.Context, userID string) (*types.SleepLog, error) {
	const q = `
		SELECT id, user_id, sleep_at, wake_at, quality, COALESCE(note, '') AS note
		FROM sleep_logs
		WHERE user_id = ? AND wake_at IS NULL
		ORDER BY sleep_at DESC
		LIMIT 1
	`
	var sl types.SleepLog
	if err := s.db.GetContext(ctx, &sl, s.rewrite(q), userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("store: get active sleep: %w", err)
	}
	return &sl, nil
}

// EndSleep closes a sleep log by setting wake_at and quality. Returns
// ErrNotFound if no matching active sleep log exists.
func (s *Store) EndSleep(ctx context.Context, userID, id, wakeAt, quality string) error {
	const q = `
		UPDATE sleep_logs SET wake_at = ?, quality = ?
		WHERE id = ? AND user_id = ? AND wake_at IS NULL
	`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), wakeAt, quality, id, userID)
	if err != nil {
		return fmt.Errorf("store: end sleep: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// ListSleep returns the user's most recent sleep logs, newest first.
func (s *Store) ListSleep(ctx context.Context, userID string, limit int) ([]types.SleepLog, error) {
	const q = `
		SELECT id, user_id, sleep_at, wake_at, quality, COALESCE(note, '') AS note
		FROM sleep_logs
		WHERE user_id = ?
		ORDER BY sleep_at DESC
		LIMIT ?
	`
	var out []types.SleepLog
	if err := s.db.SelectContext(ctx, &out, s.rewrite(q), userID, limit); err != nil {
		return nil, fmt.Errorf("store: list sleep: %w", err)
	}
	return out, nil
}

// DeleteSleep deletes a sleep log by user + ID. Returns ErrNotFound if absent.
func (s *Store) DeleteSleep(ctx context.Context, userID, id string) error {
	const q = `DELETE FROM sleep_logs WHERE id = ? AND user_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), id, userID)
	if err != nil {
		return fmt.Errorf("store: delete sleep: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}
