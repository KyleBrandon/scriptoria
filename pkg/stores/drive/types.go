package drive

import (
	"time"

	"google.golang.org/api/drive/v3"
)

type GoogleDrive struct {
	//
	watchFolderID   string
	webhookURL      string
	credentialsFile string

	driveService   *drive.Service
	channelID      string
	lastSearchTime time.Time
}
