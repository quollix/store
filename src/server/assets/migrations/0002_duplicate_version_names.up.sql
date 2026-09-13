CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE versions
    DROP CONSTRAINT IF EXISTS versions_app_id_version_name_key,
    ADD COLUMN IF NOT EXISTS content_sha256 BYTEA;

UPDATE versions
SET content_sha256 = digest(data, 'sha256')
WHERE content_sha256 IS NULL;

ALTER TABLE versions
    ALTER COLUMN content_sha256 SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS versions_content_sha256_key ON versions (content_sha256);
CREATE INDEX IF NOT EXISTS versions_app_id_version_name_created_desc ON versions (app_id, version_name, creation_timestamp DESC);
