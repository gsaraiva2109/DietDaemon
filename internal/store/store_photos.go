package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

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

// RestorePhoto inserts a progress photo for backup restore. On a
// unique-constraint violation (duplicate id — the re-run-safety case), the
// call is a safe no-op and returns nil rather than an error.
func (s *Store) RestorePhoto(ctx context.Context, p types.ProgressPhoto) error {
	const q = `
		INSERT INTO progress_photos (id, user_id, date, view, mime_type, data, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, s.rewrite(q), p.ID, p.UserID, p.Date, p.View, p.MimeType, p.Data, utcStr(p.CreatedAt))
	if err != nil {
		if isUniqueViolation(err) {
			return nil // safe no-op: already restored
		}
		return fmt.Errorf("store: restore photo: %w", err)
	}
	return nil
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
