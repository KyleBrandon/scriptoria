package gdrive

import (
	"context"
	"net/http"
	"sync"

	"github.com/KyleBrandon/scriptoria/internal/config"
	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
	"google.golang.org/api/drive/v3"
)

type GDriveStorageContext struct {
	ctx        context.Context
	mux        *http.ServeMux
	cancelFunc context.CancelFunc
	wg         *sync.WaitGroup
	store      GoogleDriveStore

	// environment settings
	webhookURL      string
	credentialsFile string
	bundles         []config.StorageBundle
	channelWatchMap map[string]database.GoogleDriveWatch

	driveService *drive.Service
	documents    chan *document.Document
}

type GoogleDriveStore interface {
	CreateGoogleDriveWatch(ctx context.Context, arg database.CreateGoogleDriveWatchParams) (database.GoogleDriveWatch, error)
	GetWatchEntriesByFolderIDs(ctx context.Context, resourceIds []string) ([]database.GoogleDriveWatch, error)
	UpdateGoogleDriveWatch(ctx context.Context, arg database.UpdateGoogleDriveWatchParams) (database.GoogleDriveWatch, error)
}
