-- +goose Up
ALTER TABLE checks
    ADD COLUMN schedule_mode TEXT NOT NULL DEFAULT 'manual'
        CHECK (schedule_mode IN ('manual', 'auto')),
    ADD COLUMN interval_ewma_seconds DOUBLE PRECISION,
    ADD COLUMN interval_jitter_ewma_seconds DOUBLE PRECISION,
    ADD COLUMN interval_sample_count INT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE checks
    DROP COLUMN IF EXISTS interval_sample_count,
    DROP COLUMN IF EXISTS interval_jitter_ewma_seconds,
    DROP COLUMN IF EXISTS interval_ewma_seconds,
    DROP COLUMN IF EXISTS schedule_mode;
