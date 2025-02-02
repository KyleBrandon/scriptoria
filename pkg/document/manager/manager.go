package manager

import (
	"context"
	"log/slog"
	"sync"

	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
)

func New(ctx context.Context, queries *database.Queries, source document.DocumentStorage, processors []document.DocumentProcessor, postProcessor document.DocumentPostProcessor) (*DocumentManager, error) {
	slog.Debug(">>InitializeDocumentManager")
	defer slog.Debug("<<InitializeDocumentManager")

	// create sub-context and cancelFunc
	mgrCtx, mgrCanceFunc := context.WithCancel(ctx)

	// create the DocumentMangaer and initialize it
	var wg sync.WaitGroup
	dm := &DocumentManager{
		ctx:           mgrCtx,
		wg:            &wg,
		cancelFunc:    mgrCanceFunc,
		store:         queries,
		source:        source,
		processors:    processors,
		postProcessor: postProcessor,
	}

	// initialize the source storage
	err := dm.source.Initialize(dm.ctx)
	if err != nil {
		slog.Error("Failed to initialize the source storage", "error", err)
		return nil, err
	}

	dm.inputCh = make(chan *document.DocumentTransform)
	inputCh := dm.inputCh

	// loop through the processors and initialize them by chaining their channels
	for _, p := range dm.processors {
		outputCh, err := p.Initialize(dm.ctx, inputCh)
		if err != nil {
			slog.Error("Failed to initialize the document processor", "error", err)
			return nil, err
		}

		inputCh = outputCh
	}

	// output processor channel is the last input
	dm.outputCh = inputCh

	err = dm.postProcessor.Initialize(dm.ctx)
	if err != nil {
		slog.Error("Failed to initialize the post processor")
	}

	return dm, nil
}

func (dm *DocumentManager) CancelAndWait() {
	// cancel all go routines
	dm.cancelFunc()

	// cancel all the processors
	for _, p := range dm.processors {
		p.CancelAndWait()
	}

	// wait until the document go routines are finished
	dm.wg.Wait()
}

func (dm *DocumentManager) StartMonitoring() {
	slog.Debug(">>StartMonitoring")
	defer slog.Debug("<<StartMonitoring")

	go dm.documentStorageMonitor()
}

func (dm *DocumentManager) documentStorageMonitor() {
	slog.Debug(">>documentStorageMonitor")
	defer slog.Debug("<<documentStorageMonitor")

	// start watching the source for new files
	docCh, err := dm.source.StartWatching()
	if err != nil {
		slog.Error("Failed to start watching on the source channel", "source", dm.sourceType, "error", err)
		return
	}

	// process each file in a go routine as they come in
	for srcDoc := range docCh {
		dm.wg.Add(1)
		go dm.processDocument(srcDoc, dm.source)
	}
}

func (dm *DocumentManager) processDocument(srcDoc *document.Document, srcStorage document.DocumentStorage) {
	defer dm.wg.Done()

	// check if we've processed this file before
	dbDoc, err := dm.store.FindDocumentBySourceId(dm.ctx, srcDoc.ID)
	if err == nil {
		// assume we've processed this or it's in process
		slog.Warn("Document exists", "id", dbDoc.ID, "sourceID", dbDoc.SourceID, "name", dbDoc.SourceName)
		return
	}

	// mark the file as having been processed
	arg := database.CreateDocumentParams{
		SourceStore: dm.sourceType,
		SourceID:    srcDoc.ID,
		SourceName:  srcDoc.Name,
	}
	_, err = dm.store.CreateDocument(dm.ctx, arg)
	if err != nil {
		slog.Error("Failed to update the document proccessing status", "id", dbDoc.ID, "error", err)
		return
	}

	reader, err := srcStorage.GetDocumentReader(srcDoc)
	if err != nil {
		slog.Error("Failed to get the document reader", "error", err)
		return
	}
	defer reader.Close()

	dm.inputCh <- &document.DocumentTransform{
		Doc:    srcDoc,
		Reader: reader,
	}

	// wait on output
	outputTransform := <-dm.outputCh
	if outputTransform.Error != nil {
		slog.Error("Document processing failed", "error", err)
		return
	}

	err = dm.postProcessor.Process(srcDoc, outputTransform)
	if err != nil {
		return
	}
}
