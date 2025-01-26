package local

import "context"

func NewStorage(ctx context.Context, store LocalDriveStore) *LocalDocumentStorage {
	drive := &LocalDocumentStorage{}

	drive.ctx = ctx
	drive.store = store

	return drive
}

func (ld *LocalDocumentStorage) Initialize() error {
	return nil
}
