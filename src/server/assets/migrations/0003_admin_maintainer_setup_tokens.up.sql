ALTER TABLE maintainers
    DROP CONSTRAINT IF EXISTS maintainers_hashed_password_key,
    ADD COLUMN IF NOT EXISTS setup_token_hash TEXT,
    ADD COLUMN IF NOT EXISTS setup_token_expiration_date TIMESTAMPTZ;

DROP TABLE IF EXISTS pending_registrations;
DROP TABLE IF EXISTS pending_email_changes;
DROP TABLE IF EXISTS disabled_maintainers;
