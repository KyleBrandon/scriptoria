package storage

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
	"github.com/KyleBrandon/scriptoria/pkg/document/storage/gdrive"
	"github.com/KyleBrandon/scriptoria/pkg/document/storage/local"
)

func BuildDocumentStorage(storeName string, queries *database.Queries, mux *http.ServeMux) (document.Storage, error) {
	slog.Debug(">>buildDocumentStorage")
	defer slog.Debug("<<buildDocumentStorage")

	var storage document.Storage
	switch storeName {
	case "Google Drive":
		storage = gdrive.New(queries, mux)
	case "Local":
		storage = local.New(queries)
	default:
		return nil, errors.New("invalid storage type")
	}

	return storage, nil
}
