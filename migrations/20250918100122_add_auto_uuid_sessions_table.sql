-- +goose Up
-- +goose StatementBegin
ALTER TABLE session_summaries
    ALTER COLUMN id TYPE UUID USING (id::UUID),
    ALTER COLUMN id SET DEFAULT gen_random_uuid();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE session_summaries
    ALTER COLUMN id DROP DEFAULT,
    ALTER COLUMN id TYPE VARCHAR(255) USING (id::VARCHAR);
-- +goose StatementEnd