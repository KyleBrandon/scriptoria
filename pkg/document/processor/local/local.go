package local

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/KyleBrandon/scriptoria/pkg/document"
)

type documentStore interface{}

type LocalDocumentProcessor struct {
	store           documentStore
	ctx             context.Context
	cancelFunc      context.CancelFunc
	wg              *sync.WaitGroup
	destinationPath string
	inputCh         chan *document.DocumentTransform
	outputCh        chan *document.DocumentTransform
}

func New(store documentStore, localStoragePath string) *LocalDocumentProcessor {
	lp := &LocalDocumentProcessor{}
	lp.store = store
	lp.destinationPath = localStoragePath

	return lp
}

func (lp *LocalDocumentProcessor) Initialize(ctx context.Context, inputCh chan *document.DocumentTransform) (chan *document.DocumentTransform, error) {
	slog.Debug(">>LocalDocumentProcessor.Initialize")
	defer slog.Debug("<<LocalDocumentProcessor.Initialize")

	lp.ctx, lp.cancelFunc = context.WithCancel(ctx)
	lp.wg = &sync.WaitGroup{}
	lp.inputCh = inputCh
	lp.outputCh = make(chan *document.DocumentTransform)

	go lp.process()

	return lp.outputCh, nil
}

func (lp *LocalDocumentProcessor) CancelAndWait() {
	lp.cancelFunc()

	lp.wg.Wait()
}

func (lp *LocalDocumentProcessor) process() {
	slog.Debug(">>LocalDocumentProcessor.process")
	defer slog.Debug("<<LocalDocumentProcessor.process")

	for {
		select {
		case <-lp.ctx.Done():
			slog.Debug("LocalDocumentProcessor.process canceled")
			return

		case t := <-lp.inputCh:
			slog.Debug("LocalDocumentProcessor.process received input")
			lp.wg.Add(1)
			go lp.processDocument(t)
		}
	}
}

func (lp *LocalDocumentProcessor) processDocument(t *document.DocumentTransform) {
	slog.Debug(">>LocalDocumentProcessor.processDocument")
	defer slog.Debug("<<LocalDocumentProcessor.processDocument")

	defer lp.wg.Done()

	defer t.Reader.Close()

	output := document.DocumentTransform{}

	// build a local file path
	fullFilePath := filepath.Join(lp.destinationPath, t.Doc.Name)

	// create the local file to save the document to
	file, err := os.Create(fullFilePath)
	if err != nil {
		output.Error = err
		lp.outputCh <- &output
		return
	}

	defer file.Close()

	// copy the document to the local file
	_, err = io.Copy(file, t.Reader)
	if err != nil {
		output.Error = err
		lp.outputCh <- &output
		return
	}

	// open the newly created file for the reader
	file, err = os.Open(fullFilePath)
	if err != nil {
		output.Error = err
		lp.outputCh <- &output
		return
	}

	// output document is largely the same as the input
	output.Doc = &document.Document{
		Name:         t.Doc.Name,
		MimeType:     t.Doc.MimeType,
		CreatedTime:  time.Now(),
		ModifiedTime: time.Now(),
	}

	output.Reader = file

	lp.outputCh <- &output
}
