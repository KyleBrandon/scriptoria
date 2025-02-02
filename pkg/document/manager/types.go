package manager

import (
	"context"
	"sync"

	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
	"github.com/google/uuid"
)

type (
	DocumentManagerStore interface {
		CreateDocument(ctx context.Context, arg database.CreateDocumentParams) (database.Document, error)
		GetDocumentById(ctx context.Context, id uuid.UUID) (database.Document, error)
		FindDocumentBySourceId(ctx context.Context, sourceID string) (database.Document, error)
		UpdateDocumentDestination(ctx context.Context, arg database.UpdateDocumentDestinationParams) (database.Document, error)
		UpdateDocumentProcessed(ctx context.Context, arg database.UpdateDocumentProcessedParams) (database.Document, error)
	}

	DocumentManager struct {
		sync.Mutex

		ctx           context.Context
		cancelFunc    context.CancelFunc
		wg            *sync.WaitGroup
		store         DocumentManagerStore
		sourceType    string
		destType      string
		source        document.DocumentStorage
		destination   document.DocumentStorage
		processors    []document.DocumentProcessor
		postProcessor document.DocumentPostProcessor
		inputCh       chan *document.DocumentTransform
		outputCh      chan *document.DocumentTransform
	}
)
