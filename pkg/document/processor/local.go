package processor

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/KyleBrandon/scriptoria/internal/config"
	"github.com/KyleBrandon/scriptoria/pkg/document"
)

type LocalDocumentProcessor struct {
	destinationPath string
}

// NewTempStorageProcessor will create a processor that will save the document to the configured temporary storage and return a reader to this location.
func NewTempStorageProcessor() *LocalDocumentProcessor {
	lp := &LocalDocumentProcessor{}

	return lp
}

func (lp *LocalDocumentProcessor) GetName() string {
	return "Local Document Processor"
}

func (lp *LocalDocumentProcessor) Initialize(tempStoragePath string, bundles []config.StorageBundle) error {
	lp.destinationPath = tempStoragePath
	return nil
}

func (lp *LocalDocumentProcessor) Process(document *document.Document, reader io.ReadCloser) (io.ReadCloser, error) {
	slog.Debug(">>LocalDocumentProcessor.processDocument")
	defer slog.Debug("<<LocalDocumentProcessor.processDocument")

	// build a local file path
	fullFilePath := filepath.Join(lp.destinationPath, document.Name)
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
