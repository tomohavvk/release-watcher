package handler

import (
	"net/http"
	"strconv"

	"github.com/shadowy-pycoder/release-watcher/internal/domain"
	"github.com/shadowy-pycoder/release-watcher/internal/repository"
	"github.com/shadowy-pycoder/release-watcher/internal/server/response"
)

func RegisterRepos(mux *http.ServeMux, repos *repository.All) {
	mux.HandleFunc("GET /api/v1/repos", handleListRepos(repos))
}

func handleListRepos(repos *repository.All) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := strconv.ParseInt(r.URL.Query().Get("org_id"), 10, 64)
		if err != nil {
			response.ValidationError(w, "org_id", "required")
			return
		}

		userID, err := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
		if err != nil {
			response.ValidationError(w, "user_id", "required")
			return
		}

		repoList, err := repos.Repository.ListByOrgWithMuteStatus(r.Context(), orgID, userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to list repos")
			return
		}

		if repoList == nil {
			repoList = []domain.Repository{}
		}

		response.OK(w, repoList)
	}
}
