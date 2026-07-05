package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type envelope struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func OK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(envelope{Data: data}); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func Created(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(envelope{Data: data}); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func Error(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope{Error: msg}); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}

func ValidationError(w http.ResponseWriter, field, msg string) {
	Error(w, http.StatusBadRequest, field+": "+msg)
}
