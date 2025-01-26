package manager

import (
	"context"
	"log/slog"
	"sync"

	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
)

func New(ctx context.Context, queries *database.Queries, source, destination document.DocumentStorage) (*DocumentManager, error) {
	slog.Debug(">>InitializeDocumentManager")
	defer slog.Debug("<<InitializeDocumentManager")

	// create sub-context and cancelFunc
	mgrCtx, mgrCanceFunc := context.WithCancel(ctx)

	// create the DocumentMangaer and initialize it
	var wg sync.WaitGroup
	dm := &DocumentManager{
		ctx:        mgrCtx,
		wg:         &wg,
		cancelFunc: mgrCanceFunc,
		store:      queries,
		documents:  make(chan document.Document, 10),
	}

	// initialize the source storage
	err := source.Initialize(dm.ctx, dm.documents)
	if err != nil {
		slog.Error("Failed to initialize the source storage", "error", err)
		return nil, err
	}

	// initialize the dest storage
	err = destination.Initialize(dm.ctx, dm.documents)
	if err != nil {
		slog.Error("Failed to initialize the destination storage", "error", err)
		return nil, err
	}

	dm.source = source
	dm.destination = destination

	return dm, nil
}

func (dm *DocumentManager) CancelAndWait() {
	// cancel all go routines
	dm.cancelFunc()

	// wait until the document go routines are finished
	dm.wg.Wait()
}

func (dm *DocumentManager) AddDocument(doc document.Document) {
	dm.documents <- doc
}

func (dm *DocumentManager) StartMonitoring() {
	slog.Debug(">>StartMonitoring")
	defer slog.Debug("<<StartMonitoring")

	go dm.documentStorageMonitor()
}

func (dm *DocumentManager) documentStorageMonitor() {
	slog.Info(">>documentStorageMonitor")
	defer slog.Info("<<documentStorageMonitor")

	dm.source.StartWatching()

	for document := range dm.documents {
		dm.wg.Add(1)
		go dm.processDocument(document)
	}
}

func (dm *DocumentManager) processDocument(document document.Document) {
	defer dm.wg.Done()

	// check if we've processed this file before
	dbDoc, err := dm.store.FindDocumentBySourceId(dm.ctx, document.ID)
	if err == nil {
		// assume we've processed this or it's in process
		slog.Warn("Document exists", "id", dbDoc.ID, "sourceID", dbDoc.SourceID, "name", dbDoc.SourceName)
		return
	}

	// TODO: process the document in another go routine
	slog.Info("Processing document", "id", document.ID, "name", document.Name, "createdTime", document.CreatedTime, "modifiedTime", document.ModifiedTime)

	// mark the file as having been processed
	arg := database.CreateDocumentParams{
		SourceStore: dm.sourceType,
		SourceID:    document.ID,
		SourceName:  document.Name,
	}
	_, err = dm.store.CreateDocument(dm.ctx, arg)
	if err != nil {
		slog.Error("Failed to update the document proccessing status", "id", dbDoc.ID, "error", err)
	}
}
