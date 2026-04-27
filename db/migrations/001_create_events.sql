-- 001_create_events.sql
-- Append-only event store.
-- version is assigned by the application (MAX+1 per stream); the UNIQUE
-- constraint on (stream_id, version) prevents concurrent duplicates.

CREATE TABLE IF NOT EXISTS events (
    id          UUID        PRIMARY KEY,
    tenant_id   TEXT        NOT NULL DEFAULT 'default',
    stream_id   TEXT        NOT NULL,
    type        TEXT        NOT NULL,
    source      TEXT        NOT NULL,
    version     BIGINT      NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    payload     JSONB       NOT NULL,
    metadata    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (stream_id, version)
);

CREATE INDEX IF NOT EXISTS idx_events_stream_id   ON events (stream_id);
CREATE INDEX IF NOT EXISTS idx_events_type        ON events (type);
CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events (occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_tenant_id   ON events (tenant_id);
