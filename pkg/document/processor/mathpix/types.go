package mathpix

import (
	"context"
	"sync"

	"github.com/KyleBrandon/scriptoria/pkg/document"
)

// Mathpix API endpoint
const (
	MathpixPdfApiURL = "https://api.mathpix.com/v3/pdf"
)

// Polling interval (seconds)
const MathpixPollInterval = 5

type (
	MathpixErrorInfo struct {
		ID      string `json:"id,omitempty"`
		Message string `json:"message,omitempty"`
	}

	// UploadResponse represents the initial response from Mathpix after uploading a PDF
	UploadResponse struct {
		PdfID     string           `json:"pdf_id"`
		Error     string           `json:"error,omitempty"`
		ErrorInfo MathpixErrorInfo `json:"error_info,omitempty"`
	}

	// PollResponse represents the response when polling for PDF processing results
	PollResponse struct {
		Status      string `json:"status"`
		PdfMarkdown string `json:"pdf_md,omitempty"`
	}

	// Identify the database methos for the Mathpix processor
	mathpixDocumentStore interface{}

	MathpixDocumentProcessor struct {
		//
		mathpixAppID  string
		mathpixAppKey string

		ctx        context.Context
		cancelFunc context.CancelFunc
		wg         *sync.WaitGroup
		store      mathpixDocumentStore
		inputCh    chan *document.DocumentTransform
		outputCh   chan *document.DocumentTransform
	}
)
