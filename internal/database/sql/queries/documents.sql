-- name: CreateDocument :one
INSERT INTO documents (
    source_store, source_id, source_name
) VALUES ( $1, $2, $3)
RETURNING *;

-- name: UpdateDocumentDestination :one
UPDATE documents
SET destination_store = $2, 
    destination_id = $3, 
    destination_name = $4, 
    transferred_at = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;


-- name: UpdateDocumentProcessed :one
UPDATE documents
SET processed_at = $2, 
processing_status = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetDocumentById :one
SELECT * FROM documents
WHERE id = $1;

-- name: FindDocumentBySourceId :one
SELECT * FROM documents
WHERE source_id = $1
;

