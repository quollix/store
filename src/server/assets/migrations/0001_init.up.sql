CREATE TABLE IF NOT EXISTS maintainers (
    maintainer_id SERIAL PRIMARY KEY,
    maintainer_name TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    hashed_password TEXT NOT NULL UNIQUE,
    ssh_public_key BYTEA NOT NULL UNIQUE,
    hashed_cookie TEXT UNIQUE,
    expiration_date TIMESTAMPTZ,
    used_space_in_bytes BIGINT NOT NULL,
    storage_limit_in_bytes BIGINT NOT NULL,
    is_admin BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS disabled_maintainers (
    maintainer TEXT PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS apps (
    app_id SERIAL PRIMARY KEY,
    maintainer_id INTEGER,
    app_name TEXT NOT NULL,
    UNIQUE(maintainer_id, app_name),
    FOREIGN KEY (maintainer_id) REFERENCES maintainers(maintainer_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS versions (
    version_id SERIAL PRIMARY KEY,
    version_name TEXT NOT NULL,
    app_id INTEGER NOT NULL,
    creation_timestamp TIMESTAMPTZ NOT NULL,
    data BYTEA NOT NULL,
    signature BYTEA NOT NULL,
    UNIQUE(app_id, version_name),
    FOREIGN KEY (app_id) REFERENCES apps(app_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS configs (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pending_registrations (
    hashed_code TEXT PRIMARY KEY,
    maintainer_name   TEXT NOT NULL,
    email       TEXT NOT NULL,
    hashed_password TEXT NOT NULL,
    ssh_public_key BYTEA NOT NULL,
    expiration_date TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS pending_email_changes (
    hashed_code     TEXT PRIMARY KEY,
    maintainer_id         INTEGER NOT NULL,
    new_email       TEXT NOT NULL,
    expiration_date TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_pending_email_changes_user
      FOREIGN KEY (maintainer_id) REFERENCES maintainers(maintainer_id) ON DELETE CASCADE
);

-- Enable trigram support for fast %...% pattern matching
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Trigram GIN indexes speed up ILIKE/LIKE '%term%' on names
CREATE INDEX IF NOT EXISTS maintainers_maintainer_name_trgm ON maintainers USING gin (maintainer_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS apps_app_name_trgm   ON apps  USING gin (app_name  gin_trgm_ops);

-- Supports JOIN apps a ON a.maintainer_id = u.maintainer_id
CREATE INDEX IF NOT EXISTS apps_maintainer_id_idx ON apps(maintainer_id);

-- Lets Postgres get the newest version per app via index order
CREATE INDEX IF NOT EXISTS versions_app_id_created_desc ON versions (app_id, creation_timestamp DESC);

-- Optional: supports LOWER(col) predicates if used; trigram remains best for %...%
CREATE INDEX IF NOT EXISTS maintainers_maintainer_name_lower_idx ON maintainers (LOWER(maintainer_name));
CREATE INDEX IF NOT EXISTS apps_app_name_lower_idx   ON apps  (LOWER(app_name));
