CREATE TABLE IF NOT EXISTS buckets (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    display_name TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_prefix TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    scope TEXT NOT NULL DEFAULT 'read_write',
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS objects (
    id BIGSERIAL PRIMARY KEY,
    bucket TEXT NOT NULL,
    object_key TEXT NOT NULL,
    file_path TEXT NOT NULL,
    size BIGINT NOT NULL,
    content_type TEXT,
    checksum TEXT,
    version_id TEXT NOT NULL,
    is_latest BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS replicas (
    id BIGSERIAL PRIMARY KEY,
    object_id BIGINT NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    node_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS replication_jobs (
    id BIGSERIAL PRIMARY KEY,
    object_id BIGINT NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    bucket TEXT NOT NULL,
    object_key TEXT NOT NULL,
    version_id TEXT NOT NULL,
    source_file_path TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    next_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Clean up duplicate latest flags before enforcing the partial unique index.
WITH ranked_latest AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY bucket, object_key
               ORDER BY created_at DESC, id DESC
           ) AS rn
    FROM objects
    WHERE is_latest = TRUE
)
UPDATE objects
SET is_latest = FALSE
WHERE id IN (
    SELECT id FROM ranked_latest WHERE rn > 1
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_objects_latest_per_key
    ON objects(bucket, object_key)
    WHERE is_latest = TRUE;

CREATE UNIQUE INDEX IF NOT EXISTS idx_objects_version_unique
    ON objects(bucket, object_key, version_id);

CREATE INDEX IF NOT EXISTS idx_objects_lookup_latest
    ON objects(bucket, object_key, is_latest);

CREATE UNIQUE INDEX IF NOT EXISTS idx_replicas_object_node_unique
    ON replicas(object_id, node_name);

CREATE INDEX IF NOT EXISTS idx_replication_jobs_fetch
    ON replication_jobs(status, next_run_at);
