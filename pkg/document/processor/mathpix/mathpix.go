package mathpix

import (
	"bytes"
	"context"
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

	"github.com/KyleBrandon/scriptoria/pkg/document"
)

func New(store mathpixDocumentStore) *MathpixDocumentProcessor {
	mp := &MathpixDocumentProcessor{}

	mp.store = store

	return mp
}

// Initialize the Mathpix document processor
func (mp *MathpixDocumentProcessor) Initialize(ctx context.Context, inputCh chan *document.DocumentTransform) (chan *document.DocumentTransform, error) {
	slog.Debug(">>MathpixDocumentProcessor.Initialize")
	defer slog.Debug("<<MathpixDocumentProcessor.Initialize")

	mp.ctx = ctx
	mp.inputCh = inputCh
	mp.outputCh = make(chan *document.DocumentTransform)

	err := mp.readConfigurationSettings()
	if err != nil {
		return nil, err
	}

	go mp.process()

	return mp.outputCh, nil
}

func (mp *MathpixDocumentProcessor) process() {
	slog.Debug(">>MathpixDocumentProcessor.process")
	defer slog.Debug("<<MathpixDocumentProcessor.process")

	for {
		select {
		case <-mp.ctx.Done():
			slog.Debug("MathpixDocumentProcessor.process canceled")
			return

		case t := <-mp.inputCh:
			slog.Debug("MathMathpixDocumentProcessor.process received document to process")
			go mp.processDocument(t)
		}
	}
}

func (mp *MathpixDocumentProcessor) processDocument(t *document.DocumentTransform) {
	slog.Debug(">>MathpixDocumentProcessor.processDocument")
	defer slog.Debug("<<MathpixDocumentProcessor.processDocument")

	output := document.DocumentTransform{}

	// Upload PDF to Mathpix
	pdfID, err := mp.sendDocumentToMathpix(t.Doc.Name, t.Reader)
	if err != nil {
		slog.Error("Error uploading PDF", "error", err)
		output.Error = err
		mp.outputCh <- &output
		return

	}

	// Poll for results
	err = mp.pollForResults(pdfID)
	if err != nil {
		slog.Error("Error getting results", "error", err)
		output.Error = err
		mp.outputCh <- &output
		return
	}

	markdownText, err := mp.queryConversionResults(pdfID)
	if err != nil {
		slog.Error("Failed to query conversion results", "error", err)
		output.Error = err
		mp.outputCh <- &output
		return
	}

	name := t.Doc.GetDocumentName() + ".mmd"

	// create an output document that represents the multi-markdown file
	output.Doc = &document.Document{
		Name:         name,
		MimeType:     "text/markdown",
		CreatedTime:  time.Now(),
		ModifiedTime: time.Now(),
	}

	output.Reader = io.NopCloser(strings.NewReader(markdownText))
	mp.outputCh <- &output
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
