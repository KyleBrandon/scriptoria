package processor

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type BundleProcessor struct {
	tempStoragePath string
	notesPath       string
	attachmentsPath string
}

// NewBundleProcessor will return a processor that will bundle the Markdown document and PDF attachment into the specific folder locations.
func NewBundleProcessor(notesPath, attachmentsPath string) *BundleProcessor {
	bp := &BundleProcessor{}

	bp.notesPath = notesPath
	bp.attachmentsPath = attachmentsPath

	return bp
}

func (bp *BundleProcessor) Initialize(tempStoragePath string) error {
	bp.tempStoragePath = tempStoragePath
	return nil
}

func (bp *BundleProcessor) Process(sourceName string, reader io.ReadCloser) (io.ReadCloser, error) {
	slog.Debug(">>LocalDocumentProcessor.processDocument")
	defer slog.Debug("<<LocalDocumentProcessor.processDocument")

	// write the output to the notes.
	filePath := bp.createNotesFilePath(sourceName)
	err := CopyFileFromReader(filePath, reader)
	if err != nil {
		slog.Error("Failed to copy the processed document", "sourceName", sourceName, "error", err)
		return nil, err
	}

	// write the original pdf to the attachments folderr in Obsidian
	err = bp.copyAttachment(sourceName)
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

func (bp *BundleProcessor) createNotesFilePath(sourceName string) string {
	name := strings.TrimSuffix(sourceName, filepath.Ext(sourceName))
	name = fmt.Sprintf("%s.md", name)
	filePath := filepath.Join(bp.notesPath, name)

	return filePath
}

func (bp *BundleProcessor) copyAttachment(sourceName string) error {
	srcPath := filepath.Join(bp.tempStoragePath, sourceName)
	destPath := filepath.Join(bp.attachmentsPath, sourceName)

	// Copy the original document to the attachements folder
	err := copyFile(srcPath, destPath)
	if err != nil {
		slog.Error("Failed to copy the notes file", "error", err)
		return err
	}

	return nil
}
