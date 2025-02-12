package manager

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/KyleBrandon/scriptoria/internal/config"
	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
	"github.com/KyleBrandon/scriptoria/pkg/document/processor"
	"github.com/KyleBrandon/scriptoria/pkg/document/processor/chatgpt"
	"github.com/KyleBrandon/scriptoria/pkg/document/processor/mathpix"
	"github.com/KyleBrandon/scriptoria/pkg/document/processor/obsidian"
	"github.com/KyleBrandon/scriptoria/pkg/document/storage"
)

func New(ctx context.Context, queries *database.Queries, config config.Config, mux *http.ServeMux) (*DocumentManager, error) {
	slog.Debug(">>DocumentManager.New")
	defer slog.Debug("<<DocumentManager.New")

	// create sub-context and cancelCauseFunc
	mgrCtx, cancelCauseFunc := context.WithCancelCause(ctx)

	// create the DocumentMangaer and initialize it
	var wg sync.WaitGroup
	dm := &DocumentManager{
		ctx:             mgrCtx,
		wg:              &wg,
		cancelCauseFunc: cancelCauseFunc,
		store:           queries,
		config:          config,
	}

	// initialize the storage reader
	err := dm.initializeStorage(queries, mux)
	if err != nil {
		return nil, err
	}

	// initialize the processors
	err = dm.initializeProcessors(queries, config)
	if err != nil {
		return nil, err
	}

	return dm, nil
}

func (dm *DocumentManager) initializeStorage(queries *database.Queries, mux *http.ServeMux) error {
	slog.Debug(">>DocumentManager.initializeStorage")
	defer slog.Debug("<<DocumentManager.initializeStorage")

	storage, err := storage.BuildDocumentStorage(dm.config.SourceStore, queries, mux)
	if err != nil {
		slog.Error("Failed to initialize the source storage", "error", err)
		return err
	}

	// initialize the source storage
	err = storage.Initialize(dm.ctx, dm.config.Bundles)
	if err != nil {
		slog.Error("Failed to initialize the source storage", "error", err)
		return err
	}

	dm.srcStorage = storage

	return nil
}

func (dm *DocumentManager) initializeProcessors(queries *database.Queries, config config.Config) error {
	slog.Debug(">>DocumentManager.initializeProcessors")
	defer slog.Debug("<<DocumentManager.initializeProcessors")

	cfg := processor.ProcessorConfig{
		Ctx:               dm.ctx,
		CancelCauseFunc:   dm.cancelCauseFunc,
		Store:             queries,
		TempStorageFolder: config.TempStorageFolder,
		Bundles:           config.Bundles,
	}

	dm.processors = make([]*processor.ProcessorContext, 0)
	dm.processors = append(dm.processors, processor.New(cfg, processor.NewTempStorageProcessor()))
	dm.processors = append(dm.processors, processor.New(cfg, mathpix.NewMathpixProcessor()))
	dm.processors = append(dm.processors, processor.New(cfg, chatgpt.NewChatGPTProcessor()))
	dm.processors = append(dm.processors, processor.New(cfg, obsidian.NewObsidianProcessor()))
	dm.processors = append(dm.processors, processor.New(cfg, processor.NewBundleProcessor()))

	dm.inputCh = make(chan *document.TransformContext)
	inputCh := dm.inputCh

	// loop through the processors and initialize them by chaining their channels
	for _, p := range dm.processors {
		outputCh, err := p.Initialize(inputCh)
		if err != nil {
			slog.Error("Failed to initialize the processors", "error", err)
			return err
		}

		inputCh = outputCh
	}

	// output processor channel is the last input
	dm.outputCh = inputCh

	return nil
}

func (dm *DocumentManager) CancelAndWait() {
	// cancel all go routines
	dm.cancelCauseFunc(nil)

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

	dm.wg.Add(1)
	go dm.documentStorageMonitor()
}

func (dm *DocumentManager) documentStorageMonitor() {
	slog.Debug(">>documentStorageMonitor")
	defer slog.Debug("<<documentStorageMonitor")

	defer dm.wg.Done()

	// start watching the source for new files
	docCh, err := dm.srcStorage.StartWatching()
	if err != nil {
		slog.Error("Failed to start watching on the source channel", "source", dm.config.SourceStore, "error", err)
		return
	}

	// process each file in a go routine as they come in

	for {
		select {
		case <-dm.ctx.Done():
			slog.Debug("DocumentManager.documentStorageMonitor canceled")
			return

		case srcDoc := <-docCh:
			dm.wg.Add(1)
			go dm.processDocument(srcDoc, dm.srcStorage)
		}
	}
}

func (dm *DocumentManager) processDocument(srcDoc *document.Document, srcStorage document.Storage) {
	slog.Debug(">>DocumentManger.processDocument")
	defer slog.Debug("<<DocumentManger.processDocument")

	defer dm.wg.Done()

	// check if we've processed this document and create it's state in the database
	err := dm.createNewDocument(srcDoc)
	if err != nil {
		return
	}

	// get the io.Reader for the document from the source storae
	inputReader, err := srcStorage.GetReader(srcDoc)
	if err != nil {
		slog.Error("Failed to get the document reader", "error", err)
		return
	}

	// Send the document transform context to the input channel (first processor)
	dm.inputCh <- &document.TransformContext{
		Doc:    srcDoc,
		Reader: inputReader,
	}

	// wait on output channel
	t := <-dm.outputCh

	// if we have a final reader make sure it's closed
	if t.Reader != nil {
		t.Reader.Close()
	}

	// archive the file now that we're done processing it
	srcStorage.Archive(srcDoc)

	slog.Info("Finished processing the document", "sourceName", t.Doc.Name)
}

func (dm *DocumentManager) createNewDocument(srcDoc *document.Document) error {
	slog.Debug(">>DocumentManager.createNewDocument")
	defer slog.Debug("<<DocumentManager.createNewDocument")

	// check if we've processed this file before
	dbDoc, err := dm.store.FindDocumentBySourceId(dm.ctx, srcDoc.ID)
	if err == nil {
		// assume we've processed this or it's in process
		slog.Warn("Document exists", "id", dbDoc.ID, "sourceID", dbDoc.SourceID, "name", dbDoc.SourceName)
		return err
	}

	// mark the file as having been processed
	arg := database.CreateDocumentParams{
		SourceStore: dm.config.SourceStore,
		SourceID:    srcDoc.ID,
		SourceName:  srcDoc.Name,
	}
	_, err = dm.store.CreateDocument(dm.ctx, arg)
	if err != nil {
		slog.Error("Failed to update the document proccessing status", "id", dbDoc.ID, "error", err)
		return err
	}

	return nil
}
