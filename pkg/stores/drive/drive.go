package drive

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

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func Initialize(mux *http.ServeMux) error {
	h := &Handler{}

	h.readConfigurationSettings()

	err := h.getDriveService()
	if err != nil {
		return err
	}

	h.subscribeToFolderChanges(mux)

	return nil
}

// Initialize environment variables
func (h *Handler) readConfigurationSettings() {
	h.credentialsFile = os.Getenv("GOOGLE_SERVICE_KEY_FILE")
	h.watchFolderID = os.Getenv("GOOGLE_WATCH_FOLDER_ID")
	h.webhookURL = os.Getenv("GOOGLE_WEBHOOK_URL")
}

// Authenticate using Service Account and return a Drive Service
func (h *Handler) getDriveService() error {
	// Load service account JSON
	data, err := os.ReadFile(h.credentialsFile)
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

	h.driveService = service

	return nil
}

func (h *Handler) pollForChanges() {
	// Monitor changes in a loop
	var startPageToken string
	err := h.getDriveService()
	if err != nil {
		return
	}

	// Get startPageToken
	startTokenResp, err := h.driveService.Changes.GetStartPageToken().Do()
	if err != nil {
		slog.Error("Error getting start page token", "error", err)
		return
	}

	if len(startPageToken) == 0 {
		startPageToken = startTokenResp.StartPageToken
	}
	for {

		changeList, err := h.driveService.Changes.List(startPageToken).Spaces("drive").Do()
		if err != nil {
			slog.Error("Error getting changes", "error", err)
			continue
		}

		for _, change := range changeList.Changes {
			fmt.Printf(
				"Change detected: Kind: %v, Change Type: %v, File ID: %s, Drive ID: %s, File Name: %s, Mime Type: %v, Trashed: %v, Removed: %v\n",
				change.Kind,
				change.ChangeType,
				change.FileId,
				change.File.DriveId,
				change.File.Name,
				change.File.MimeType,
				change.File.Trashed,
				change.Removed)
		}

		// Update the page token
		if changeList.NewStartPageToken != "" {
			startPageToken = changeList.NewStartPageToken
		}
	}
}

// Subscribe to folder changes
func (h *Handler) subscribeToFolderChanges(mux *http.ServeMux) error {
	slog.Debug(">>subscribeToFolderChanges")
	defer slog.Debug("<<subscribeToFolderChanges")

	// Register the webhook call back
	u, err := url.Parse(h.webhookURL)
	if err != err {
		slog.Error("Failed to parse the GOOGLE_WEBHOOK_URL", "error", err)
		return err
	}

	mux.HandleFunc(fmt.Sprintf("POST %s", u.Path), h.webhookHandler)

	// Initialzie the channel to watch.  This will stay active for 24 hours then need to be recreated
	h.channelID = uuid.New().String()
	req := &drive.Channel{
		Id:         h.channelID,
		Type:       "web_hook",
		Address:    h.webhookURL,
		Expiration: time.Now().Add(24 * time.Hour).UnixMilli(),
	}

	// Watch for changes in the folder
	_, err = h.driveService.Files.Watch(h.watchFolderID, req).Do()
	if err != nil {
		slog.Error("Error subscribing to folder changes", "error", err)
		return err
	}

	return nil
}

// Webhook handler for receiving Google Drive notifications
func (h *Handler) webhookHandler(w http.ResponseWriter, r *http.Request) {
	// Extract headers sent by Google Drive
	resourceState := r.Header.Get("X-Goog-Resource-State")
	resourceID := r.Header.Get("X-Goog-Resource-ID")
	channelID := r.Header.Get("X-Goog-Channel-ID")

	// did we receive a notification for an old channel?
	if channelID != h.channelID {
		h.stopChannelWatch(channelID)
		return
	}

	slog.Debug("Resource changed", "Channel ID", channelID, "Resource State", resourceState, "Resource ID", resourceID)

	// If we receive a 'sync' notification, ignore it for now.
	// We could use this for initialzing the state of the vault?
	if resourceState == "sync" {
		slog.Debug("Google Drive sync notification received, ignorning")
		return
	}

	// Check for new or modified files
	h.checkForNewOrModifiedFiles()

	w.WriteHeader(http.StatusOK)
}

// Check for new or modified files in the folder
func (h *Handler) checkForNewOrModifiedFiles() {
	slog.Debug(">>checkForNewOrModifiedFiles")
	defer slog.Debug("<<checkForNewOrModifiedFiles")

	query := fmt.Sprintf("mimeType='application/pdf' and '%s' in parents", h.watchFolderID)

	fileList, err := h.driveService.Files.List().Q(query).Fields("files(id, name, modifiedTime)").Do()
	if err != nil {
		slog.Error("Failed to fetch files", "error", err)
		return
	}

	if len(fileList.Files) == 0 {
		slog.Debug("No files found.")
		return
	}

	slog.Info("New or Modified Files", "# modified", len(fileList.Files))
	for _, file := range fileList.Files {
		slog.Info("Modified File:", "fileName", file.Name, "fileID", file.Id, "modifiedTime", file.ModifiedTime)
		h.downloadFile(file.Id, file.Name, "/Users/kyle/workspaces/scriptoria/download")
	}
}

// Download a file from Google Drive
func (h *Handler) downloadFile(fileID, fileName, outputPath string) error {
	slog.Debug(">>downloadFile")
	defer slog.Debug("<<downloadFile")

	// TODO: detect files that can be downloaded
	// Get the file data
	resp, err := h.driveService.Files.Get(fileID).Download()
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

func (h *Handler) stopChannelWatch(channelID string) {
	ch := &drive.Channel{
		Id:         channelID,
		ResourceId: h.watchFolderID,
	}

	// Stop watching the channel
	h.driveService.Channels.Stop(ch).Do()
}
