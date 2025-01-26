package local

import (
	"context"

	"github.com/KyleBrandon/scriptoria/pkg/document"
)

type (
	LocalStorageContext struct {
		ctx   context.Context
		store LocalDriveStore

		documents chan document.Document
	}

	// LocalDriveStore is used to access the database
	LocalDriveStore interface{}
)
