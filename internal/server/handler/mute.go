package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/shadowy-pycoder/release-watcher/internal/domain"
	"github.com/shadowy-pycoder/release-watcher/internal/repository"
	"github.com/shadowy-pycoder/release-watcher/internal/server/response"
)

func RegisterMute(mux *http.ServeMux, repos *repository.All) {
	mux.HandleFunc("POST /api/v1/mutes", handleMute(repos))
	mux.HandleFunc("DELETE /api/v1/mutes/{repoId}", handleUnmute(repos))
	mux.HandleFunc("GET /api/v1/mutes", handleListMutes(repos))
}

func handleMute(repos *repository.All) http.HandlerFunc {
	type request struct {
		UserID int64 `json:"user_id"`
		RepoID int64 `json:"repo_id"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := repos.Mute.Mute(r.Context(), req.UserID, req.RepoID); err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to mute repo")
			return
		}

		response.Created(w, map[string]string{"status": "muted"})
	}
}

func handleUnmute(repos *repository.All) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID, err := strconv.ParseInt(r.PathValue("repoId"), 10, 64)
		if err != nil {
			response.ValidationError(w, "repoId", "invalid")
			return
		}

		userID, err := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
		if err != nil {
			response.ValidationError(w, "user_id", "required")
			return
		}

		if err := repos.Mute.Unmute(r.Context(), userID, repoID); err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to unmute repo")
			return
		}

		response.OK(w, map[string]string{"status": "unmuted"})
	}
}

func handleListMutes(repos *repository.All) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
		if err != nil {
			response.ValidationError(w, "user_id", "required")
			return
		}

		mutes, err := repos.Mute.ListByUser(r.Context(), userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to list mutes")
			return
		}

		if mutes == nil {
			mutes = []domain.MutedRepo{}
		}

		response.OK(w, mutes)
	}
}
