-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expected_interval_seconds INT NOT NULL CHECK (expected_interval_seconds > 0),
    grace_seconds INT NOT NULL DEFAULT 180 CHECK (grace_seconds >= 0),
    status TEXT NOT NULL CHECK (status IN ('up', 'late', 'down', 'paused')) DEFAULT 'up',
    last_ping_at TIMESTAMPTZ,
    next_due_at TIMESTAMPTZ,
    last_duration_ms INT,
    baseline_duration_ms DOUBLE PRECISION,
    sample_count_duration INT NOT NULL DEFAULT 0,
    last_output_size INT,
    baseline_output_size DOUBLE PRECISION,
    sample_count_output INT NOT NULL DEFAULT 0,
    last_success BOOLEAN,
    last_alerted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX checks_user_name_unique_idx ON checks(user_id, lower(name)) WHERE deleted_at IS NULL;
CREATE INDEX checks_user_status_idx ON checks(user_id, status);
CREATE INDEX checks_next_due_idx ON checks(next_due_at) WHERE deleted_at IS NULL;

CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    check_id UUID NOT NULL REFERENCES checks(id),
    type TEXT NOT NULL CHECK (type IN ('ping', 'start', 'fail', 'warning')) DEFAULT 'ping',
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    duration_ms INT,
    output_size INT,
    success BOOLEAN,
    meta_json JSONB
);

CREATE INDEX events_check_received_idx ON events(check_id, received_at DESC);

CREATE TABLE incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    check_id UUID NOT NULL REFERENCES checks(id),
    status TEXT NOT NULL CHECK (status IN ('open', 'resolved')),
    opened_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    open_event_id BIGINT REFERENCES events(id),
    resolve_event_id BIGINT REFERENCES events(id)
);

CREATE UNIQUE INDEX incidents_one_open_per_check_idx ON incidents(check_id) WHERE status = 'open';
CREATE INDEX incidents_check_status_idx ON incidents(check_id, status);

CREATE TABLE alert_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    type TEXT NOT NULL CHECK (type IN ('email', 'webhook')),
    target TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE alert_deliveries (
    id BIGSERIAL PRIMARY KEY,
    incident_id UUID REFERENCES incidents(id),
    check_id UUID NOT NULL REFERENCES checks(id),
    channel_id UUID NOT NULL REFERENCES alert_channels(id),
    kind TEXT NOT NULL CHECK (kind IN ('down', 'repeat', 'recovered', 'warning')),
    status TEXT NOT NULL CHECK (status IN ('queued', 'sending', 'sent', 'failed', 'dead_letter')),
    priority INT NOT NULL DEFAULT 0,
    attempt_count INT NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    error_text TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX alert_deliveries_claim_idx ON alert_deliveries(status, next_attempt_at, id);

CREATE TABLE ping_idempotency (
    check_id UUID NOT NULL REFERENCES checks(id),
    idempotency_key TEXT NOT NULL,
    response_status TEXT NOT NULL,
    response_next_due_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (check_id, idempotency_key)
);

CREATE INDEX ping_idempotency_expires_idx ON ping_idempotency(expires_at);

-- +goose Down
DROP TABLE IF EXISTS ping_idempotency;
DROP TABLE IF EXISTS alert_deliveries;
DROP TABLE IF EXISTS alert_channels;
DROP TABLE IF EXISTS incidents;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS checks;
DROP TABLE IF EXISTS users;
