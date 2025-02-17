package chatgpt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/KyleBrandon/scriptoria/internal/config"
	"github.com/KyleBrandon/scriptoria/pkg/document"
	"github.com/sashabaranov/go-openai"
)

// NewChatGPTProcessor will return processor that will send the document through ChatGPT with instructions to clean the formatting, spelling, and grammar.
func NewChatGPTProcessor() *ChatgptDocumentProcessor {
	cp := &ChatgptDocumentProcessor{}

	return cp
}

func (lp *ChatgptDocumentProcessor) GetName() string {
	return "ChatGPT Document Processor"
}

func (cp *ChatgptDocumentProcessor) Initialize(tempStoragePath string, bundles []config.StorageBundle) error {
	err := cp.readConfigurationSettings()
	if err != nil {
		slog.Error("Failed to read the configuration settings for ChatGPT", "error", err)
		return err
	}

	return nil
}

func (cp *ChatgptDocumentProcessor) readConfigurationSettings() error {
	cp.chatgptAPIKey = os.Getenv("CHATGPT_API_KEY")
	if len(cp.chatgptAPIKey) == 0 {
		return errors.New("environment variable CHATGPT_API_KEY is not present")
	}

	return nil
}

func (cp *ChatgptDocumentProcessor) Process(document *document.Document, reader io.ReadCloser) (io.ReadCloser, error) {
	slog.Debug(">>ChatgptDocumentProcessor.processDocument")
	defer slog.Debug("<<ChatgptDocumentProcessor.processDocument")

	// Initialize OpenAI client
	client := openai.NewClient(cp.chatgptAPIKey)

	content, err := io.ReadAll(reader)
	if err != err {
		slog.Error("Failed to read the input document to clean up", "error", err)
		return nil, err
	}

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
		return nil, err
	}

	// Get the cleaned-up text
	buffer := resp.Choices[0].Message.Content

	// For some reason ChatGPT will occasionally surround the entire processed output with
	// a Markdown code block. Check to see if the document is surrounded in a code block.
	// If so, remove it.
	cleanedMarkdown := strings.TrimPrefix(strings.TrimSuffix(string(buffer), "```"), "```markdown")
	// set the new reader
	r := io.NopCloser(strings.NewReader(cleanedMarkdown))

	return r, nil
}
