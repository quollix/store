ALTER TABLE versions
	ADD COLUMN is_migration_checkpoint BOOLEAN NOT NULL DEFAULT FALSE,
	ADD COLUMN download_count BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS versions_app_id_migration_checkpoint_created_asc
	ON versions (app_id, creation_timestamp ASC)
	WHERE is_migration_checkpoint;
