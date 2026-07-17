# Upgrading

The Compose deployment is pinned to a specific DietDaemon release tag. Choose the
release you want deliberately, then update the `dietdaemon` image tag in
`docker-compose.yml`; do not replace it with `latest`.

## Before upgrading

Take a backup and make sure you can access it before changing versions. Follow the
[backup guidance](BACKUP.md) for scheduled backups or export data from Settings.

## Upgrade

After setting the image tag to the chosen release, pull and restart only DietDaemon:

```bash
docker compose pull dietdaemon
docker compose up -d --no-deps dietdaemon
```

## Roll back

If the upgrade needs to be undone, restore any earlier release tag in
`docker-compose.yml`, then run the same commands:

```bash
docker compose pull dietdaemon
docker compose up -d --no-deps dietdaemon
```

Keep the pre-upgrade backup available as the recovery point while you verify the
rolled-back deployment.
