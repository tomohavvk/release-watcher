package handler

import (
	"net/http"
	"strconv"

	"github.com/shadowy-pycoder/release-watcher/internal/domain"
	"github.com/shadowy-pycoder/release-watcher/internal/repository"
	"github.com/shadowy-pycoder/release-watcher/internal/server/response"
)

func RegisterFeed(mux *http.ServeMux, repos *repository.All) {
	mux.HandleFunc("GET /api/v1/feed", handleFeed(repos))
}

func handleFeed(repos *repository.All) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
		if err != nil {
			response.ValidationError(w, "user_id", "required")
			return
		}

		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
				limit = parsed
			}
		}

		offset := 0
		if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		releases, err := repos.Release.Feed(r.Context(), userID, limit, offset)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to get feed")
			return
		}

		if releases == nil {
			releases = []domain.Release{}
		}

		response.OK(w, releases)
	}
}
