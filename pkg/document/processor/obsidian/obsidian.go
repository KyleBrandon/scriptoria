package obsidian

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/KyleBrandon/scriptoria/internal/config"
	"github.com/KyleBrandon/scriptoria/pkg/document"
)

// NewObsidianProcessor will return a processor that will add a link to the Markdown file to the original PDF attachment.
func NewObsidianProcessor() *ObsidianDocumentPostProcessor {
	op := ObsidianDocumentPostProcessor{}

	return &op
}

func (op *ObsidianDocumentPostProcessor) Initialize(tempStoragePath string, bundles []config.StorageBundle) error {
	slog.Debug(">>ObsidianDocumentPostProcessor.Initialize")
	defer slog.Debug("<<ObsidianDocumentPostProcessor.Initialize")
	op.tempStoragePath = tempStoragePath

	return nil
}

func (op *ObsidianDocumentPostProcessor) Process(document *document.Document, reader io.ReadCloser) (io.ReadCloser, error) {
	slog.Debug(">>Obsidian.Process")
	defer slog.Debug("<<Obsidian.Process")

	sourceName := document.Name

	// save the markdown note to the notes folder in Obsidian
	markdown, err := op.saveMarkdownNote(sourceName, reader)
	if err != nil {
		return nil, err
	}

	return io.NopCloser(strings.NewReader(markdown)), nil
}

func (op *ObsidianDocumentPostProcessor) saveMarkdownNote(sourceName string, reader io.ReadCloser) (string, error) {
	// TODO: we should copy the notes markdown file then append the link to it instead of reading it all into memory
	markdownDocument, err := io.ReadAll(reader)
	if err != nil {
		slog.Error("Failed to read the output transform", "error", err)
		return "", err
	}

	// We want to append a link to the original scanned PDF at the end of the note
	output := fmt.Sprintf("%s\n\n![[%s]]", markdownDocument, sourceName)

	return output, nil
}
