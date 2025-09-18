-- +goose Up
-- +goose StatementBegin
CREATE TABLE session_summaries (
    id VARCHAR(255) PRIMARY KEY,
    device_id VARCHAR(255) NOT NULL,
    device_name VARCHAR(255),
    platform VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_ip VARCHAR(45),
    user_agent TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE session_summaries;
-- +goose StatementEnd