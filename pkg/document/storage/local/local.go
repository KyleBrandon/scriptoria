package local

import (
	"context"

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
