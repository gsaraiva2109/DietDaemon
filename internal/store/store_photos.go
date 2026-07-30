package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// maxPhotoRows caps ListPhotoMetadata as a safety ceiling, not pagination —
// audited as safe at this project's current scale.
const maxPhotoRows = 10000

// ListPhotoMetadata returns progress photo records without the BLOB data.
func (s *Store) ListPhotoMetadata(ctx context.Context, userID string) ([]types.ProgressPhoto, error) {
	q := fmt.Sprintf(`
		SELECT id, user_id, date, view, mime_type, created_at
		FROM progress_photos WHERE user_id = ?
		ORDER BY date DESC
		LIMIT %d
	`, maxPhotoRows)
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

// GetPhotoData returns a single progress photo including BLOB data. Scoped to
// userID at the SQL layer: a photoID belonging to another user returns
// ErrNotFound, the same as a photoID that doesn't exist.
func (s *Store) GetPhotoData(ctx context.Context, userID, photoID string) (types.ProgressPhoto, error) {
	const q = `
		SELECT id, user_id, date, view, mime_type, data, created_at
		FROM progress_photos WHERE id = ? AND user_id = ?
	`
	var row photoRow
	if err := s.db.GetContext(ctx, &row, s.rewrite(q), photoID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ProgressPhoto{}, types.ErrNotFound
		}
		return types.ProgressPhoto{}, fmt.Errorf("store: get photo data: %w", err)
	}
	return row.toProgressPhoto(), nil
}

// GetPhotosData batch-fetches BLOB data for multiple photo IDs in a single
// query, returning a map keyed by photo ID. Scoped to userID at the SQL
// layer: IDs belonging to another user are treated the same as unknown IDs —
// simply absent from the map (no error). Returns an empty map for an empty
// input, without touching the DB.
func (s *Store) GetPhotosData(ctx context.Context, userID string, photoIDs []string) (map[string][]byte, error) {
	out := make(map[string][]byte, len(photoIDs))
	if len(photoIDs) == 0 {
		return out, nil
	}

	placeholders := make([]string, len(photoIDs))
	args := make([]any, 0, len(photoIDs)+1)
	for i, id := range photoIDs {
		placeholders[i] = s.dialect.Placeholder(i + 1)
		args = append(args, id)
	}
	args = append(args, userID)

	// #nosec G201 -- placeholder expansion is ? only, values are args
	q := fmt.Sprintf(`SELECT id, data FROM progress_photos WHERE id IN (%s) AND user_id = %s`,
		strings.Join(placeholders, ","), s.dialect.Placeholder(len(photoIDs)+1))

	var rows []photoDataRow
	if err := s.db.SelectContext(ctx, &rows, s.rewrite(q), args...); err != nil {
		return nil, fmt.Errorf("store: get photos data: %w", err)
	}
	for _, r := range rows {
		out[r.ID] = r.Data
	}
	return out, nil
}

// photoDataRow is the flat DB shape for the id+data-only batch fetch used by
// GetPhotosData.
type photoDataRow struct {
	ID   string `db:"id"`
	Data []byte `db:"data"`
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
