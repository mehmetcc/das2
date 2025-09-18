-- +goose Up
-- +goose StatementBegin
ALTER TABLE session_summaries
    ADD COLUMN IF NOT EXISTS person_id BIGINT;

-- Drop then add FK (no IF NOT EXISTS; use NOT VALID to avoid full scan now)
ALTER TABLE session_summaries
    DROP CONSTRAINT IF EXISTS fk_session_person;

ALTER TABLE session_summaries
    ADD CONSTRAINT fk_session_person
    FOREIGN KEY (person_id) REFERENCES persons(id) ON DELETE CASCADE
    NOT VALID;

-- Partial index (skips NULLs) for fast user-session lookups
CREATE INDEX IF NOT EXISTS idx_session_summaries_person_id
    ON session_summaries (person_id)
    WHERE person_id IS NOT NULL;

-- (Optional now or later) Validate existing non-null rows without blocking inserts
ALTER TABLE session_summaries
    VALIDATE CONSTRAINT fk_session_person;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE session_summaries
    DROP CONSTRAINT IF EXISTS fk_session_person;

DROP INDEX IF EXISTS idx_session_summaries_person_id;

ALTER TABLE session_summaries
    DROP COLUMN IF EXISTS person_id;
-- +goose StatementEnd
