package drive

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func NewHandler(mux *http.ServeMux) {
	h := &Handler{}

	h.initConfig()

	err := h.getDriveService()
	if err != nil {
		return
	}

	mux.HandleFunc("POST /webhook", h.webhookHandler)
	h.subscribeToFolderChanges()
}

// Initialize environment variables
func (h *Handler) initConfig() {
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
func (h *Handler) subscribeToFolderChanges() {
	uniqueChannelID := uuid.New().String()

	req := &drive.Channel{
		Id:         uniqueChannelID,
		Type:       "web_hook",
		Address:    h.webhookURL,
		Expiration: time.Now().Add(24 * time.Hour).UnixMilli(),
	}

	// Watch for changes in the folder
	_, err := h.driveService.Files.Watch(h.watchFolderID, req).Do()
	if err != nil {
		slog.Error("Error subscribing to folder changes", "error", err)
		return
	}

	fmt.Println("✅ Subscribed to folder changes:", h.watchFolderID)
}

// Webhook handler for receiving Google Drive notifications
func (h *Handler) webhookHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("📌 Google Drive Notification Received!")

	// Extract headers sent by Google Drive
	resourceState := r.Header.Get("X-Goog-Resource-State")
	resourceID := r.Header.Get("X-Goog-Resource-ID")

	fmt.Printf("Resource State: %s, Resource ID: %s\n", resourceState, resourceID)

	if resourceState == "sync" {
		fmt.Println("✅ Google Drive sync notification received, ignoring.")
		return
	}

	// Check for new or modified files
	h.checkForNewOrModifiedFiles()

	w.WriteHeader(http.StatusOK)
}

// Check for new or modified files in the folder
func (h *Handler) checkForNewOrModifiedFiles() {
	query := fmt.Sprintf("'%s' in parents", h.watchFolderID)

	fileList, err := h.driveService.Files.List().Q(query).Fields("files(id, name, modifiedTime)").Do()
	if err != nil {
		log.Fatalf("❌ Error fetching files: %v", err)
	}

	if len(fileList.Files) == 0 {
		fmt.Println("⚠ No files found.")
		return
	}

	fmt.Println("📂 New or Modified Files:")
	for _, file := range fileList.Files {
		fmt.Printf("- %s (ID: %s, Modified: %s)\n", file.Name, file.Id, file.ModifiedTime)
	}
}

// Check Google Drive for new or modified PDFs
func (h *Handler) checkForNewOrModifiedPDFs() {
	query := "mimeType='application/pdf'"
	if h.watchFolderID != "" {
		query += fmt.Sprintf(" and '%s' in parents", h.watchFolderID)
	}

	fileList, err := h.driveService.Files.List().Q(query).Fields("files(id, name, modifiedTime)").Do()
	if err != nil {
		log.Fatalf("Error fetching files: %v", err)
	}

	fmt.Println("Checking for new or modified PDFs...")
	for _, file := range fileList.Files {
		fmt.Printf("PDF Found: %s (ID: %s, Modified: %s)\n", file.Name, file.Id, file.ModifiedTime)
	}
}
