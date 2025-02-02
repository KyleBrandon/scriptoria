package chatgpt

import (
	"context"
	"sync"

	"github.com/KyleBrandon/scriptoria/pkg/document"
)

type (

	// Identify the database methos for the Mathpix processor
	chatgptDocumentStore interface{}

	ChatgptDocumentProcessor struct {
		chatgptAPIKey string

		ctx        context.Context
		cancelFunc context.CancelFunc
		wg         *sync.WaitGroup
		store      chatgptDocumentStore
		inputCh    chan *document.DocumentTransform
		outputCh   chan *document.DocumentTransform
	}
)
