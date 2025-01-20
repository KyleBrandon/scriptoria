package drive

import (
	"google.golang.org/api/drive/v3"
)

type Handler struct {
	//
	watchFolderID   string
	webhookURL      string
	credentialsFile string

	driveService *drive.Service
	channelID    string
}
