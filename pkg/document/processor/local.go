package processor

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

type documentStore interface{}

type LocalDocumentProcessor struct {
	destinationPath string
}

// NewTempStorageProcessor will create a processor that will save the document to the configured temporary storage and return a reader to this location.
func NewTempStorageProcessor() *LocalDocumentProcessor {
	lp := &LocalDocumentProcessor{}

	return lp
}

func (lp *LocalDocumentProcessor) Initialize(tempStoragePath string) error {
	lp.destinationPath = tempStoragePath
	return nil
}

func (lp *LocalDocumentProcessor) Process(sourceName string, reader io.ReadCloser) (io.ReadCloser, error) {
	slog.Debug(">>LocalDocumentProcessor.processDocument")
	defer slog.Debug("<<LocalDocumentProcessor.processDocument")

	// build a local file path
	fullFilePath := filepath.Join(lp.destinationPath, sourceName)
	err := CopyFileFromReader(fullFilePath, reader)
	if err != nil {
		return nil, err
	}

	// open the newly created file for the reader
	file, err := os.Open(fullFilePath)
	if err != nil {
		return nil, err
	}

	// set the new transform reader
	return file, nil
}
