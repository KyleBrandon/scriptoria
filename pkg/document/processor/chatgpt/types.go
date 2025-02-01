package chatgpt

import (
	"context"

	"github.com/KyleBrandon/scriptoria/pkg/document"
)

// Mathpix API endpoint
const (
	MathpixPdfApiURL = "https://api.mathpix.com/v3/pdf"
)

// Polling interval (seconds)
const MathpixPollInterval = 5

type (

	// Identify the database methos for the Mathpix processor
	chatgptDocumentStore interface{}

	ChatgptDocumentProcessor struct {
		chatgptAPIKey string

		ctx      context.Context
		store    chatgptDocumentStore
		inputCh  chan *document.DocumentTransform
		outputCh chan *document.DocumentTransform
	}
)
