package gdrive

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func NewStorage(ctx context.Context, store GoogleDriveStore, mux *http.ServeMux) *GoogleDriveStorage {
	drive := &GoogleDriveStorage{}

	drive.ctx = ctx
	drive.store = store
	drive.mux = mux

	return drive
}

func (gd *GoogleDriveStorage) Initialize() error {
	slog.Debug(">>GoogleDrive Initialize")
	defer slog.Debug("<<GoogleDrive Initialize")

	err := gd.readConfigurationSettings()
	if err != nil {
		return err
	}

	err = gd.getDriveService()
	if err != nil {
		return err
	}

	// register the webhook for Google Drive
	err = gd.registerWebhook()
	if err != nil {
		return err
	}

	// Determine if we should renew the watch channel
	err = gd.createWatchChannel()
	if err != nil {
		slog.Error("Failed to crate watch channel", "error", err)
		return err
	}

	return nil
}

func (gd *GoogleDriveStorage) WatchForFiles(docs chan<- document.Document) error {
	return nil
}

// Initialize environment variables
func (gd *GoogleDriveStorage) readConfigurationSettings() error {
	gd.credentialsFile = os.Getenv("GOOGLE_SERVICE_KEY_FILE")
	gd.watchFolderID = os.Getenv("GOOGLE_WATCH_FOLDER_ID")
	gd.webhookURL = os.Getenv("GOOGLE_WEBHOOK_URL")

	return nil
}

// Authenticate using Service Account and return a Drive Service
func (gd *GoogleDriveStorage) getDriveService() error {
	// Load service account JSON
	data, err := os.ReadFile(gd.credentialsFile)
	if err != nil {
		slog.Error("Unable to read service account file", "error", err)
		return err
	}

	// Authenticate with Google Drive API using Service Account
	creds, err := google.CredentialsFromJSON(context.Background(), data, drive.DriveScope)
	if err != nil {
		slog.Error("Unable to parse credentials", "error", err)
		return err
	}

	// Create an HTTP client using TokenSource
	client := oauth2.NewClient(context.Background(), creds.TokenSource)

	// Create Google Drive service
	service, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		slog.Error("Unable to create Drive client", "error", err)
		return err
	}

	gd.driveService = service

	return nil
}

// Subscribe to folder changes
func (gd *GoogleDriveStorage) registerWebhook() error {
	slog.Debug(">>registerWebhook")
	defer slog.Debug("<<registerWebhook")

	// Register the webhook call back
	u, err := url.Parse(gd.webhookURL)
	if err != err {
		slog.Error("Failed to parse the GOOGLE_WEBHOOK_URL", "error", err)
		return err
	}

	gd.mux.HandleFunc(fmt.Sprintf("POST %s", u.Path), gd.webhookHandler)

	return nil
}

func (gd *GoogleDriveStorage) createWatchChannel() error {
	var createChannel bool
	// read the database for the last watch channel created
	watch, err := gd.store.GetLatestGoogleDriveWatch(gd.ctx)
	if err == nil {
		// we have a watch channel, see if it's still valid
		if watch.WebhookUrl != gd.webhookURL {
			createChannel = true
		}

		if time.Now().UnixMilli() > watch.ExpiresAt-60000 {
			createChannel = true
		}
	} else {
		// we don't have a watch channel so we need to create one
		createChannel = true
	}

	if !createChannel {
		slog.Info("Use existing watch channel")
		gd.channelID = watch.ChannelID
		gd.expiration = watch.ExpiresAt

		return nil
	}

	slog.Info("Create new watch channel")
	gd.channelID = uuid.New().String()
	gd.expiration = time.Now().Add(24 * time.Hour).UnixMilli()

	req := &drive.Channel{
		Id:         gd.channelID,
		Type:       "web_hook",
		Address:    gd.webhookURL,
		Expiration: gd.expiration,
	}

	// Watch for changes in the folder
	_, err = gd.driveService.Files.Watch(gd.watchFolderID, req).Do()
	if err != nil {
		return nil
	}

	args := database.CreateGoogleDriveWatchParams{
		ChannelID:  gd.channelID,
		ResourceID: gd.watchFolderID,
		ExpiresAt:  gd.expiration,
		WebhookUrl: gd.webhookURL,
	}

	_, err = gd.store.CreateGoogleDriveWatch(gd.ctx, args)
	if err != nil {
		return err
	}

	return nil
}

// Webhook handler for receiving Google Drive notifications
func (gd *GoogleDriveStorage) webhookHandler(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>GoogleDrive.webhookHandler")
	defer slog.Debug("<<GoogleDrive.webhookHandler")

	// Extract headers sent by Google Drive
	resourceState := r.Header.Get("X-Goog-Resource-State")
	channelID := r.Header.Get("X-Goog-Channel-ID")
	resourceID := r.Header.Get("X-Goog-Resource-ID")

	// did we receive a notification for an old channel?
	if channelID != gd.channelID {
		gd.stopChannelWatch(channelID, resourceID)
		return
	}

	// If we receive a 'sync' notification, ignore it for now.
	// We could use this for initialzing the state of the vault?
	if resourceState == "sync" {
		slog.Debug("Google Drive sync notification received, ignorning")
		return
	}

	// TODO: Setup Go Routine to process file change notifications

	// Check for new or modified files
	gd.checkForNewOrModifiedFiles(resourceState)

	// check if the watch channel should be renewed
	gd.renewWatchChannelIfNeeded()

	w.WriteHeader(http.StatusOK)
}

func (gd *GoogleDriveStorage) renewWatchChannelIfNeeded() {
	if time.Now().UnixMilli() > gd.expiration-60000 { // Renew 1 min before expiry
		gd.createWatchChannel() // Recreate the watch
	}
}

// Check for new or modified files in the folder
func (gd *GoogleDriveStorage) checkForNewOrModifiedFiles(resourceState string) {
	slog.Debug(">>GoogleDrive.checkForNewOrModifiedFiles")
	defer slog.Debug("<<GoogleDrive.checkForNewOrModifiedFiles")

	// build the query string to find the new fines in Google Drive
	query := buildFileSearchQuery(gd.watchFolderID, gd.lastSearchTime)

	fileList, err := gd.driveService.Files.List().Q(query).Fields("files(id, name, createdTime, modifiedTime)").Do()
	if err != nil {
		slog.Error("Failed to fetch files", "error", err)
		return
	}

	gd.lastSearchTime = time.Now()

	if len(fileList.Files) == 0 {
		slog.Debug("No files found.")
		return
	}

	slog.Info("State change", "resourceState", resourceState, "#files", len(fileList.Files))
	for _, file := range fileList.Files {
		slog.Info("Modified File:", "fileName", file.Name, "driveID", file.DriveId, "fileID", file.Id, "createdTime", file.CreatedTime, "modifiedTime", file.ModifiedTime)
		gd.downloadFile(file.Id, file.Name, "/Users/kyle/workspaces/scriptoria/download")
	}
}

func buildFileSearchQuery(resource string, lastSearchTime time.Time) string {
	query := fmt.Sprintf("mimeType='application/pdf' and '%s' in parents ", resource)
	if !lastSearchTime.IsZero() {
		lastCheckTime := lastSearchTime.Format(time.RFC3339)
		query = fmt.Sprintf("%s and createdTime > '%s'", query, lastCheckTime)
	}

	return query
}

// Download a file from Google Drive
func (gd *GoogleDriveStorage) downloadFile(fileID, fileName, outputPath string) error {
	slog.Debug(">>downloadFile")
	defer slog.Debug("<<downloadFile")

	// TODO: detect files that can be downloaded
	// Get the file data
	resp, err := gd.driveService.Files.Get(fileID).Download()
	if err != nil {
		slog.Error("Unable to download file", "error", err)
		return err

	}
	defer resp.Body.Close()

	// Create output file
	outFile, err := os.Create(filepath.Join(outputPath, fileName))
	if err != nil {
		slog.Error("Unable to create local file", "error", err)
		return err
	}

	defer outFile.Close()

	// Copy file content to the local file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		slog.Error("Unable to save file", "error", err)
		return err
	}

	slog.Debug("File downloaded successfully", "outputPath", outputPath)

	return nil
}

func (gd *GoogleDriveStorage) stopChannelWatch(channelID, resourceID string) {
	ch := &drive.Channel{
		Id:         channelID,
		ResourceId: resourceID,
	}

	// Stop watching the channel
	gd.driveService.Channels.Stop(ch).Do()
}
