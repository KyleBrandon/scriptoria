package local

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/KyleBrandon/scriptoria/pkg/document"
)

func New(store LocalDriveStore) *LocalStorageContext {
	drive := &LocalStorageContext{}

	drive.store = store

	return drive
}

func (ld *LocalStorageContext) Initialize(ctx context.Context, documents chan document.Document) error {
	ld.ctx = ctx
	return nil
}

func (ld *LocalStorageContext) StartWatching() error {
	return nil
}

func (ld *LocalStorageContext) GetFileReader(sourceFileID string) (io.ReadCloser, error) {
	return nil, errors.ErrUnsupported
}

func (ld *LocalStorageContext) Write(document document.Document, reader io.Reader) error {
	outputPath := "/Users/kyle/workspaces/scriptoria/download"
	// Create output file
	outFile, err := os.Create(filepath.Join(outputPath, document.Name))
	if err != nil {
		slog.Error("Unable to create local file", "error", err)
		return err
	}

	defer outFile.Close()

	// Copy file content to the local file
	_, err = io.Copy(outFile, reader)
	if err != nil {
		slog.Error("Unable to save file", "error", err)
		return err
	}
	return nil
}
