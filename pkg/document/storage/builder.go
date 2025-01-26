package storage

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
	"github.com/KyleBrandon/scriptoria/pkg/document/storage/gdrive"
	"github.com/KyleBrandon/scriptoria/pkg/document/storage/local"
)

func BuildDocumentStorage(ctx context.Context, storeName string, queries *database.Queries, mux *http.ServeMux) (document.DocumentStorage, error) {
	slog.Debug(">>buildDocumentStorage")
	defer slog.Debug("<<buildDocumentStorage")

	var storage document.DocumentStorage
	switch storeName {
	case "Google Drive":
		storage = gdrive.NewStorage(ctx, queries, mux)
	case "Local":
		storage = local.NewStorage(ctx, queries)
	}

	err := storage.Initialize()
	if err != nil {
		slog.Error("Failed to initialize the storage", "storeName", storeName, "error", err)
		return nil, err
	}

	return storage, nil
}
