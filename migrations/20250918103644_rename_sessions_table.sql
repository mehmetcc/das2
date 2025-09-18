-- +goose Up
-- +goose StatementBegin
ALTER TABLE session_summaries RENAME TO sessions;

-- Keep foreign key & indexes consistent
ALTER INDEX IF EXISTS idx_session_summaries_person_id RENAME TO idx_sessions_person_id;
ALTER INDEX IF EXISTS idx_session_summaries_person_id_created_at RENAME TO idx_sessions_person_id_created_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions RENAME TO session_summaries;

ALTER INDEX IF EXISTS idx_sessions_person_id RENAME TO idx_session_summaries_person_id;
ALTER INDEX IF EXISTS idx_sessions_person_id_created_at RENAME TO idx_session_summaries_person_id_created_at;
-- +goose StatementEnd
