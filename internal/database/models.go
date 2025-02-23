// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package database

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Document struct {
	ID               uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
	SourceStore      string
	SourceID         string
	SourceName       string
	ProcessedAt      sql.NullTime
	ProcessingStatus sql.NullString
}

type GoogleDriveWatch struct {
	ID         uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ChannelID  string
	ResourceID string
	ExpiresAt  int64
	WebhookUrl string
}
