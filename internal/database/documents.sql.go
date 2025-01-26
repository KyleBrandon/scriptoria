// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: documents.sql

package database

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const createDocument = `-- name: CreateDocument :one
INSERT INTO documents (
    source_store, source_id, source_name
) VALUES ( $1, $2, $3)
RETURNING id, created_at, updated_at, source_store, source_id, source_name, destination_store, destination_id, destination_name, transferred_at, processed_at, processing_status
`

type CreateDocumentParams struct {
	SourceStore string
	SourceID    string
	SourceName  string
}

func (q *Queries) CreateDocument(ctx context.Context, arg CreateDocumentParams) (Document, error) {
	row := q.db.QueryRowContext(ctx, createDocument, arg.SourceStore, arg.SourceID, arg.SourceName)
	var i Document
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.SourceStore,
		&i.SourceID,
		&i.SourceName,
		&i.DestinationStore,
		&i.DestinationID,
		&i.DestinationName,
		&i.TransferredAt,
		&i.ProcessedAt,
		&i.ProcessingStatus,
	)
	return i, err
}

const getDocumentById = `-- name: GetDocumentById :one
SELECT id, created_at, updated_at, source_store, source_id, source_name, destination_store, destination_id, destination_name, transferred_at, processed_at, processing_status FROM documents
WHERE id = $1
`

func (q *Queries) GetDocumentById(ctx context.Context, id uuid.UUID) (Document, error) {
	row := q.db.QueryRowContext(ctx, getDocumentById, id)
	var i Document
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.SourceStore,
		&i.SourceID,
		&i.SourceName,
		&i.DestinationStore,
		&i.DestinationID,
		&i.DestinationName,
		&i.TransferredAt,
		&i.ProcessedAt,
		&i.ProcessingStatus,
	)
	return i, err
}

const updateDocumentDestination = `-- name: UpdateDocumentDestination :one
UPDATE documents
SET destination_store = $2, destination_id = $3, destination_name = $4, transferred_at = $5
WHERE id = $1
RETURNING id, created_at, updated_at, source_store, source_id, source_name, destination_store, destination_id, destination_name, transferred_at, processed_at, processing_status
`

type UpdateDocumentDestinationParams struct {
	ID               uuid.UUID
	DestinationStore sql.NullString
	DestinationID    sql.NullString
	DestinationName  sql.NullString
	TransferredAt    sql.NullTime
}

func (q *Queries) UpdateDocumentDestination(ctx context.Context, arg UpdateDocumentDestinationParams) (Document, error) {
	row := q.db.QueryRowContext(ctx, updateDocumentDestination,
		arg.ID,
		arg.DestinationStore,
		arg.DestinationID,
		arg.DestinationName,
		arg.TransferredAt,
	)
	var i Document
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.SourceStore,
		&i.SourceID,
		&i.SourceName,
		&i.DestinationStore,
		&i.DestinationID,
		&i.DestinationName,
		&i.TransferredAt,
		&i.ProcessedAt,
		&i.ProcessingStatus,
	)
	return i, err
}

const updateDocumentProcessed = `-- name: UpdateDocumentProcessed :one
UPDATE documents
SET processed_at = $2, processing_status = $3
WHERE id = $1
RETURNING id, created_at, updated_at, source_store, source_id, source_name, destination_store, destination_id, destination_name, transferred_at, processed_at, processing_status
`

type UpdateDocumentProcessedParams struct {
	ID               uuid.UUID
	ProcessedAt      sql.NullTime
	ProcessingStatus sql.NullString
}

func (q *Queries) UpdateDocumentProcessed(ctx context.Context, arg UpdateDocumentProcessedParams) (Document, error) {
	row := q.db.QueryRowContext(ctx, updateDocumentProcessed, arg.ID, arg.ProcessedAt, arg.ProcessingStatus)
	var i Document
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.SourceStore,
		&i.SourceID,
		&i.SourceName,
		&i.DestinationStore,
		&i.DestinationID,
		&i.DestinationName,
		&i.TransferredAt,
		&i.ProcessedAt,
		&i.ProcessingStatus,
	)
	return i, err
}
