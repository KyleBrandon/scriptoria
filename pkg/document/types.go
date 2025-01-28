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
		GetFileReader(document Document) (io.ReadCloser, error)
		Write(sourceDocument Document, reader io.Reader) (Document, error)
	}

	DocumentContext interface {
		AddDocument(document Document)
	}
)
