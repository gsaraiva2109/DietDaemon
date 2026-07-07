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
