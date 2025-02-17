package processor

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/KyleBrandon/scriptoria/internal/config"
	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
)

type ProcessorConfig struct {
	Ctx               context.Context
	CancelCauseFunc   context.CancelCauseFunc
	Store             ProcessorStore
	TempStorageFolder string
	AttachmentsFolder string
	NotesFolder       string
	Bundles           []config.StorageBundle
}

// Processor is an interface to define the processing of a document.  Implementations
//
//	will create the Intialize method which defines the input/output channels for documents to
//	enter and leave the processor.
type Processor interface {
	// Initialize the processor
	Initialize(tempStoragePath string, bundles []config.StorageBundle) error

	// Process the document passed in the reader and return another reader with the new transformed document.
	Process(document *document.Document, reader io.ReadCloser) (io.ReadCloser, error)

	// Name of the Processor
	GetName() string
}

type ProcessorContext struct {
	ctx             context.Context
	cancelCauseFunc context.CancelCauseFunc
	store           ProcessorStore

	tempStoragePath string
	bundles         []config.StorageBundle

	wg        *sync.WaitGroup
	processor Processor
	inputCh   chan *document.TransformContext
	outputCh  chan *document.TransformContext
}

type ProcessorStore interface {
	UpdateDocumentProcessed(ctx context.Context, arg database.UpdateDocumentProcessedParams) (database.Document, error)
}

func New(cfg ProcessorConfig, processor Processor) *ProcessorContext {
	slog.Debug(">>ProcessorContext.New")
	defer slog.Debug("<<ProcessorContext.New")
	pc := &ProcessorContext{
		ctx:             cfg.Ctx,
		cancelCauseFunc: cfg.CancelCauseFunc,
		store:           cfg.Store,
		tempStoragePath: cfg.TempStorageFolder,
		bundles:         cfg.Bundles,
		processor:       processor,
		wg:              &sync.WaitGroup{},
		outputCh:        make(chan *document.TransformContext),
	}

	return pc
}

func (pc *ProcessorContext) Initialize(inputCh chan *document.TransformContext) (chan *document.TransformContext, error) {
	slog.Debug(">>ProcessorContext.Initialize")
	defer slog.Debug("<<ProcessorContext.Initialize")

	err := pc.processor.Initialize(pc.tempStoragePath, pc.bundles)
	if err != nil {
		return nil, err
	}

	pc.wg.Add(1)
	pc.inputCh = inputCh
	go pc.process()

	return pc.outputCh, nil
}

func (pc *ProcessorContext) CancelAndWait() {
	pc.cancelCauseFunc(nil)
	pc.wg.Wait()
}

func (pc *ProcessorContext) process() {
	slog.Debug(">>ProcessorContext.process")
	defer slog.Debug("<<ProcessorContext.process")

	defer pc.wg.Done()

	for {
		select {
		case <-pc.ctx.Done():
			slog.Debug("ProcessorContext.process canceled")
			return

		case t := <-pc.inputCh:
			pc.wg.Add(1)
			go pc.processWrapper(t)
		}
	}
}

func (pc *ProcessorContext) processWrapper(t *document.TransformContext) {
	defer pc.wg.Done()
	defer t.Reader.Close()

	pc.updateDocumentProcessingStatus(t, "start processing")

	reader, err := pc.processor.Process(t.SourceDocument, t.Reader)
	if err != nil {
		pc.cancelCauseFunc(err)
		pc.updateDocumentProcessingStatus(t, err.Error())
		return
	}

	pc.updateDocumentProcessingStatus(t, "finished processing")

	// continue to the next processor
	t.Reader = reader
	pc.outputCh <- t
}

func (pc *ProcessorContext) updateDocumentProcessingStatus(tc *document.TransformContext, message string) {
	processorName := pc.processor.GetName()
	statusMessage := fmt.Sprintf("%s: %s", processorName, message)

	args := database.UpdateDocumentProcessedParams{
		ID:               tc.DocumentID,
		ProcessedAt:      sql.NullTime{Time: time.Now().UTC(), Valid: true},
		ProcessingStatus: sql.NullString{String: statusMessage, Valid: true},
	}

	_, err := pc.store.UpdateDocumentProcessed(pc.ctx, args)
	if err != nil {
		slog.Error("Failed to update the document status in the database", "error", err)
	}
}

func CopyFileFromReader(fullFilePath string, reader io.ReadCloser) error {
	// create the local file to save the document to
	file, err := os.Create(fullFilePath)
	if err != nil {
		return err
	}

	defer file.Close()

	// copy the document to the local file
	_, err = io.Copy(file, reader)
	if err != nil {
		return err
	}

	return nil
}

// CopyFile copies a file from src to dst
func copyFile(src, dst string) error {
	// Open the source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create the destination file
	destinationFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destinationFile.Close()

	// Copy the content from source to destination
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Ensure the destination file gets flushed and closed properly
	err = destinationFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}
