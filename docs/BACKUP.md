# Scheduled Backup

DietDaemon can automatically export a user's meals and daily rollups on a recurring
schedule, in addition to the on-demand CSV/JSON export already available from
Settings → Export data. Each user opts in independently and picks where their backup
goes: the server's local disk, or an S3-compatible bucket.

## How it works

```
[Backup runner] --tick (BACKUP_CHECK_INTERVAL)--> for each user with backup_config.enabled
                                                      is (now - last_run_at) >= interval_hrs?
                                                        yes -> export meals.csv + rollups.csv
                                                               -> write to destination
                                                               -> update last_run_at
```

The runner is a second, independent background loop alongside the existing nudge
scheduler — same shape (a ticker + a per-user check), separate concern. It reuses the
exact CSV writer used by the on-demand export endpoint (`internal/exportfmt`), so
scheduled backups and manual exports are byte-identical in format.

A user can also trigger a backup immediately via "Run now" in the dashboard, which
calls the same export logic outside the interval gate.

## Prerequisites

- **Local destination**: a writable directory on the server, set via `BACKUP_LOCAL_DIR`.
- **S3 destination**: an S3-compatible bucket and the [default AWS credential
  chain](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-gosdk.html)
  configured on the host (env vars `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`, a shared
  credentials file, or an instance/task role). Credentials are infrastructure-level
  configuration, never stored per-user — only the bucket, prefix, region, and endpoint
  are per-user settings, so the same server credentials can back up every user to their
  own bucket/prefix.

## Configuration

### Server (environment variables)

| Variable                | Default | Description                                                    |
|-------------------------|---------|----------------------------------------------------------------|
| `BACKUP_LOCAL_DIR`      | (empty) | Base directory for the "local" destination. Empty disables it. |
| `BACKUP_CHECK_INTERVAL` | `1h`    | How often the runner checks which users are due for a backup.  |

### Per-user (Settings → Backup, or `PUT /api/v1/settings/backup`)

| Field         | Applies to | Description                                                  |
|---------------|------------|--------------------------------------------------------------|
| `Enabled`     | both       | Turns scheduled backup on/off for this user.                 |
| `Destination` | both       | `local` or `s3`.                                             |
| `IntervalHrs` | both       | Hours between runs (default 24).                             |
| `LocalSubdir` | local      | Subdirectory under `BACKUP_LOCAL_DIR` for this user's files. |
| `S3Bucket`    | s3         | Target bucket.                                               |
| `S3Prefix`    | s3         | Key prefix (e.g. `dietdaemon-backups/alice`).                |
| `S3Region`    | s3         | Overrides the SDK's default region when set.                 |
| `S3Endpoint`  | s3         | Custom endpoint for S3-compatible stores (e.g. MinIO).       |

`local_subdir` is validated against `BACKUP_LOCAL_DIR` before any file is written — a
value that would escape the base directory (e.g. `../../etc`) is rejected outright.

## API

| Method & path                      | Description                                  |
|------------------------------------|----------------------------------------------|
| `GET /api/v1/settings/backup`      | Read the authenticated user's backup config. |
| `PUT /api/v1/settings/backup`      | Create/update the config.                    |
| `POST /api/v1/settings/backup/run` | Trigger an immediate backup.                 |

## Troubleshooting

### "backup is not enabled on this server"

`POST /api/v1/settings/backup/run` returns this (503) when the server didn't wire up a
backup runner at all — this shouldn't happen in a normal deployment; the runner starts
unconditionally, so check the server logs for a startup error.

### "local destination not configured (set BACKUP_LOCAL_DIR)"

The user's `Destination` is `local` but the operator never set `BACKUP_LOCAL_DIR`. Set
it and restart, or switch the user to `s3`.

### "local_subdir ... escapes base directory"

`LocalSubdir` contains `..` or otherwise resolves outside `BACKUP_LOCAL_DIR`. Use a
plain relative folder name.

### "s3_bucket not configured"

`Destination` is `s3` but `S3Bucket` is empty. Set it via `PUT /api/v1/settings/backup`.

### S3 uploads fail with a credentials error

The AWS default credential chain found nothing. Set `AWS_ACCESS_KEY_ID` /
`AWS_SECRET_ACCESS_KEY` (or attach an instance/task role) to the DietDaemon host/container.
