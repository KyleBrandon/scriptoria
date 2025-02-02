package gdrive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Create a new Google Drive storage context
func New(store GoogleDriveStore, mux *http.ServeMux) *GDriveStorageContext {
	drive := &GDriveStorageContext{}

	drive.store = store
	drive.mux = mux

	return drive
}

// Initialize the Google Drive storage watcher
func (gd *GDriveStorageContext) Initialize(ctx context.Context) error {
	slog.Debug(">>GoogleDrive Initialize")
	defer slog.Debug("<<GoogleDrive Initialize")

	gd.documents = make(chan *document.Document, 10)

	gd.ctx = ctx
	err := gd.readConfigurationSettings()
	if err != nil {
		return err
	}

	err = gd.getDriveService()
	if err != nil {
		return err
	}

	return nil
}

// StartWatching for files in the Google Drive folder
func (gd *GDriveStorageContext) StartWatching() (chan *document.Document, error) {
	// register the webhook for Google Drive
	err := gd.registerWebhook()
	if err != nil {
		return nil, err
	}

	// Determine if we should renew the watch channel
	err = gd.createWatchChannel()
	if err != nil {
		slog.Error("Failed to crate watch channel", "error", err)
		return nil, err
	}

	// Do an initial query of the files that are in the folder
	go gd.QueryFiles()

	return gd.documents, nil
}

// QueryFiles from the watch folder and send them on the channel
// TODO: send files all at once instead of one at a time
func (gd *GDriveStorageContext) QueryFiles() {
	slog.Debug(">>GoogleDrive.checkForNewOrModifiedFiles")
	defer slog.Debug("<<GoogleDrive.checkForNewOrModifiedFiles")

	// build the query string to find the new fines in Google Drive
	query := gd.buildFileSearchQuery()

	fileList, err := gd.driveService.Files.List().Q(query).Fields("files(id, name, mimeType, createdTime, modifiedTime)").Do()
	if err != nil {
		slog.Error("Failed to fetch files", "error", err)
		return
	}

	if len(fileList.Files) == 0 {
		slog.Debug("No files found.")
		return
	}

	slog.Debug("GDriveStorage process file list", "file Count", len(fileList.Files))
	for _, file := range fileList.Files {
		slog.Debug("File:", "fileName", file.Name, "driveID", file.DriveId, "fileID", file.Id, "createdTime", file.CreatedTime, "modifiedTime", file.ModifiedTime)

		createdTime, err := time.Parse(time.RFC3339, file.CreatedTime)
		if err != nil {
			slog.Warn("Failed to parse the created time for the file", "fileID", file.Id, "fileName", file.Name, "createdTime", file.CreatedTime, "error", err)
		}

		modifiedTime, err := time.Parse(time.RFC3339, file.ModifiedTime)
		if err != nil {
			slog.Warn("Failed to parse the modified time for the file", "fileID", file.Id, "fileName", file.Name, "modifiedTime", file.ModifiedTime, "error", err)
		}

		document := document.Document{
			ID:           file.Id,
			Name:         file.Name,
			MimeType:     file.MimeType,
			CreatedTime:  createdTime,
			ModifiedTime: modifiedTime,
		}

		gd.documents <- &document
	}
}

// Write the given file to the storage
func (gd *GDriveStorageContext) Write(srcDoc *document.Document, reader io.ReadCloser) (*document.Document, error) {
	defer reader.Close()
	return &document.Document{}, errors.ErrUnsupported
}

// Get a io.Reader for the document
func (gd *GDriveStorageContext) GetDocumentReader(document *document.Document) (io.ReadCloser, error) {
	// Get the file data
	resp, err := gd.driveService.Files.Get(document.ID).Download()
	if err != nil {
		slog.Error("Unable to get the file reader", "error", err)
		return nil, err

	}

	return resp.Body, nil
}

// Initialize environment variables
func (gd *GDriveStorageContext) readConfigurationSettings() error {
	gd.credentialsFile = os.Getenv("GOOGLE_SERVICE_KEY_FILE")
	if len(gd.credentialsFile) == 0 {
		return errors.New("environment variable GOOGLE_SERVICE_KEY_FILE is not present")
	}

	gd.watchFolderID = os.Getenv("GOOGLE_WATCH_FOLDER_ID")
	if len(gd.watchFolderID) == 0 {
		return errors.New("environment variable GOOGLE_WATCH_FOLDER_ID is not present")
	}

	gd.webhookURL = os.Getenv("GOOGLE_WEBHOOK_URL")
	if len(gd.webhookURL) == 0 {
		return errors.New("environment variable GOOGLE_WEBHOOK_URL is not present")
	}

	return nil
}

// Authenticate using Service Account and return a Drive Service
func (gd *GDriveStorageContext) getDriveService() error {
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
func (gd *GDriveStorageContext) registerWebhook() error {
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

func (gd *GDriveStorageContext) createWatchChannel() error {
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
		slog.Debug("Use existing watch channel")
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
func (gd *GDriveStorageContext) webhookHandler(w http.ResponseWriter, r *http.Request) {
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
	if resourceState != "add" {
		slog.Debug("Webhook received non-add resource state", "channelID", channelID, "resourceID", resourceID, "resourceState", resourceState)
		return
	}

	// Check for new or modified files
	gd.QueryFiles()

	// check if the watch channel should be renewed
	gd.renewWatchChannelIfNeeded()

	w.WriteHeader(http.StatusOK)
}

func (gd *GDriveStorageContext) renewWatchChannelIfNeeded() {
	if time.Now().UnixMilli() > gd.expiration-60000 { // Renew 1 min before expiry
		gd.createWatchChannel() // Recreate the watch
	}
}

func (gd *GDriveStorageContext) buildFileSearchQuery() string {
	query := fmt.Sprintf("mimeType='application/pdf' and '%s' in parents", gd.watchFolderID)

	return query
}

func (gd *GDriveStorageContext) stopChannelWatch(channelID, resourceID string) {
	ch := &drive.Channel{
		Id:         channelID,
		ResourceId: resourceID,
	}

	// Stop watching the channel
	gd.driveService.Channels.Stop(ch).Do()
}
