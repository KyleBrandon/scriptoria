package gdrive

import (
	"context"
	"net/http"

	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
	"google.golang.org/api/drive/v3"
)

type GDriveStorageContext struct {
	ctx   context.Context
	store GoogleDriveStore
	mux   *http.ServeMux

	// environment settings
	watchFolderID   string
	webhookURL      string
	credentialsFile string
	expiration      int64

	driveService *drive.Service
	channelID    string

	documents chan document.Document
}

type GoogleDriveStore interface {
	CreateGoogleDriveWatch(ctx context.Context, arg database.CreateGoogleDriveWatchParams) (database.GoogleDriveWatch, error)
	GetLatestGoogleDriveWatch(ctx context.Context) (database.GoogleDriveWatch, error)
}
