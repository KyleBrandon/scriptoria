package obsidian

import (
	"context"
)

type (
	obsidianDocumentStore interface{}

	ObsidianDocumentPostProcessor struct {
		ctx             context.Context
		store           obsidianDocumentStore
		tempStoragePath string
	}
)
