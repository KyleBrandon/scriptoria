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

	// DocumentTransform represents a state of a document at a given time for it to be transformed.
	//  Ex:
	//      Input PDF
	//      Output Markdown
	DocumentTransform struct {
		Doc    *Document     // Represents the current Document state for the transform.
		Reader io.ReadCloser // Reader for the current Document representation.
		Error  error         // Error of the transformation output.
	}

	// DocumentProcessor is an interface to define the processing of a document.  Implementations
	//  will create the Intialize method which defines the input/output channels for documents to
	//  enter and leave the processor.
	DocumentProcessor interface {
		// Initialize the document processor receiving a context and th input channel that documents
		//  will arrive on.
		//  Returns the output channel that transformations will be sent to.
		Initialize(ctx context.Context, inputCh chan *DocumentTransform) (chan *DocumentTransform, error)

		// Cancel any current processing and wait for it to finish.
		CancelAndWait()
	}

	// DocumentPostProcessor allows for a final step to be executed that will receive the original
	//  input Document e.g. PDF and the final output transformation.
	// This can be used to perform any final processing on the original and final documents.
	DocumentPostProcessor interface {
		// Initialize the DocumentPostProcessor
		Initialize(ctx context.Context) error
		// Process will peroform any needed processing on the source and output.
		Process(srcDoc *Document, outputTransform *DocumentTransform) error
	}

	// DocumentStorage represents where a Document will be read from and to.
	DocumentStorage interface {
		// Initlaize the DocumentStorage
		Initialize(ctx context.Context) error

		// StartWatching for documents that need to be processed.
		StartWatching() (chan *Document, error)

		// Given a document, create a reader for its contents.
		GetDocumentReader(document *Document) (io.ReadCloser, error)

		// Write a document to the DocumentStorage.
		Write(sourceDocument *Document, reader io.ReadCloser) (*Document, error)
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
