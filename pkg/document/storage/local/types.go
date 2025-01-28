package local

import (
	"context"

	"github.com/KyleBrandon/scriptoria/pkg/document"
)

type (
	LocalStorageContext struct {
		ctx   context.Context
		store LocalDriveStore

		localFilePath string

		documents chan document.Document
	}

	// LocalDriveStore is used to access the database
	LocalDriveStore interface{}
)
