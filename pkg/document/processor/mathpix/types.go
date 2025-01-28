package mathpix

// Mathpix API endpoint
const (
	MathpixPdfApiURL = "https://api.mathpix.com/v3/pdf"
)

// Polling interval (seconds)
const MathpixPollInterval = 5

// UploadResponse represents the initial response from Mathpix after uploading a PDF
type UploadResponse struct {
	PdfID string `json:"pdf_id"`
}

// PollResponse represents the response when polling for PDF processing results
type PollResponse struct {
	Status      string `json:"status"`
	PdfMarkdown string `json:"pdf_md,omitempty"`
}

type MathpixDocumentProcessor struct {
	//
	markdownFileLocation string
	mathpixAppID         string
	mathpixAppKey        string
}
