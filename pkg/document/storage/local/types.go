package local

import "context"

type (
	LocalDocumentStorage struct {
		ctx   context.Context
		store LocalDriveStore
	}

	LocalDriveStore interface{}
)
