package manager

import (
	"context"
	"sync"

	"github.com/KyleBrandon/scriptoria/internal/config"
	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
	"github.com/KyleBrandon/scriptoria/pkg/document/processor"
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

		ctx             context.Context
		cancelCauseFunc context.CancelCauseFunc
		wg              *sync.WaitGroup
		config          config.Config
		store           DocumentManagerStore
		srcStorage      document.Storage
		processors      []*processor.ProcessorContext
		inputCh         chan *document.TransformContext
		outputCh        chan *document.TransformContext
	}
)
