package document

import (
	"context"
	"io"
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
		GetFileReader(sourceFileID string) (io.ReadCloser, error)
		Write(document Document, reader io.Reader) error
	}

	DocumentContext interface {
		AddDocument(document Document)
	}
)
