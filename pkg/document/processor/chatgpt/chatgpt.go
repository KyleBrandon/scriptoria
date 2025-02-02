package chatgpt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/KyleBrandon/scriptoria/pkg/document"
	"github.com/sashabaranov/go-openai"
)

func New(store chatgptDocumentStore) *ChatgptDocumentProcessor {
	mp := &ChatgptDocumentProcessor{}

	mp.store = store

	return mp
}

// Initialize the Mathpix document processor
func (cp *ChatgptDocumentProcessor) Initialize(ctx context.Context, inputCh chan *document.DocumentTransform) (chan *document.DocumentTransform, error) {
	slog.Debug(">>ChatgptDocumentProcessor.Initialize")
	defer slog.Debug("<<ChatgptDocumentProcessor.Initialize")

	cp.ctx, cp.cancelFunc = context.WithCancel(ctx)
	cp.wg = &sync.WaitGroup{}
	cp.inputCh = inputCh
	cp.outputCh = make(chan *document.DocumentTransform)

	err := cp.readConfigurationSettings()
	if err != nil {
		return nil, err
	}

	cp.wg.Add(1)
	go cp.process()

	return cp.outputCh, nil
}

func (cp *ChatgptDocumentProcessor) CancelAndWait() {
	cp.cancelFunc()
	cp.wg.Wait()
}

func (cp *ChatgptDocumentProcessor) readConfigurationSettings() error {
	cp.chatgptAPIKey = os.Getenv("CHATGPT_API_KEY")
	if len(cp.chatgptAPIKey) == 0 {
		return errors.New("environment variable CHATGPT_API_KEY is not present")
	}

	return nil
}

func (cp *ChatgptDocumentProcessor) process() {
	slog.Debug(">>ChatgptDocumentProcessor.process")
	defer slog.Debug("<<ChatgptDocumentProcessor.process")

	defer cp.wg.Done()

	for {
		select {
		case <-cp.ctx.Done():
			slog.Debug("MathpixDocumentProcessor.process canceled")
			return

		case t := <-cp.inputCh:
			slog.Debug("MathMathpixDocumentProcessor.process received document to process")
			if t.Error != nil {
				continue
			}

			cp.wg.Add(1)
			go cp.processDocument(t)
		}
	}
}

func (cp *ChatgptDocumentProcessor) processDocument(t *document.DocumentTransform) {
	slog.Debug(">>ChatgptDocumentProcessor.processDocument")
	defer slog.Debug("<<ChatgptDocumentProcessor.processDocument")

	defer cp.wg.Done()

	// Initialize OpenAI client
	client := openai.NewClient(cp.chatgptAPIKey)

	content, err := io.ReadAll(t.Reader)
	if err != err {
		slog.Error("Failed to read the input document to clean up", "error", err)
		t.Reader.Close()
		t.Error = err
		cp.outputCh <- t
		return
	}

	t.Reader.Close()

	// Create a prompt for GPT to clean up the Markdown
	systemMessage := "You are an AI that processes Markdown text. Your task is to clean up the input by fixing Markdown syntax, correcting spelling and grammar, and ensuring proper formatting. Do NOT include any extra explanations, comments, or surrounding text—only return the valid Markdown output."
	prompt := fmt.Sprintf("Here is a Markdown file that was generated via OCR. Fix the Markdown formatting, correct any spelling and grammar errors, and ensure the syntax is valid. Do not add any explanations,comments, and do not surround the document text in a markdown code block. ONLY RETURN THE CLEANED MARKDOWN CONTENT AND NOTHING ELSE:\n\n%s", content)

	// Call the ChatGPT API
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{Role: "system", Content: systemMessage},
				{Role: "user", Content: prompt},
			},
			Temperature: 0.2, // Keep responses precise
		},
	)
	if err != nil {
		slog.Error("ChatGPT API error", "error", err)
		t.Error = err
		cp.outputCh <- t
		return
	}

	// Get the cleaned-up text
	cleanedMarkdown := resp.Choices[0].Message.Content

	// set the new reader
	t.Reader = io.NopCloser(strings.NewReader(cleanedMarkdown))
	cp.outputCh <- t
}
