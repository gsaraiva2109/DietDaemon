package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/jmoiron/sqlx"
)

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
