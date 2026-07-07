ALTER TABLE backup_config ADD COLUMN last_meals_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE backup_config ADD COLUMN last_rollups_count INTEGER NOT NULL DEFAULT 0;
