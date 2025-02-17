package mathpix

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/KyleBrandon/scriptoria/internal/config"
	"github.com/KyleBrandon/scriptoria/pkg/document"
)

// NewMathpixProcessor will create a document processor to send to the Mathix PDF API to get a Markdown version of the document.
// The reader that is returned will be for an in-memory version of the Markdown file.
func NewMathpixProcessor() *MathpixDocumentProcessor {
	mp := &MathpixDocumentProcessor{}

	mp.readConfigurationSettings()
	return mp
}

func (lp *MathpixDocumentProcessor) GetName() string {
	return "Mathpix Document Processor"
}

func (mp *MathpixDocumentProcessor) Initialize(tempStoragePath string, bundles []config.StorageBundle) error {
	mp.tempStoragePath = tempStoragePath

	err := mp.readConfigurationSettings()
	if err != nil {
		slog.Error("Failed to read the Mathpix configuration settings", "error", err)
		return err
	}

	return nil
}

func (mp *MathpixDocumentProcessor) Process(document *document.Document, reader io.ReadCloser) (io.ReadCloser, error) {
	slog.Debug(">>MathpixDocumentProcessor.processDocument")
	defer slog.Debug("<<MathpixDocumentProcessor.processDocument")

	sourceName := document.Name

	// Upload PDF to Mathpix
	pdfID, err := mp.sendDocumentToMathpix(sourceName, reader)
	if err != nil {
		slog.Error("Error uploading PDF", "error", err)
		return nil, err
	}

	// Poll for results
	err = mp.pollForResults(pdfID)
	if err != nil {
		slog.Error("Error getting results", "error", err)
		return nil, err
	}

	markdownText, err := mp.queryConversionResults(pdfID)
	if err != nil {
		slog.Error("Failed to query conversion results", "error", err)
		return nil, err
	}

	// save the original markdown from Mathpx to the temp folder
	// name := strings.TrimSuffix(sourceName, filepath.Ext(sourceName))
	// name = fmt.Sprintf("%s.md", name)
	// filePath := filepath.Join(mp.tempStoragePath, name)
	// processor.CopyFileFromReader(filePath, io.NopCloser(strings.NewReader(markdownText)))

	// set the new reader
	r := io.NopCloser(strings.NewReader(markdownText))

	return r, nil
}

// Initialize environment variables
func (mp *MathpixDocumentProcessor) readConfigurationSettings() error {
	mp.mathpixAppID = os.Getenv("MATHPIX_APP_ID")
	if len(mp.mathpixAppID) == 0 {
		return errors.New("environment variable MATHPIX_APP_ID is not present")
	}

	mp.mathpixAppKey = os.Getenv("MATHPIX_APP_KEY")
	if len(mp.mathpixAppKey) == 0 {
		return errors.New("environment variable MATHPIX_APP_KEY is not present")
	}

	return nil
}

// UploadPDF uploads a PDF file to Mathpix and returns the Job ID
func (mp *MathpixDocumentProcessor) sendDocumentToMathpix(name string, reader io.Reader) (string, error) {
	slog.Debug(">>sendDocumentToMathpix")
	defer slog.Debug("<<sendDocumentToMathpix")

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		slog.Error("Failed to create form file", "error", err)
		return "", err
	}

	// copy the document input to the request body
	_, err = io.Copy(part, reader)
	if err != nil {
		slog.Error("Failed to copy file to form part", "error", err)
		return "", err
	}
	writer.Close()

	// Create HTTP request
	req, err := mp.newRequest("POST", MathpixPdfApiURL, body)
	if err != nil {
		slog.Error("Failed to create POST request for mathpix API", "error", err)
		return "", err
	}

	// Set additional headers
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// send the request
	respBody, err := mp.doRequest(req)
	if err != nil {
		slog.Error("Failed to send mathpix request", "error", err)
		return "", err
	}

	// Process the response for the PDF id
	var uploadResp UploadResponse
	err = json.Unmarshal(respBody, &uploadResp)
	if err != nil {
		slog.Error("Failed to unmarshal mathpix response", "error", err)
		return "", err
	}

	if len(uploadResp.Error) != 0 {
		return "", fmt.Errorf("mathpix error: %s, ErrorInfo.ID=%s, ErrorInfo.Message=%s", uploadResp.Error, uploadResp.ErrorInfo.ID, uploadResp.ErrorInfo.Message)
	}

	return uploadResp.PdfID, nil
}

// PollForResults polls Mathpix API for PDF processing status
func (mp *MathpixDocumentProcessor) pollForResults(pdfID string) error {
	slog.Debug(">>PollForResults", "pdfID", pdfID)
	defer slog.Debug("<<PollForResults")

	pollURL := fmt.Sprintf("%s/%s", MathpixPdfApiURL, pdfID)

	// TODO: This would run forever
	for {
		req, err := mp.newRequest("GET", pollURL, nil)
		if err != nil {
			slog.Error("Failed to create GET request for mathpix document status", "error", err)
			return err
		}

		bodyContents, err := mp.doRequest(req)
		if err != nil {
			slog.Error("Failed to send GET request for mathpix documetn status", "error", err)
			return err
		}

		// Parse JSON
		var pollResp PollResponse
		err = json.Unmarshal(bodyContents, &pollResp)
		if err != nil {
			slog.Error("Failed to unmarshal mathpix document status", "body", string(bodyContents), "error", err)
			return err
		}

		slog.Debug("Mathpix", "pollStatus", pollResp.Status)

		// If processing is done, return the markdown text
		switch pollResp.Status {
		case "completed":
			return nil
		case "error":
			return fmt.Errorf("mathpix PDF processing failed")
		}

		// Wait before polling again
		time.Sleep(MathpixPollInterval * time.Second)
	}
}

func (mp *MathpixDocumentProcessor) queryConversionResults(pdfID string) (string, error) {
	slog.Debug(">>MathpixDocumentProcessor.queryConversionResults")
	defer slog.Debug("<<MathpixDocumentProcessor.queryConversionResults")
	resultsURL := fmt.Sprintf("%s/%s.md", MathpixPdfApiURL, pdfID)

	req, err := mp.newRequest("GET", resultsURL, nil)
	if err != nil {
		slog.Error("Failed to crate GET request for mathpix document status", "error", err)
		return "", err
	}

	bodyContents, err := mp.doRequest(req)
	if err != nil {
		slog.Error("Failed to send GET request for mathpix documetn status", "error", err)
		return "", err
	}

	return string(bodyContents), nil
}

func (mp *MathpixDocumentProcessor) newRequest(method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("app_id", mp.mathpixAppID)
	req.Header.Set("app_key", mp.mathpixAppKey)

	return req, nil
}

func (mp *MathpixDocumentProcessor) doRequest(req *http.Request) ([]byte, error) {
	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("request failed with status_code=%d and status=%s", resp.StatusCode, resp.Status)
	}

	// Parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}
