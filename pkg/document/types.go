package document

import (
	"context"
	"io"
	"time"

	"github.com/KyleBrandon/scriptoria/internal/config"
	"github.com/google/uuid"
)

type (
	// Document represents a document that is passed through the system for transformation.
	Document struct {
		StorageDocumentID string    // ID of the document in the system it came from.  Can be empty for state transitions.
		StorageFolderID   string    // ID of the folder that the document is stored in
		Name              string    // Name of the current document representation
		CreatedTime       time.Time // Time the document was created
		ModifiedTime      time.Time // Time  the document was last modified
	}

	// TransformContext represents a state of a document at a given time for it to be transformed.
	//  Ex:
	//      Input PDF
	//      Output Markdown
	TransformContext struct {
		DocumentID     uuid.UUID     // Database document ID
		SourceDocument *Document     // Source document
		Reader         io.ReadCloser // Reader for the current Document representation.
	}

	// Storage represents where a Document will be read from and to.
	Storage interface {
		// Initlaize the DocumentStorage
		Initialize(ctx context.Context, bundles []config.StorageBundle) error

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
