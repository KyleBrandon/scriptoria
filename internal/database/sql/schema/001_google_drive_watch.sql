-- +goose Up
CREATE TABLE google_drive_watch (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    channel_id VARCHAR(500) NOT NULL,
    resource_id VARCHAR(500) NOT NULL,
    expires_at BIGINT NOT NULL,
    webhook_url VARCHAR(1000) NOT NULL
);


-- +goose Down
DROP TABLE google_drive_watch;
