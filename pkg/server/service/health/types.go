package health

import (
	"log/slog"
	"sync"
)

type Handler struct {
	logger   *slog.Logger
	levelVar *slog.LevelVar
	mu       sync.RWMutex
}
