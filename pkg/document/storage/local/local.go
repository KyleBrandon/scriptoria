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

func New(store LocalDriveStore) *LocalStorageContext {
	drive := &LocalStorageContext{}

	drive.store = store

	return drive
}

func (ld *LocalStorageContext) Initialize(ctx context.Context) error {
	ld.ctx = ctx
	err := ld.readConfigurationSettings()
	if err != nil {
		return err
	}
	return nil
}

func (ld *LocalStorageContext) readConfigurationSettings() error {
	ld.localFilePath = os.Getenv("LOCAL_STORAGE_PATH")
	if len(ld.localFilePath) == 0 {
		return errors.New("environment variable LOCAL_STORAGE_PATH is not present")
	}

	return nil
}

func (ld *LocalStorageContext) StartWatching() (chan *document.Document, error) {
	return nil, errors.ErrUnsupported
}

func (ld *LocalStorageContext) GetDocumentReader(document *document.Document) (io.ReadCloser, error) {
	file, err := os.Open(document.ID)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (ld *LocalStorageContext) Write(srcDoc *document.Document, reader io.ReadCloser) (*document.Document, error) {
	defer reader.Close()

	filePath := filepath.Join(ld.localFilePath, srcDoc.Name)

	// Create output file
	outFile, err := os.Create(filePath)
	if err != nil {
		slog.Error("Unable to create local file", "error", err)
		return &document.Document{}, err
	}

	defer outFile.Close()

	// Copy file content to the local file
	_, err = io.Copy(outFile, reader)
	if err != nil {
		slog.Error("Unable to save file", "error", err)
		return &document.Document{}, err
	}

	destDoc := document.Document{
		ID:   filePath,
		Name: srcDoc.Name,
		// TODO: how to fix this?
		MimeType:     "",
		CreatedTime:  time.Now(),
		ModifiedTime: time.Now(),
	}

	return &destDoc, nil
}
