-- +goose Up
-- +goose StatementBegin
-- keep the default for created_at
ALTER TABLE session_summaries
    ALTER COLUMN created_at SET DEFAULT now();

-- composite index (fine even if person_id has NULLs)
CREATE INDEX IF NOT EXISTS idx_session_summaries_person_id_created_at
    ON session_summaries (person_id, created_at DESC);

-- IMPORTANT: do NOT set NOT NULL here (old rows exist)
-- Instead, enforce non-null for NEW/UPDATED rows only:
ALTER TABLE session_summaries
    DROP CONSTRAINT IF EXISTS chk_session_person_id_not_null;

ALTER TABLE session_summaries
    ADD CONSTRAINT chk_session_person_id_not_null
    CHECK (person_id IS NOT NULL) NOT VALID;  -- not checked against existing rows
-- Do NOT VALIDATE yet; keep it unvalidated until you clean old rows.
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_session_summaries_person_id_created_at;

ALTER TABLE session_summaries
    DROP CONSTRAINT IF EXISTS chk_session_person_id_not_null;

ALTER TABLE session_summaries
    ALTER COLUMN created_at DROP DEFAULT;
-- +goose StatementEnd
