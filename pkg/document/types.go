package document

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"time"
)

type (
	// Document represents a document that is passed through the system for transformation.
	Document struct {
		ID           string    // ID of the document in the system it came from.  Can be empty for state transitions.
		Name         string    // Name of the current document representation
		MimeType     string    // Mime type of the document
		CreatedTime  time.Time // Time the document was created
		ModifiedTime time.Time // Time  the document was last modified
	}

	// TransformContext represents a state of a document at a given time for it to be transformed.
	//  Ex:
	//      Input PDF
	//      Output Markdown
	TransformContext struct {
		SourceName string
		Reader     io.ReadCloser // Reader for the current Document representation.
	}

	// Storage represents where a Document will be read from and to.
	Storage interface {
		// Initlaize the DocumentStorage
		Initialize(ctx context.Context) error

		// CancelAndWait for the storage to finish any work
		CancelAndWait()

		// StartWatching for documents that need to be processed.
		StartWatching() (chan *Document, error)

		// Given a document, create a reader for its contents.
		GetReader(document *Document) (io.ReadCloser, error)

		// Write a document to the DocumentStorage.
		Write(sourceDocument *Document, reader io.ReadCloser) (*Document, error)

		// Archive the document.  This is called after the document is successfully processed to ensure we don't process it again.
		Archive(sourceDocument *Document) error
	}
)

// GetDocumentType will return the file extension of the document.
func (d *Document) GetDocumentType() string {
	return filepath.Ext(d.Name)
}

// Get DocumentName will return just the name (no extension) of the document.
func (d *Document) GetDocumentName() string {
	name := strings.TrimSuffix(d.Name, filepath.Ext(d.Name))

	return name
}
