package document

import (
	"context"
	"time"
)

type (
	Document struct {
		ID           string
		Name         string
		CreatedTime  time.Time
		ModifiedTime time.Time
	}

	DocumentStorage interface {
		Initialize(ctx context.Context, documents chan Document) error
		StartWatching() error
	}

	DocumentContext interface {
		AddDocument(document Document)
	}
)
