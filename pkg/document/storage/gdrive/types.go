package gdrive

import (
	"context"
	"net/http"
	"time"

	"github.com/KyleBrandon/scriptoria/internal/database"
	"google.golang.org/api/drive/v3"
)

type GoogleDriveStorage struct {
	ctx   context.Context
	store GoogleDriveStore
	mux   *http.ServeMux

	// TODO: move these to the configuration file
	// environment settings
	watchFolderID   string
	webhookURL      string
	credentialsFile string
	expiration      int64

	driveService   *drive.Service
	channelID      string
	lastSearchTime time.Time
}

type GoogleDriveStore interface {
	CreateGoogleDriveWatch(ctx context.Context, arg database.CreateGoogleDriveWatchParams) (database.GoogleDriveWatch, error)
	GetLatestGoogleDriveWatch(ctx context.Context) (database.GoogleDriveWatch, error)
}
