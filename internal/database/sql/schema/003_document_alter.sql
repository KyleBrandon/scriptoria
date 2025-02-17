-- +goose Up
ALTER TABLE documents
DROP COLUMN transferred_at,
DROP COLUMN destination_store,
DROP COLUMN destination_id,
DROP COLUMN destination_name;


-- +goose Down
ALTER TABLE plunges 
ADD COLUMN transferred_at TIMESTAMP,
ADD COLUMN destination_store TEXT,
ADD COLUMN destination_id TEXT UNIQUE,
ADD COLUMN destination_name TEXT;
