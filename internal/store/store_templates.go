package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// Meal templates
// ---------------------------------------------------------------------------

// SaveTemplate inserts or upserts a meal template and its items.
func (s *Store) SaveTemplate(ctx context.Context, t types.MealTemplate) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const q = `
		INSERT INTO meal_templates (id, user_id, name, created_at, last_used)
		VALUES (:id, :user_id, :name, :created_at, :last_used)
		ON CONFLICT(id) DO UPDATE SET
			name      = excluded.name,
			last_used = excluded.last_used
	`
	query, args, err := sqlx.Named(q, map[string]any{
		"id": t.ID, "user_id": t.UserID, "name": t.Name,
		"created_at": utcStr(t.CreatedAt), "last_used": utcStr(t.LastUsed),
	})
	if err != nil {
		return fmt.Errorf("store: bind save template: %w", err)
	}
	if _, err = tx.ExecContext(ctx, s.rewrite(query), args...); err != nil {
		return fmt.Errorf("store: insert template: %w", err)
	}

	const delQ = `DELETE FROM meal_template_items WHERE template_id = ?`
	if _, err = tx.ExecContext(ctx, s.rewrite(delQ), t.ID); err != nil {
		return fmt.Errorf("store: delete template items: %w", err)
	}

	const itemPrefix = `
		INSERT INTO meal_template_items
			(id, template_id, position, raw_phrase, quantity, unit, normalized_grams,
			 food_id, food_name, source, match_score,
			 kcal, protein, carbs, fat, fiber)
		VALUES `
	rows := make([][]any, 0, len(t.Items))
	for i, it := range t.Items {
		rows = append(rows, templateItemValues(newID(), t.ID, i, it))
	}
	if err := s.insertRows(ctx, tx, itemPrefix, "", rows); err != nil {
		return fmt.Errorf("store: insert template items: %w", err)
	}

	return tx.Commit()
}

// GetTemplates returns all templates for a user, newest first.
func (s *Store) GetTemplates(ctx context.Context, userID string) ([]types.MealTemplate, error) {
	const q = `
		SELECT id, user_id, name, created_at, last_used
		FROM meal_templates WHERE user_id = ?
		ORDER BY created_at DESC
	`
	var rows []struct {
		ID        string `db:"id"`
		UserID    string `db:"user_id"`
		Name      string `db:"name"`
		CreatedAt string `db:"created_at"`
		LastUsed  string `db:"last_used"`
	}
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), userID); err != nil {
		return nil, fmt.Errorf("store: get templates: %w", err)
	}

	out := make([]types.MealTemplate, 0, len(rows))
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, types.MealTemplate{
			ID: r.ID, UserID: r.UserID, Name: r.Name,
			CreatedAt: parseUTC(r.CreatedAt), LastUsed: parseUTC(r.LastUsed),
		})
		ids = append(ids, r.ID)
	}
	if len(out) == 0 {
		return out, nil
	}

	itemsByTemplate, err := s.loadTemplateItems(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		items := itemsByTemplate[out[i].ID]
		if items == nil {
			items = []types.ResolvedItem{}
		}
		out[i].Items = items
	}

	return out, nil
}

// GetTemplate returns a single template by ID.
func (s *Store) GetTemplate(ctx context.Context, templateID string) (types.MealTemplate, error) {
	const q = `
		SELECT id, user_id, name, created_at, last_used
		FROM meal_templates WHERE id = ?
	`
	var row struct {
		ID        string `db:"id"`
		UserID    string `db:"user_id"`
		Name      string `db:"name"`
		CreatedAt string `db:"created_at"`
		LastUsed  string `db:"last_used"`
	}
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), templateID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.MealTemplate{}, types.ErrNotFound
		}
		return types.MealTemplate{}, fmt.Errorf("store: get template: %w", err)
	}

	t := types.MealTemplate{
		ID: row.ID, UserID: row.UserID, Name: row.Name,
		CreatedAt: parseUTC(row.CreatedAt), LastUsed: parseUTC(row.LastUsed),
	}

	const itemQ = `
		SELECT template_id, raw_phrase, quantity, unit, normalized_grams,
		       food_id, food_name, source, match_score,
		       kcal, protein, carbs, fat, fiber
		FROM meal_template_items
		WHERE template_id = ?
		ORDER BY position
	`
	var itemRows []templateItemRow
	if err := s.db.SelectContext(ctx, &itemRows, s.rewrite(itemQ), templateID); err != nil {
		return types.MealTemplate{}, fmt.Errorf("store: load template items: %w", err)
	}
	t.Items = make([]types.ResolvedItem, 0, len(itemRows))
	for _, ir := range itemRows {
		t.Items = append(t.Items, ir.toResolvedItem())
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// templateItemRow is the flat DB shape of meal_template_items.
type templateItemRow struct {
	TemplateID      string  `db:"template_id"`
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

func (r templateItemRow) toResolvedItem() types.ResolvedItem {
	macros := types.Macros{Calories: r.Kcal, Protein: r.Protein, Carbs: r.Carbs, Fat: r.Fat, Fiber: r.Fiber}
	return types.ResolvedItem{
		Parsed: types.ParsedItem{
			RawPhrase: r.RawPhrase, Quantity: r.Quantity, Unit: r.Unit, NormalizedGrams: r.NormalizedGrams,
		},
		Match: types.FoodMatch{
			FoodID: r.FoodID, Name: r.FoodName, Source: r.Source, MatchScore: r.MatchScore,
			Per100g: macrosPer100g(macros, r.NormalizedGrams),
		},
		Macros: macros,
	}
}

func templateItemValues(id, templateID string, position int, it types.ResolvedItem) []any {
	return []any{
		id, templateID, position, it.Parsed.RawPhrase, it.Parsed.Quantity, it.Parsed.Unit, it.Parsed.NormalizedGrams,
		it.Match.FoodID, it.Match.Name, it.Match.Source, it.Match.MatchScore,
		it.Macros.Calories, it.Macros.Protein, it.Macros.Carbs, it.Macros.Fat, it.Macros.Fiber,
	}
}

// loadTemplateItems fetches all meal_template_items for the given template IDs,
// grouped by template_id.
func (s *Store) loadTemplateItems(ctx context.Context, templateIDs []string) (map[string][]types.ResolvedItem, error) {
	if len(templateIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(templateIDs))
	args := make([]any, len(templateIDs))
	for i, id := range templateIDs {
		placeholders[i] = s.dialect.Placeholder(i + 1)
		args[i] = id
	}

	// #nosec G201 -- placeholder expansion is ? only, values are args
	q := fmt.Sprintf(`
		SELECT template_id, raw_phrase, quantity, unit, normalized_grams,
		       food_id, food_name, source, match_score,
		       kcal, protein, carbs, fat, fiber
		FROM meal_template_items
		WHERE template_id IN (%s)
		ORDER BY template_id, position
	`, strings.Join(placeholders, ","))

	var rows []templateItemRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), args...); err != nil {
		return nil, fmt.Errorf("store: query template items: %w", err)
	}

	out := make(map[string][]types.ResolvedItem)
	for _, r := range rows {
		out[r.TemplateID] = append(out[r.TemplateID], r.toResolvedItem())
	}
	return out, nil
}
