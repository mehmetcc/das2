-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions
  ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_sessions_last_used_at ON sessions (last_used_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_sessions_last_used_at;
ALTER TABLE sessions
  DROP COLUMN IF EXISTS last_used_at;
-- +goose StatementEnd

