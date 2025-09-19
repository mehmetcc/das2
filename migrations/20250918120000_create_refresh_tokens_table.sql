-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id    BIGINT NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    session_id   UUID   NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    token_hash   TEXT   NOT NULL,
    issued_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL,
    rotated_at   TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    replaced_by  UUID NULL REFERENCES refresh_tokens(id),
    user_agent   TEXT,
    ip           VARCHAR(64),
    device_id    VARCHAR(128)
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_session_active
  ON refresh_tokens (session_id)
  WHERE revoked_at IS NULL AND rotated_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_person_active
  ON refresh_tokens (person_id, expires_at)
  WHERE revoked_at IS NULL AND rotated_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at
  ON refresh_tokens (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS refresh_tokens;
-- +goose StatementEnd

