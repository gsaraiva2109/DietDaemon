# Restore

`cmd/restore` is the read-side counterpart to [scheduled backup](BACKUP.md): it reads
the CSV/blob files a backup wrote (local disk or S3) and replays them into a store,
entity by entity.

Because this is disaster-recovery tooling, the target `-db` is assumed to be
empty or fresh — restore never reads its configuration (destination, bucket,
subdir, ...) from the target database's `backup_config` table the way the
scheduled backup runner does. Every setting comes from CLI flags instead, so
restore works even against a completely blank database.

## Usage

### Local disk

```
go run ./cmd/restore \
  -user <user-id> \
  -db ./data/dietdaemon.db \
  -destination local \
  -dir ./backups \
  -subdir alice
```

### S3

```
go run ./cmd/restore \
  -user <user-id> \
  -db ./data/dietdaemon.db \
  -destination s3 \
  -s3-bucket my-bucket \
  -s3-prefix dietdaemon-backups/alice \
  -s3-region us-east-1
```

S3 credentials come from the [default AWS credential
chain](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-gosdk.html)
(env vars, shared config, instance/task role) — same as scheduled backup, never
passed as a flag.

### Dry run

List the files a backup contains without touching the store:

```
go run ./cmd/restore \
  -user <user-id> \
  -db ./data/dietdaemon.db \
  -destination local \
  -dir ./backups \
  -dry-run
```

In `-dry-run` mode the target database is never opened — the flag is validated but
the store isn't touched, so `-db` doesn't even need to point to a real file.

### Flags

| Flag            | Required | Description                                                            |
|-----------------|----------|--------------------------------------------------------------------------|
| `-user`         | yes      | User ID to restore into.                                                 |
| `-db`           | yes      | SQLite database path for the target store.                               |
| `-destination`  | yes      | `local` or `s3`.                                                         |
| `-dir`          | local    | Local disk base directory holding the backup (`localdisk.New`'s baseDir). |
| `-subdir`       | local    | Backup subdirectory the user's files live under (`BackupConfig.LocalSubdir`). |
| `-s3-bucket`    | s3       | S3 bucket holding the backup.                                            |
| `-s3-prefix`    | s3       | S3 key prefix the user's files live under.                               |
| `-s3-region`    | s3       | S3 region override.                                                      |
| `-s3-endpoint`  | s3       | S3-compatible endpoint override (e.g. MinIO).                            |
| `-dry-run`      | no       | List backup files found without touching the store.                      |

## What's preserved

| Entity            | Preserved? | Notes |
|--------------------|------------|-------|
| Meals              | **Lossy**  | `meals.csv` only carries meal-level macro totals and a date, not the per-item breakdown or time-of-day. Each restored meal becomes a single synthetic line-item whose macros equal the row's totals, timestamped at midnight UTC on the recorded date. This is a limitation of the existing `meals.csv` export format (already used by production backups), not something this restore path can fix without breaking already-taken backups. |
| Daily rollups      | Fully preserved | Consumed and target macros round-trip exactly. |
| Weight             | Fully preserved | ID, date, weight, note round-trip exactly. |
| Body measurements  | Fully preserved | ID, date, and all measurement fields round-trip exactly. |
| Sleep              | Fully preserved | Including `wake_at` for completed sleep sessions. |
| Workouts           | Fully preserved | Including individual exercises (sets/reps/weight), round-tripped via the `exercises_json` column. |
| Water              | Fully preserved | Amount, timestamp, note round-trip exactly. |
| Fasting            | Fully preserved | Including end time and completion state for closed fasts. |
| Progress photos    | Fully preserved | Blob data plus metadata (date, view, mime type). |

## Idempotency

Restore is safe to run more than once against the same backup: every store write it
uses reuses the backup row's original ID, so a re-insert of a row that's already
present hits a unique-constraint violation and is treated as a no-op rather than an
error. Weight, measurements, and daily rollups go through the same upsert-by-date
logic scheduled backups already use, so re-running restore overwrites those rows with
identical values instead of duplicating them. A partial or interrupted restore can
always be re-run from the start without fear of duplicate data.

## Out of scope

Like scheduled backup, this restores application-level data (the 9 entities above)
from CSV/blob files — it is not a database-level restore. Recovering from a full
`pg_dump` or a raw SQLite file copy is out of scope for this tool; self-hosters who
took a full point-in-time snapshot should restore it directly via `sqlite3 <path>
.restore <backup>` or `pg_restore`/`psql` against their database.
