# Progress Photo Storage Policy

## Current policy (approved)

Progress photos are stored as raw BLOB/BYTEA rows directly in the `progress_photos`
table — no filesystem path, no S3 or other object storage. Schema (mirrored in both
backends):

```sql
CREATE TABLE IF NOT EXISTS progress_photos (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date       TEXT NOT NULL,
    view       TEXT NOT NULL CHECK(view IN ('front', 'side', 'back')),
    mime_type  TEXT NOT NULL,
    data       BLOB NOT NULL,   -- BYTEA on Postgres
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

See `migrations/sqlite/001_init.sql:324-333` and `migrations/postgres/001_init.sql:362-371`.
`data` is untyped/unsized BLOB/BYTEA — no separate size column, no external reference.

This is the deliberate, approved architecture at current scale — not a stopgap
pending a "real" object-storage backend. Photos live in the same database as
every other user record, back up and restore with it, and require no separate
storage service to operate or reason about.

## Enforced limits

- **5MB per-upload cap** — app-layer, enforced in `handleUploadPhoto`
  (`internal/api/handler_photos.go`) via `http.MaxBytesReader` +
  `ParseMultipartForm` + `io.LimitReader`, all bounded to the same 5MB ceiling.
- **Mime-type allowlist** (`image/jpeg`, `image/png`, `image/webp`) — added in #208.
- **Per-user quota** (100 photos) — added in #208.
- **`view` field CHECK constraint** (`front`/`side`/`back`) — enforced at the SQL
  layer in both migrations above. Defense-in-depth only since #208 validates it
  at the app layer first.

## Access control guarantees

`GetPhotoData` and `GetPhotosData` (`internal/store/store_photos.go:59-72` and
`:79-105`) scope every query by `user_id` at the SQL layer (`WHERE id = ? AND
user_id = ?`), not just at the handler layer — a photo ID belonging to another
user is indistinguishable from an ID that doesn't exist. This is a regression-
tested guarantee: see `TestGetPhotoData_CrossUserAccessDenied` and
`TestGetPhotosData_CrossUserAccessDenied` in
`internal/store/store_photos_test.go`.

`handlePhotoData` (`internal/api/handler_photos.go:28-45`) additionally re-checks
`photo.UserID != userID` after the store call returns. This is redundant given
the SQL-layer scoping above, but kept as defense-in-depth in case the handler is
ever wired to a store implementation that isn't scoped correctly.

## Migration trigger conditions (future, not now)

None of the following are close today. This is a documented trigger list, not a
roadmap commitment — a future migration to S3 or another object store would only
be reconsidered if one of these actually materializes:

1. **Total DB file size** exceeds an operational ceiling comfortable for current
   backup/hosting tooling (single-file SQLite backups, `pg_dump` for Postgres).
2. **Aggregate photo volume**, per-user or instance-wide, makes single-file
   SQLite backups slow or unwieldy in size.
3. **A move to a managed/multi-instance deployment** where a shared local DB
   file (or single Postgres volume) stops being viable and object storage
   becomes the natural shared backend instead.

## docker-compose.yml unaffected

Today's `docker-compose.yml` defines exactly three volumes: `dietdaemon_data`,
`ollama_data`, `postgres_data`. No object-storage service exists, and this doc
does not add one.

## See also

- [docs/BACKUP.md](BACKUP.md) — how progress photo blobs are exported
  (one blob file per photo, plus a `photos.csv` index) during scheduled and
  on-demand backups.
- [docs/RESTORE.md](RESTORE.md) — how those blobs are read back in during restore.
