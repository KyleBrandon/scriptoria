package processor

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/KyleBrandon/scriptoria/internal/config"
	"github.com/KyleBrandon/scriptoria/pkg/document"
)

var ErrBundleNotFound = errors.New("could not find the bundle")

type BundleProcessor struct {
	tempStoragePath string
	bundles         []config.StorageBundle
}

// NewBundleProcessor will return a processor that will bundle the Markdown document and PDF attachment into the specific folder locations.
func NewBundleProcessor() *BundleProcessor {
	bp := &BundleProcessor{}

	return bp
}

func (lp *BundleProcessor) GetName() string {
	return "Bundle Document Processor"
}

func (bp *BundleProcessor) Initialize(tempStoragePath string, bundles []config.StorageBundle) error {
	bp.tempStoragePath = tempStoragePath
	bp.bundles = bundles
	return nil
}

func (bp *BundleProcessor) Process(document *document.Document, reader io.ReadCloser) (io.ReadCloser, error) {
	slog.Debug(">>LocalDocumentProcessor.processDocument")
	defer slog.Debug("<<LocalDocumentProcessor.processDocument")

	// write the output to the notes.
	filePath, err := bp.createNotesFilePath(document)
	if err != nil {
		return nil, err
	}

	err = CopyFileFromReader(filePath, reader)
	if err != nil {
		slog.Error("Failed to copy the processed document", "sourceName", document.Name, "error", err)
		return nil, err
	}

	// write the original pdf to the attachments folderr in Obsidian
	err = bp.copyAttachment(document)
	if err != nil {
		return nil, err
	}

	// send the document file back as a reader
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (bp *BundleProcessor) createNotesFilePath(document *document.Document) (string, error) {
	sourceName := document.Name
	bundle, err := bp.getBundle(document)
	if err != nil {
		return "", err
	}

	name := strings.TrimSuffix(sourceName, filepath.Ext(sourceName))
	name = fmt.Sprintf("%s.md", name)
	filePath := filepath.Join(bundle.DestNotesFolder, name)

	return filePath, nil
}

func (bp *BundleProcessor) copyAttachment(document *document.Document) error {
	sourceName := document.Name
	bundle, err := bp.getBundle(document)
	if err != nil {
		return err
	}

	srcPath := filepath.Join(bp.tempStoragePath, sourceName)
	destPath := filepath.Join(bundle.DestAttachmentsFolder, sourceName)

	// Copy the original document to the attachements folder
	err = copyFile(srcPath, destPath)
	if err != nil {
		slog.Error("Failed to copy the notes file", "error", err)
		return err
	}

	return nil
}

func (bp *BundleProcessor) getBundle(document *document.Document) (config.StorageBundle, error) {
	for _, b := range bp.bundles {
		if b.SourceFolder == document.StorageFolderID {
			return b, nil
		}
	}

	return config.StorageBundle{}, ErrBundleNotFound
}
