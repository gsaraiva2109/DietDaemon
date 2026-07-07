package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

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
