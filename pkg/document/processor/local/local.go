package local

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/KyleBrandon/scriptoria/pkg/document"
)

type documentStore interface{}

type LocalDocumentProcessor struct {
	store         documentStore
	ctx           context.Context
	localFilePath string
	fullFilePath  string
	inputCh       chan *document.DocumentTransform
	outputCh      chan *document.DocumentTransform
}

func New(store documentStore) *LocalDocumentProcessor {
	lp := &LocalDocumentProcessor{}

	return lp
}

func (lp *LocalDocumentProcessor) Initialize(ctx context.Context, inputCh chan *document.DocumentTransform) (chan *document.DocumentTransform, error) {
	slog.Debug(">>LocalDocumentProcessor.Initialize")
	defer slog.Debug("<<LocalDocumentProcessor.Initialize")

	lp.ctx = ctx
	lp.inputCh = inputCh
	lp.outputCh = make(chan *document.DocumentTransform)

	err := lp.readConfigurationSettings()
	if err != nil {
		return nil, err
	}

	go lp.process()

	return lp.outputCh, nil
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
			go lp.processDocument(t)
		}
	}
}

func (lp *LocalDocumentProcessor) processDocument(t *document.DocumentTransform) {
	slog.Debug(">>LocalDocumentProcessor.processDocument")
	defer slog.Debug("<<LocalDocumentProcessor.processDocument")

	output := document.DocumentTransform{}

	// build a local file path
	lp.fullFilePath = filepath.Join(lp.localFilePath, t.Doc.Name)

	// create the local file to save the document to
	file, err := os.Create(lp.fullFilePath)
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
	file, err = os.Open(lp.fullFilePath)
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

func (lp *LocalDocumentProcessor) readConfigurationSettings() error {
	lp.localFilePath = os.Getenv("LOCAL_STORAGE_PATH")
	if len(lp.localFilePath) == 0 {
		return errors.New("environment variable LOCAL_STORAGE_PATH is not present")
	}

	return nil
}
