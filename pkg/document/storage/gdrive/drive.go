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
	"strings"
	"sync"
	"time"

	"github.com/KyleBrandon/scriptoria/internal/config"
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
	slog.Debug(">>GDriveStorageContext.New")
	defer slog.Debug("<<GDriveStorageContext.New")

	drive := &GDriveStorageContext{}

	drive.store = store
	drive.mux = mux
	drive.wg = &sync.WaitGroup{}

	return drive
}

// Initialize the Google Drive storage watcher
func (gd *GDriveStorageContext) Initialize(ctx context.Context, bundles []config.StorageBundle) error {
	slog.Debug(">>GoogleDrive Initialize")
	defer slog.Debug("<<GoogleDrive Initialize")

	gd.bundles = bundles
	gd.documents = make(chan *document.Document, 10)

	gd.ctx, gd.cancelFunc = context.WithCancel(ctx)
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

// Cancel the context and wait for any go routine to finish
func (gd *GDriveStorageContext) CancelAndWait() {
	gd.cancelFunc()

	gd.wg.Wait()
}

// StartWatching for files in the Google Drive folder
func (gd *GDriveStorageContext) StartWatching() (chan *document.Document, error) {
	// register the webhook for Google Drive
	err := gd.registerWebhook()
	if err != nil {
		return nil, err
	}

	// Determine if we should renew the watch channel
	err = gd.createWatchChannels()
	if err != nil {
		slog.Error("Failed to crate watch channel", "error", err)
		return nil, err
	}

	// schedule a timer to renew the watch channels
	gd.scheduleChannelRenewal()

	// Do an initial query of the files that are in the folder
	gd.wg.Add(1)
	go gd.QueryFiles()

	return gd.documents, nil
}

// QueryFiles from the watch folder and send them on the channel
// TODO: send files all at once instead of one at a time
func (gd *GDriveStorageContext) QueryFiles() {
	slog.Debug(">>GoogleDrive.checkForNewOrModifiedFiles")
	defer slog.Debug("<<GoogleDrive.checkForNewOrModifiedFiles")

	defer gd.wg.Done()

	// build the query string to find the new fines in Google Drive
	query := gd.buildFileSearchQuery()

	fileList, err := gd.driveService.Files.List().Q(query).Fields("files(id, name, parents, createdTime, modifiedTime)").Do()
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
			StorageDocumentID: file.Id,
			StorageFolderID:   file.Parents[0],
			Name:              file.Name,
			CreatedTime:       createdTime,
			ModifiedTime:      modifiedTime,
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
func (gd *GDriveStorageContext) GetReader(document *document.Document) (io.ReadCloser, error) {
	// Get the file data
	resp, err := gd.driveService.Files.Get(document.StorageDocumentID).Download()
	if err != nil {
		slog.Error("Unable to get the file reader", "error", err)
		return nil, err

	}

	return resp.Body, nil
}

func (gd *GDriveStorageContext) Archive(document *document.Document) error {
	// move the document to the archive folder
	file, err := gd.driveService.Files.Get(document.StorageDocumentID).Fields("parents").Do()
	if err != nil {
		return err
	}

	archiveFolderID := ""
	for _, b := range gd.bundles {
		if b.SourceFolder == document.StorageFolderID {
			archiveFolderID = b.ArchiveFolder
		}
	}

	if len(archiveFolderID) == 0 {
		return fmt.Errorf("failed to find an archive folder for document: %s in folder: %s", document.Name, document.StorageFolderID)
	}

	previousParents := strings.Join(file.Parents, ",")
	_, err = gd.driveService.Files.Update(document.StorageDocumentID, nil).
		AddParents(archiveFolderID).
		RemoveParents(previousParents).
		Fields("id, parents").
		Do()
	if err != nil {
		return err
	}

	return nil
}

// Initialize environment variables
func (gd *GDriveStorageContext) readConfigurationSettings() error {
	gd.credentialsFile = os.Getenv("GOOGLE_SERVICE_KEY_FILE")
	if len(gd.credentialsFile) == 0 {
		return errors.New("environment variable GOOGLE_SERVICE_KEY_FILE is not present")
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

// Webhook handler for receiving Google Drive notifications
func (gd *GDriveStorageContext) webhookHandler(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>GoogleDrive.webhookHandler")
	defer slog.Debug("<<GoogleDrive.webhookHandler")

	// Extract headers sent by Google Drive
	resourceState := r.Header.Get("X-Goog-Resource-State")
	channelID := r.Header.Get("X-Goog-Channel-ID")
	resourceID := r.Header.Get("X-Goog-Resource-ID")

	// did we receive a notification for an old channel?
	if !gd.watchChannelExists(channelID) {
		slog.Error("watch channel does not exist", "channelID", channelID)
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
	gd.wg.Add(1)
	go gd.QueryFiles()

	w.WriteHeader(http.StatusOK)
}

func (gd *GDriveStorageContext) watchChannelExists(channelID string) bool {
	for _, v := range gd.channelWatchMap {
		if v.ChannelID == channelID {
			return true
		}
	}

	return false
}

func (gd *GDriveStorageContext) scheduleChannelRenewal() {
	time.AfterFunc(30*time.Minute, func() {
		err := gd.createWatchChannels()
		if err != nil {
			slog.Error("scheduleChannelRenewal failed", "error", err)
		}
	})
}

func (gd *GDriveStorageContext) createWatchChannels() error {
	slog.Debug(">>GDrive.createWatchChannels")
	defer slog.Debug("<<GDrive.createWatchChannels")

	// create a list of folder (resource) ids to query and a map of folders to watch channel information
	resourceIds := make([]string, 0)
	gd.channelWatchMap = make(map[string]database.GoogleDriveWatch)
	for _, b := range gd.bundles {
		slog.Debug("Watching folder", "resourceID", b.SourceFolder)
		// build resource list of folders to query from the database
		resourceIds = append(resourceIds, b.SourceFolder)

		// build a map of expected watch entries with initial values
		gd.channelWatchMap[b.SourceFolder] = database.GoogleDriveWatch{
			ResourceID: b.SourceFolder,
			ChannelID:  uuid.New().String(),
			ExpiresAt:  time.Now().Add(24 * time.Hour).UnixMilli(),
			WebhookUrl: gd.webhookURL,
		}
	}

	// get any corresponding watch channels for the folders
	dbWatch, err := gd.store.GetWatchEntriesByFolderIDs(gd.ctx, resourceIds)
	if err != nil {
		// we failed to query any of the expected resource identifiers
		slog.Error("failed to query watch entries by folder ID", "error", err)
		return err
	}

	// loop through the previously configured channels and update the default map with their current settings
	for _, w := range dbWatch {
		gd.channelWatchMap[w.ResourceID] = w
	}

	// loop through the map of watch channels we need to maintain
	for _, v := range gd.channelWatchMap {
		if err = gd.createWatchChannel(v); err != nil {
			return err
		}
	}

	return nil
}

func (gd *GDriveStorageContext) createWatchChannel(wc database.GoogleDriveWatch) error {
	// for entries that were previously configured, determine if we need to re-create the channel
	if wc.ID != uuid.Nil {
		// consider it expired if it's been alive over 23 hours
		expired := time.Now().UnixMilli() > wc.ExpiresAt-60000
		if wc.WebhookUrl == gd.webhookURL && !expired {
			// we don't need to create a new channel as it current exists for the correct web hook and it's not expired
			slog.Debug("current channel is valid", "resourceID", wc.ResourceID, "channelID", wc.ChannelID)
			return nil
		} else {
			// the channel either expired or has a stale webhook URL
			wc.ChannelID = uuid.New().String()
			wc.ExpiresAt = time.Now().Add(24 * time.Hour).UnixMilli()
			wc.WebhookUrl = gd.webhookURL
		}
	}

	slog.Debug("createWatchChannel", "resourceID", wc.ResourceID, "channelID", wc.ChannelID)
	req := &drive.Channel{
		Id:         wc.ChannelID,
		Type:       "web_hook",
		Address:    wc.WebhookUrl,
		Expiration: wc.ExpiresAt,
	}

	// Watch for changes in the folder
	_, err := gd.driveService.Files.Watch(wc.ResourceID, req).Do()
	if err != nil {
		slog.Error("Failed to watch folder", "resourceID", wc.ResourceID, "error", err)
		return nil
	}

	dbc, err := gd.createOrUpdateChannel(wc)
	if err != nil {
		slog.Error("Failed to create or update the watch channel", "resourceID", wc.ResourceID, "channelID", wc.ChannelID, "error", err)
		return err
	}

	// save the newly created/updated watch channel in our map
	gd.channelWatchMap[wc.ResourceID] = dbc

	return nil
}

func (gd *GDriveStorageContext) createOrUpdateChannel(wc database.GoogleDriveWatch) (database.GoogleDriveWatch, error) {
	// if we don't have a database id then we need to create the channel
	if wc.ID == uuid.Nil {
		args := database.CreateGoogleDriveWatchParams{
			ChannelID:  wc.ChannelID,
			ResourceID: wc.ResourceID,
			ExpiresAt:  wc.ExpiresAt,
			WebhookUrl: wc.WebhookUrl,
		}

		return gd.store.CreateGoogleDriveWatch(gd.ctx, args)
	} else {
		// we're updating an existing channel
		args := database.UpdateGoogleDriveWatchParams{
			ID:         wc.ID,
			ChannelID:  wc.ChannelID,
			ResourceID: wc.ResourceID,
			ExpiresAt:  wc.ExpiresAt,
			WebhookUrl: wc.WebhookUrl,
		}

		return gd.store.UpdateGoogleDriveWatch(gd.ctx, args)
	}
}

func (gd *GDriveStorageContext) buildFileSearchQuery() string {
	query := "mimeType='application/pdf' and ("

	index := 0
	for k := range gd.channelWatchMap {
		if index != 0 {
			query = query + " or "
		}
		query = fmt.Sprintf("%s'%s' in parents", query, k)
		index++
	}

	query = query + ")"

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
