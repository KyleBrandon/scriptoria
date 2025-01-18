package utils

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

func RespondWithJSON(writer http.ResponseWriter, code int, payload interface{}) {
	resultData, err := json.Marshal(payload)
	if err != nil {
		RespondWithError(writer, http.StatusBadRequest, "Error marshalling result", err)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	writer.Write(resultData)
}

func RespondWithError(writer http.ResponseWriter, code int, message string, err error) {
	slog.Error(message, "http_status", code, "error", err)

	response := struct {
		Error string `json:"error"`
	}{
		Error: message,
	}

	RespondWithJSON(writer, code, response)
}

func RespondWithString(writer http.ResponseWriter, contentType string, code int, msg string) {
	writer.Header().Set("Content-Type", contentType)
	writer.WriteHeader(code)
	io.WriteString(writer, msg)
}

func RespondWithNoContent(writer http.ResponseWriter, code int) {
	writer.WriteHeader(code)
}
