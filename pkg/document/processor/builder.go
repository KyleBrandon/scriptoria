package processor

import (
	"log/slog"

	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
	"github.com/KyleBrandon/scriptoria/pkg/document/processor/mathpix"
)

func BuildProcessor(processorName string, queries *database.Queries) (document.DocumentProcessor, error) {
	slog.Debug(">>buildDocumentProcessor")
	defer slog.Debug("<<buildDocumentProcessor")

	var processor document.DocumentProcessor
	switch processorName {
	case "Mathpix":
		processor = mathpix.New(queries)
	}

	return processor, nil
}
