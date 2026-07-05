package handler

import (
	"net/http"

	"github.com/shadowy-pycoder/release-watcher/internal/server/response"
)

func RegisterHealth(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, map[string]string{"status": "ok"})
	})
}
