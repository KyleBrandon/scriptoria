package document

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"time"
)

type (
	Document struct {
		ID           string
		Name         string
		MimeType     string
		CreatedTime  time.Time
		ModifiedTime time.Time
	}

	DocumentTransform struct {
		Doc    *Document
		Reader io.ReadCloser
		Error  error
	}

	DocumentProcessor interface {
		Initialize(ctx context.Context, inputCh chan *DocumentTransform) (chan *DocumentTransform, error)
	}

	DocumentStorage interface {
		Initialize(ctx context.Context) error
		StartWatching() (chan *Document, error)
		GetDocumentReader(document *Document) (io.ReadCloser, error)
		Write(sourceDocument *Document, reader io.Reader) (*Document, error)
	}
)

func (d *Document) GetDocumentType() string {
	return filepath.Ext(d.Name)
}

func (d *Document) GetDocumentName() string {
	name := strings.TrimSuffix(d.Name, filepath.Ext(d.Name))

	return name
}
