package obsidian

import (
	"context"
)

type (
	obsidianDocumentStore interface{}

	ObsidianDocumentPostProcessor struct {
		ctx               context.Context
		store             obsidianDocumentStore
		localStoragePath  string
		attachmentsFolder string
		notesFolder       string
	}
)
