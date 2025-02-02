package obsidian

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/KyleBrandon/scriptoria/pkg/document"
)

func New(store obsidianDocumentStore, localStoragePath, attachmentsFolder, notesFolder string) *ObsidianDocumentPostProcessor {
	op := ObsidianDocumentPostProcessor{}
	op.store = store
	op.localStoragePath = localStoragePath
	op.attachmentsFolder = attachmentsFolder
	op.notesFolder = notesFolder

	return &op
}

func (op *ObsidianDocumentPostProcessor) Initialize(ctx context.Context) error {
	slog.Debug(">>ObsidianDocumentPostProcessor.Initialize")
	defer slog.Debug("<<ObsidianDocumentPostProcessor.Initialize")

	return nil
}

func (op *ObsidianDocumentPostProcessor) Process(t *document.DocumentTransform) error {
	slog.Debug(">>Obsidian.Process")
	defer slog.Debug("<<Obsidian.Process")

	err := op.saveMarkdownNote(t)
	if err != nil {
		return err
	}

	// write the original pdf to the attachments
	err = op.copyAttachment(t)
	if err != nil {
		return err
	}

	return nil
}

func (op *ObsidianDocumentPostProcessor) saveMarkdownNote(t *document.DocumentTransform) error {
	defer t.Reader.Close()

	buffer, err := io.ReadAll(t.Reader)
	if err != nil {
		slog.Error("Failed to read the output transform", "error", err)
		return err
	}

	// Check to see if the document is surrounded in a "markdown" code block.  If so, remove it.
	markdownDocument := strings.TrimPrefix(strings.TrimSuffix(string(buffer), "```"), "```markdown")

	output := fmt.Sprintf("%s\n\n![[%s]]", markdownDocument, t.SourceName)

	// write the output to the notes.

	name := strings.TrimSuffix(t.SourceName, filepath.Ext(t.SourceName))
	name = fmt.Sprintf("%s.md", name)

	filePath := filepath.Join(op.notesFolder, name)
	file, err := os.Create(filePath)
	if err != nil {
		slog.Error("Failed to create the notes file", "filePath", filePath, "error", err)
		return err
	}

	defer file.Close()

	_, err = file.WriteString(output)
	if err != nil {
		slog.Error("Failed to write the notes file", "filePath", filePath, "error", err)
		return err
	}

	return nil
}

func (op *ObsidianDocumentPostProcessor) copyAttachment(t *document.DocumentTransform) error {
	srcPath := filepath.Join(op.localStoragePath, t.SourceName)
	destPath := filepath.Join(op.attachmentsFolder, t.SourceName)

	// Copy the original document to the attachements folder
	err := CopyFile(srcPath, destPath)
	if err != nil {
		slog.Error("Failed to copy the notes file", "error", err)
		return err
	}

	// delete the temp attachemt
	err = os.Remove(srcPath)
	if err != nil {
		slog.Error("Failed to remove the source document", "srcPath", srcPath, "error", err)
		return err
	}

	return nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	// Open the source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create the destination file
	destinationFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destinationFile.Close()

	// Copy the content from source to destination
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Ensure the destination file gets flushed and closed properly
	err = destinationFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}
