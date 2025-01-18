package health

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/KyleBrandon/scriptoria/pkg/utils"
)

func NewHandler(mux *http.ServeMux, levelVar *slog.LevelVar, logger *slog.Logger) {
	h := &Handler{}
	h.logger = logger
	h.levelVar = levelVar
	h.RegisterRoutes(mux)
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/health", h.handlerHealthGet)
	mux.HandleFunc("GET /v1/logger", h.handlerLoggerGet)
	mux.HandleFunc("PUT /v1/logger", h.handlerLoggerUpdate)
}

func (h *Handler) handlerHealthGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>handlerGetHealth")
	defer slog.Debug("<<handlerGetHealth")

	response := struct {
		Status string `json:"status"`
	}{
		Status: "ok",
	}

	utils.RespondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) handlerLoggerGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>handlerLoggerGet")
	defer slog.Debug("<<handlerLoggerGet")

	h.mu.Lock()
	defer h.mu.Unlock()
	logLevel := h.levelVar.Level().String()
	slog.Info("Current log level", "level", logLevel)

	response := struct {
		LogLevel string `json:"log_level"`
	}{
		LogLevel: fmt.Sprintf("Current log level: %s", logLevel),
	}

	utils.RespondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) handlerLoggerUpdate(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>handlerLoggerUpdate")
	defer slog.Debug("<<handlerLoggerUpdate")

	h.mu.Lock()
	defer h.mu.Unlock()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	defer r.Body.Close()

	request := struct {
		LogLevel string `json:"log_level"`
	}{}

	if err := json.Unmarshal(body, &request); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	level, err := utils.ParseLogLevel(request.LogLevel)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid log level", err)
		return
	}

	h.levelVar.Set(level)

	utils.RespondWithNoContent(w, http.StatusOK)
}
