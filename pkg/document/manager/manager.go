package manager

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document"
)

func InitializeManager(ctx context.Context, queries *database.Queries, mux *http.ServeMux, source, destination document.DocumentStorage) (*DocumentManager, error) {
	slog.Debug(">>InitializeDocumentManager")
	defer slog.Debug("<<InitializeDocumentManager")

	// create sub-context and cancelFunc
	mgrCtx, mgrCanceFunc := context.WithCancel(ctx)

	var wg sync.WaitGroup
	dm := &DocumentManager{
		ctx:        mgrCtx,
		wg:         &wg,
		cancelFunc: mgrCanceFunc,
		store:      queries,
	}

	return dm, nil
}

func (dm *DocumentManager) CancelAndWait() {
	// cancel all go routines
	dm.cancelFunc()

	// wait until the document go routines are finished
	dm.wg.Wait()
}

func (dm *DocumentManager) StartMonitoring() {
	slog.Debug(">>StartMonitoring")
	defer slog.Debug("<<StartMonitoring")

	go dm.documentStorageMonitor()
}

func (dm *DocumentManager) documentStorageMonitor() {
	// loop waiting for new documents to read and write

	for {
		select {}
	}
}
