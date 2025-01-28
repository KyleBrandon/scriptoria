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
	"path/filepath"
	"time"

	"github.com/KyleBrandon/scriptoria/pkg/document"
)

func New() *MathpixDocumentProcessor {
	mp := &MathpixDocumentProcessor{}

	mp.readConfigurationSettings()

	return mp
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

	mp.markdownFileLocation = os.Getenv("MARKDOWN_FILE_LOCATION")
	if len(mp.markdownFileLocation) == 0 {
		return errors.New("environment variable MARKDOWN_FILE_LOCATION is not present")
	}

	return nil
}

func (mp *MathpixDocumentProcessor) ProcessDocument(doc document.Document, documentStorage document.DocumentStorage) {
	slog.Debug(">>ProcessDocument")
	defer slog.Debug("<<ProcessDocument")

	// Upload PDF to Mathpix
	pdfID, err := mp.sendDocumentToMathpix(doc, documentStorage)
	if err != nil {
		slog.Error("Error uploading PDF", "error", err)
		return
	}

	slog.Debug("Upload successful", "jobId", pdfID)

	// Poll for results
	err = mp.pollForResults(pdfID)
	if err != nil {
		slog.Error("Error getting results", "error", err)
		return
	}

	markdownText, err := mp.queryConversionResults(pdfID)
	if err != nil {
		slog.Error("Failed to query conversion results", "error", err)
		return
	}

	// Save to Markdown file
	err = mp.saveToMarkdown(doc.Name, markdownText)
	if err != nil {
		slog.Error("Error saving markdown file", "error", err)
		return
	}
}

// UploadPDF uploads a PDF file to Mathpix and returns the Job ID
func (mp *MathpixDocumentProcessor) sendDocumentToMathpix(doc document.Document, documentStorage document.DocumentStorage) (string, error) {
	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", doc.Name)
	if err != nil {
		slog.Error("Failed to create form file", "error", err)
		return "", err
	}

	reader, err := documentStorage.GetFileReader(doc)
	if err != nil {
		slog.Error("Failed to get the document reader", "error", err)
		return "", err
	}

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

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())

	respBody, err := mp.doRequest(req)
	if err != nil {
		slog.Error("Failed to send mathpix request", "error", err)
		return "", err
	}

	var uploadResp UploadResponse
	err = json.Unmarshal(respBody, &uploadResp)
	if err != nil {
		slog.Error("Failed to unmarshal mathpix response", "error", err)
		return "", err
	}

	return uploadResp.PdfID, nil
}

// PollForResults polls Mathpix API for PDF processing status
func (mp *MathpixDocumentProcessor) pollForResults(pdfID string) error {
	slog.Debug(">>PollForResults")
	defer slog.Debug("<<PollForResults")

	pollURL := fmt.Sprintf("%s/%s", MathpixPdfApiURL, pdfID)

	for {
		req, err := mp.newRequest("GET", pollURL, nil)
		if err != nil {
			slog.Error("Failed to crate GET request for mathpix document status", "error", err)
			return err
		}

		respBody, err := mp.doRequest(req)
		if err != nil {
			slog.Error("Failed to send GET request for mathpix documetn status", "error", err)
			return err
		}

		// Parse JSON
		var pollResp PollResponse
		err = json.Unmarshal(respBody, &pollResp)
		if err != nil {
			slog.Error("Failed to unmarshal mathpix document status", "body", string(respBody), "error", err)
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
		slog.Debug("Waiting for before polling again...")
		time.Sleep(MathpixPollInterval * time.Second)
	}
}

func (mp *MathpixDocumentProcessor) queryConversionResults(pdfID string) (string, error) {
	resultsURL := fmt.Sprintf("%s/%s.mmd", MathpixPdfApiURL, pdfID)

	req, err := mp.newRequest("GET", resultsURL, nil)
	if err != nil {
		slog.Error("Failed to crate GET request for mathpix document status", "error", err)
		return "", err
	}

	respBody, err := mp.doRequest(req)
	if err != nil {
		slog.Error("Failed to send GET request for mathpix documetn status", "error", err)
		return "", err
	}

	return string(respBody), nil
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

	// Parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

// SaveToMarkdown saves the extracted Markdown to a file
func (mp *MathpixDocumentProcessor) saveToMarkdown(name, content string) error {
	markdownFilePath := fmt.Sprintf("%s.md", filepath.Join(mp.markdownFileLocation, name))

	file, err := os.Create(markdownFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}
