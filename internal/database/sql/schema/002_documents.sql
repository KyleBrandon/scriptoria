-- +goose Up
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,

    source_store TEXT NOT NULL,
    source_id TEXT UNIQUE NOT NULL,
    source_name TEXT NOT NULL,

    destination_store TEXT,
    destination_id TEXT UNIQUE,
    destination_name TEXT,

    transferred_at TIMESTAMP,

    processed_at TIMESTAMP,
    processing_status TEXT
);


-- +goose Down
DROP TABLE documents;
