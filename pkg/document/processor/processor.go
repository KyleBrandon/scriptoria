package processor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/KyleBrandon/scriptoria/pkg/document"
)

type ProcessorConfig struct {
	Ctx               context.Context
	CancelCauseFunc   context.CancelCauseFunc
	Store             ProcessorStore
	TempStorageFolder string
	AttachmentsFolder string
	NotesFolder       string
}

// Processor is an interface to define the processing of a document.  Implementations
//
//	will create the Intialize method which defines the input/output channels for documents to
//	enter and leave the processor.
type Processor interface {
	// Initialize the processor
	Initialize(tempStoragePath string) error

	// Process the document passed in the reader and return another reader with the new transformed document.
	Process(sourceName string, reader io.ReadCloser) (io.ReadCloser, error)
}

type ProcessorContext struct {
	ctx             context.Context
	cancelCauseFunc context.CancelCauseFunc
	store           ProcessorStore

	tempStoragePath string
	attachmentsPath string
	notesPath       string

	wg        *sync.WaitGroup
	processor Processor
	inputCh   chan *document.TransformContext
	outputCh  chan *document.TransformContext
}

type ProcessorStore interface {
	//
}

func New(config ProcessorConfig, processor Processor) *ProcessorContext {
	slog.Debug(">>ProcessorContext.New")
	defer slog.Debug("<<ProcessorContext.New")
	pc := &ProcessorContext{
		ctx:             config.Ctx,
		cancelCauseFunc: config.CancelCauseFunc,
		store:           config.Store,
		tempStoragePath: config.TempStorageFolder,
		attachmentsPath: config.AttachmentsFolder,
		notesPath:       config.NotesFolder,
		processor:       processor,
		wg:              &sync.WaitGroup{},
		outputCh:        make(chan *document.TransformContext),
	}

	return pc
}

func (pc *ProcessorContext) Initialize(inputCh chan *document.TransformContext) (chan *document.TransformContext, error) {
	slog.Debug(">>ProcessorContext.Initialize")
	defer slog.Debug("<<ProcessorContext.Initialize")

	err := pc.processor.Initialize(pc.tempStoragePath)
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

	reader, err := pc.processor.Process(t.SourceName, t.Reader)
	if err != nil {
		pc.cancelCauseFunc(err)
		return
	}

	// continue to the next processor
	t.Reader = reader
	pc.outputCh <- t
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
