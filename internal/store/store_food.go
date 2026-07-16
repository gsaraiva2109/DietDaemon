package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/normalize"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Global food catalog + per-user usage stats
// ---------------------------------------------------------------------------

// LookupFood matches phrase against the user's personal aliases and joins to
// the global food catalog. Returns types.ErrNoMatch when no alias matches.
func (s *Store) LookupFood(ctx context.Context, userID, phrase string) (types.FoodMatch, error) {
	normalized := normalize.Normalize(phrase)

	const q = `
		SELECT f.food_id, f.name, f.source, f.kcal_100g, f.protein_100g,
		       f.carbs_100g, f.fat_100g, f.fiber_100g
		FROM food_aliases fa
		JOIN foods f ON f.food_id = fa.food_id
		WHERE fa.user_id = ? AND fa.alias_normalized = ?
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

// GetFood loads a food from the global catalog by its food_id. Returns
// types.ErrNoMatch when the food does not exist.
func (s *Store) GetFood(ctx context.Context, foodID string) (types.FoodMatch, error) {
	const q = `
		SELECT food_id, name, source, kcal_100g, protein_100g,
		       carbs_100g, fat_100g, fiber_100g,
		       category, brand, barcode, image_url, serving_size, serving_unit
		FROM foods
		WHERE food_id = ?
	`
	var row foodMatchRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), foodID); err != nil {
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
	FoodID      string  `db:"food_id"`
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
}

func (r foodMatchRow) toFoodMatch() types.FoodMatch {
	return types.FoodMatch{
		FoodID: r.FoodID,
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
	}
}

// ListFoodsWithoutVectors returns every food in the global catalog that has
// no row in food_vectors yet, e.g. rows written by a bulk catalog import
// (which never calls EmbedFood) rather than the live resolver's
// embedding-on-write path. Used by the backfill maintenance operation to make
// the whole catalog embedding-matchable, not just accidentally-touched foods.
func (s *Store) ListFoodsWithoutVectors(ctx context.Context) ([]types.FoodMatch, error) {
	const q = `
		SELECT f.food_id, f.name, f.source, f.kcal_100g, f.protein_100g,
		       f.carbs_100g, f.fat_100g, f.fiber_100g,
		       f.category, f.brand, f.barcode, f.image_url, f.serving_size, f.serving_unit
		FROM foods f
		LEFT JOIN food_vectors fv ON fv.food_id = f.food_id
		WHERE fv.food_id IS NULL
	`
	var rows []foodMatchRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q)); err != nil {
		return nil, fmt.Errorf("store: list foods without vectors: %w", err)
	}
	matches := make([]types.FoodMatch, len(rows))
	for i, r := range rows {
		matches[i] = r.toFoodMatch()
	}
	return matches, nil
}

// UpsertFood writes a resolved food into the global catalog (shared by every
// user — a food's name/source/macros are resolved once, ever, regardless of
// how many users log it), ensures a per-user usage-stats row exists, and adds
// any new normalized aliases for this user, all within a single transaction.
func (s *Store) UpsertFood(ctx context.Context, userID string, match types.FoodMatch, aliases []string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	foodQuery, foodArgs, err := sqlx.Named(foodUpsertQuery, foodNamedArgs(match))
	if err != nil {
		return fmt.Errorf("store: bind upsert food: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.rewrite(foodQuery), foodArgs...); err != nil {
		return fmt.Errorf("store: upsert food: %w", err)
	}

	const statsQ = `
		INSERT INTO user_food_stats (user_id, food_id, query_count, last_used)
		VALUES (?, ?, 0, '')
		ON CONFLICT(user_id, food_id) DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, s.rewrite(statsQ), userID, match.FoodID); err != nil {
		return fmt.Errorf("store: ensure user food stats: %w", err)
	}

	const aliasQ = `
		INSERT INTO food_aliases (user_id, alias_normalized, food_id)
		VALUES `
	const aliasSuffix = ` ON CONFLICT DO NOTHING`
	rows := make([][]any, 0, len(aliases))
	for _, alias := range aliases {
		normalized := normalize.Normalize(alias)
		if normalized != "" {
			rows = append(rows, []any{userID, normalized, match.FoodID})
		}
	}
	if err := s.insertRows(ctx, tx, aliasQ, aliasSuffix, rows); err != nil {
		return fmt.Errorf("store: insert aliases: %w", err)
	}

	return tx.Commit()
}

// RecordFoodQuery bumps this user's query_count and last_used for a food,
// creating the usage-stats row if it doesn't exist yet (e.g. the first time
// this user matches a food another user already resolved).
func (s *Store) RecordFoodQuery(ctx context.Context, userID, foodID string) error {
	const q = `
		INSERT INTO user_food_stats (user_id, food_id, query_count, last_used)
		VALUES (?, ?, 1, ?)
		ON CONFLICT(user_id, food_id) DO UPDATE SET
			query_count = user_food_stats.query_count + 1,
			last_used   = excluded.last_used
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), userID, foodID, utcNow())
	return err
}

// foodUpsertQuery is the shared global-catalog upsert used by UpsertFood
// (per-user transaction) and BulkUpsertFoods (store_food_bulk.go, global-only
// batches). Keeping it package-level lets both reuse identical SQL.
const foodUpsertQuery = `
	INSERT INTO foods
		(food_id, name, source, kcal_100g, protein_100g, carbs_100g, fat_100g, fiber_100g,
		 category, brand, barcode, image_url, serving_size, serving_unit, created_at, updated_at)
	VALUES (:food_id, :name, :source, :kcal_100g, :protein_100g, :carbs_100g, :fat_100g, :fiber_100g,
	        :category, :brand, :barcode, :image_url, :serving_size, :serving_unit, :now, :now)
	ON CONFLICT(food_id) DO UPDATE SET
		name         = excluded.name,
		source       = excluded.source,
		kcal_100g    = excluded.kcal_100g,
		protein_100g = excluded.protein_100g,
		carbs_100g   = excluded.carbs_100g,
		fat_100g     = excluded.fat_100g,
		fiber_100g   = excluded.fiber_100g,
		category     = excluded.category,
		brand        = excluded.brand,
		barcode      = excluded.barcode,
		image_url    = excluded.image_url,
		serving_size = excluded.serving_size,
		serving_unit = excluded.serving_unit,
		updated_at   = excluded.updated_at
`

// foodNamedArgs builds the named-parameter map for every upsert of a global
// foods row (UpsertFood, BulkUpsertFoods, CorrectMealItem's cache refresh).
func foodNamedArgs(match types.FoodMatch) map[string]any {
	return map[string]any{
		"food_id":      match.FoodID,
		"name":         match.Name,
		"source":       match.Source,
		"kcal_100g":    match.Per100g.Calories,
		"protein_100g": match.Per100g.Protein,
		"carbs_100g":   match.Per100g.Carbs,
		"fat_100g":     match.Per100g.Fat,
		"fiber_100g":   match.Per100g.Fiber,
		"category":     match.Category,
		"brand":        match.Brand,
		"barcode":      match.Barcode,
		"image_url":    match.ImageURL,
		"serving_size": match.ServingSize,
		"serving_unit": match.ServingUnit,
		"now":          utcNow(),
	}
}

// ---------------------------------------------------------------------------
// Food discovery — always scoped to foods this user has personally used
// (i.e. has a user_food_stats row for), joined against the global catalog.
// ---------------------------------------------------------------------------

// ListFoods returns paginated food entries this user has used, optionally
// filtered by source.
func (s *Store) ListFoods(ctx context.Context, userID, source string, limit, offset int) ([]types.FoodDetail, error) {
	args := []any{userID}
	q := `
		SELECT f.food_id, ufs.user_id, f.name, f.source, f.kcal_100g, f.protein_100g, f.carbs_100g, f.fat_100g, f.fiber_100g,
		       f.category, f.brand, f.barcode, f.image_url, f.serving_size, f.serving_unit,
		       ufs.query_count, ufs.last_used
		FROM user_food_stats ufs
		JOIN foods f ON f.food_id = ufs.food_id
		WHERE ufs.user_id = ?`
	if source != "" {
		q += ` AND f.source = ?`
		args = append(args, source)
	}
	q += ` ORDER BY ufs.last_used DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	var rows []foodDetailRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), args...); err != nil {
		return nil, fmt.Errorf("store: list foods: %w", err)
	}
	return foodDetailRows(rows), nil
}

// SearchFoods searches foods this user has used by name and alias using
// full-text search. The search index itself spans the global catalog plus
// every user's personal aliases (food_search.user_id = ” for global rows),
// but results are still restricted to foods this user has a stats row for.
func (s *Store) SearchFoods(ctx context.Context, userID, query string) ([]types.FoodDetail, error) {
	searchParam := s.dialect.SearchQuery(query)

	var q string
	if s.driver == "postgres" {
		q = `
		SELECT f.food_id, ufs.user_id, f.name, f.source,
		       f.kcal_100g, f.protein_100g, f.carbs_100g, f.fat_100g, f.fiber_100g,
		       f.category, f.brand, f.barcode, f.image_url, f.serving_size, f.serving_unit,
		       ufs.query_count, ufs.last_used
		FROM user_food_stats ufs
		JOIN foods f ON f.food_id = ufs.food_id
		WHERE ufs.user_id = ? AND ufs.food_id IN (
			SELECT fs.food_id FROM food_search fs
			WHERE fs.tsv @@ to_tsquery('simple', ?) AND (fs.user_id = '' OR fs.user_id = ?)
		)
		ORDER BY ufs.query_count DESC
		LIMIT 20
	`
	} else {
		q = `
		SELECT f.food_id, ufs.user_id, f.name, f.source,
		       f.kcal_100g, f.protein_100g, f.carbs_100g, f.fat_100g, f.fiber_100g,
		       f.category, f.brand, f.barcode, f.image_url, f.serving_size, f.serving_unit,
		       ufs.query_count, ufs.last_used
		FROM user_food_stats ufs
		JOIN foods f ON f.food_id = ufs.food_id
		WHERE ufs.user_id = ? AND ufs.food_id IN (
			SELECT fs.food_id FROM food_search fs
			WHERE food_search MATCH ? AND (fs.user_id = '' OR fs.user_id = ?)
		)
		ORDER BY ufs.query_count DESC
		LIMIT 20
	`
	}
	var rows []foodDetailRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID, searchParam, userID); err != nil {
		return nil, fmt.Errorf("store: search foods: %w", err)
	}
	return foodDetailRows(rows), nil
}

// SearchCatalog browses the global food catalog directly (unlike ListFoods/
// SearchFoods/FrequentFoods, it is NOT scoped to foods this user has used):
// every food ever bulk-imported or resolved is visible. It LEFT JOINs
// user_food_stats so callers can tell which results are already in the
// user's personal library (in_library, query_count, last_used), defaulting
// those to false/0/"" for catalog-only foods.
func (s *Store) SearchCatalog(ctx context.Context, userID, query, source string, limit, offset int) ([]types.FoodDetail, error) {
	args := []any{userID}
	q := `
		SELECT f.food_id, f.name, f.source, f.kcal_100g, f.protein_100g, f.carbs_100g, f.fat_100g, f.fiber_100g,
		       f.category, f.brand, f.barcode, f.image_url, f.serving_size, f.serving_unit,
		       COALESCE(ufs.query_count, 0) AS query_count, COALESCE(ufs.last_used, '') AS last_used,
		       (ufs.food_id IS NOT NULL) AS in_library
		FROM foods f
		LEFT JOIN user_food_stats ufs ON ufs.user_id = ? AND ufs.food_id = f.food_id
		WHERE 1 = 1`

	if query != "" {
		searchParam := s.dialect.SearchQuery(query)
		if s.driver == "postgres" {
			q += ` AND f.food_id IN (
				SELECT fs.food_id FROM food_search fs
				WHERE fs.tsv @@ to_tsquery('simple', ?) AND fs.user_id = ''
			)`
		} else {
			q += ` AND f.food_id IN (
				SELECT fs.food_id FROM food_search fs
				WHERE food_search MATCH ? AND fs.user_id = ''
			)`
		}
		args = append(args, searchParam)
	}
	if source != "" {
		q += ` AND f.source = ?`
		args = append(args, source)
	}
	if query != "" {
		q += ` ORDER BY query_count DESC, f.name ASC`
	} else {
		q += ` ORDER BY f.name ASC`
	}
	q += ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	var rows []catalogRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), args...); err != nil {
		return nil, fmt.Errorf("store: search catalog: %w", err)
	}
	out := make([]types.FoodDetail, len(rows))
	for i, r := range rows {
		fd := r.foodDetailRow.toFoodDetail()
		fd.UserID = userID
		fd.InLibrary = r.InLibrary
		out[i] = fd
	}
	return out, nil
}

// RemoveFromLibrary deletes this user's usage-stats row for a food, hiding it
// from their personal library view. It does not touch the global foods row,
// food_aliases, or meal history — the food silently reappears in the
// library next time the user logs it (UpsertFood re-inserts the stats row).
// Returns types.ErrNotFound if no row was deleted.
func (s *Store) RemoveFromLibrary(ctx context.Context, userID, foodID string) error {
	const q = `DELETE FROM user_food_stats WHERE user_id = ? AND food_id = ?`
	res, err := s.db.ExecContext(ctx, s.rewrite(q), userID, foodID)
	if err != nil {
		return fmt.Errorf("store: remove from library: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return types.ErrNotFound
	}
	return nil
}

// AddToLibrary adds a catalog food to this user's personal library without
// logging a meal — e.g. a component the user only ever eats as part of a
// combo (grilled chicken + bread) and wants available for quick-log/search
// ahead of time. Idempotent: does nothing if the food is already in the
// library. Returns types.ErrNotFound if foodID doesn't exist in the catalog.
func (s *Store) AddToLibrary(ctx context.Context, userID, foodID string) error {
	if _, err := s.GetFood(ctx, foodID); err != nil {
		return err
	}
	const q = `
		INSERT INTO user_food_stats (user_id, food_id, query_count, last_used)
		VALUES (?, ?, 0, '')
		ON CONFLICT(user_id, food_id) DO NOTHING
	`
	if _, err := s.db.ExecContext(ctx, s.rewrite(q), userID, foodID); err != nil {
		return fmt.Errorf("store: add to library: %w", err)
	}
	return nil
}

// FrequentFoods returns this user's most frequently logged foods.
func (s *Store) FrequentFoods(ctx context.Context, userID string, limit int) ([]types.FoodDetail, error) {
	const q = `
		SELECT f.food_id, ufs.user_id, f.name, f.source,
		       f.kcal_100g, f.protein_100g, f.carbs_100g, f.fat_100g, f.fiber_100g,
		       f.category, f.brand, f.barcode, f.image_url, f.serving_size, f.serving_unit,
		       ufs.query_count, ufs.last_used
		FROM user_food_stats ufs
		JOIN foods f ON f.food_id = ufs.food_id
		WHERE ufs.user_id = ? AND ufs.query_count > 0
		ORDER BY ufs.query_count DESC, ufs.last_used DESC
		LIMIT ?
	`
	var rows []foodDetailRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID, limit); err != nil {
		return nil, fmt.Errorf("store: frequent foods: %w", err)
	}
	return foodDetailRows(rows), nil
}

// GetFoodDetail returns a single food this user has used, with its aliases.
func (s *Store) GetFoodDetail(ctx context.Context, userID, foodID string) (types.FoodDetail, error) {
	const foodQ = `
		SELECT f.food_id, ufs.user_id, f.name, f.source,
		       f.kcal_100g, f.protein_100g, f.carbs_100g, f.fat_100g, f.fiber_100g,
		       f.category, f.brand, f.barcode, f.image_url, f.serving_size, f.serving_unit,
		       ufs.query_count, ufs.last_used
		FROM user_food_stats ufs
		JOIN foods f ON f.food_id = ufs.food_id
		WHERE ufs.user_id = ? AND ufs.food_id = ?
	`
	var row foodDetailRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(foodQ), userID, foodID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.FoodDetail{}, types.ErrNotFound
		}
		return types.FoodDetail{}, fmt.Errorf("store: get food detail: %w", err)
	}
	fd := row.toFoodDetail()
	fd.InLibrary = true // this path only succeeds via the user_food_stats join

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
		INSERT INTO pending_aliases (id, user_id, phrase, food_id, match_score, replacement, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), newID(), userID, phrase, foodID, matchScore, false, utcStr(time.Now()))
	if err != nil {
		return fmt.Errorf("store: add pending alias: %w", err)
	}
	return nil
}

// ListPendingAliases returns every alias candidate awaiting confirmation for a user.
func (s *Store) ListPendingAliases(ctx context.Context, userID string) ([]types.PendingAlias, error) {
	const q = `
		SELECT id, user_id, phrase, food_id, match_score, replacement, created_at
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
		Replace    bool    `db:"replacement"`
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
			MatchScore: r.MatchScore, Replace: r.Replace, CreatedAt: parseUTC(r.CreatedAt),
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

	const selectQ = `SELECT phrase, food_id, replacement FROM pending_aliases WHERE id = ? AND user_id = ?`
	var pending struct {
		Phrase  string `db:"phrase"`
		FoodID  string `db:"food_id"`
		Replace bool   `db:"replacement"`
	}
	if err := tx.GetContext(ctx, &pending, s.rewrite(selectQ), id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrNotFound
		}
		return fmt.Errorf("store: load pending alias: %w", err)
	}

	aliasQ := `INSERT INTO food_aliases (user_id, alias_normalized, food_id) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`
	if pending.Replace {
		aliasQ = `INSERT INTO food_aliases (user_id, alias_normalized, food_id) VALUES (?, ?, ?) ON CONFLICT(user_id, alias_normalized) DO UPDATE SET food_id = excluded.food_id`
	}
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
	var out []string
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

// foodDetailRow is the flat DB shape of a foods+user_food_stats join;
// types.FoodDetail nests the macro columns into Per100g and carries a
// non-column Aliases slice populated separately (GetFoodDetail).
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

// catalogRow is the flat DB shape of SearchCatalog's foods+LEFT JOIN
// user_food_stats query; embeds foodDetailRow (minus its UserID column,
// which SearchCatalog fills in from the query param instead) plus the
// LEFT JOIN presence flag.
type catalogRow struct {
	foodDetailRow
	InLibrary bool `db:"in_library"`
}

func foodDetailRows(rows []foodDetailRow) []types.FoodDetail {
	out := make([]types.FoodDetail, len(rows))
	for i, r := range rows {
		out[i] = r.toFoodDetail()
	}
	return out
}
