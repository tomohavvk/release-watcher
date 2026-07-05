package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/shadowy-pycoder/release-watcher/internal/domain"
	"github.com/shadowy-pycoder/release-watcher/internal/poller"
	"github.com/shadowy-pycoder/release-watcher/internal/repository"
	"github.com/shadowy-pycoder/release-watcher/internal/server/response"
)

func RegisterOrganization(mux *http.ServeMux, repos *repository.All, p *poller.Poller) {
	mux.HandleFunc("POST /api/v1/following", handleFollow(repos, p))
	mux.HandleFunc("DELETE /api/v1/following/{orgId}", handleUnfollow(repos))
	mux.HandleFunc("GET /api/v1/following", handleListFollowing(repos))
}

func handleFollow(repos *repository.All, p *poller.Poller) http.HandlerFunc {
	type request struct {
		UserID  int64  `json:"user_id"`
		OrgName string `json:"org_name"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.OrgName == "" {
			response.ValidationError(w, "org_name", "required")
			return
		}

		org, err := repos.Organization.FindOrCreate(r.Context(), req.OrgName)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to create organization")
			return
		}

		if err := repos.Organization.Follow(r.Context(), req.UserID, org.ID); err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to follow organization")
			return
		}

		p.Trigger()

		response.Created(w, org)
	}
}

func handleUnfollow(repos *repository.All) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := strconv.ParseInt(r.PathValue("orgId"), 10, 64)
		if err != nil {
			response.ValidationError(w, "orgId", "invalid")
			return
		}

		userID, err := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
		if err != nil {
			response.ValidationError(w, "user_id", "required")
			return
		}

		if err := repos.Organization.Unfollow(r.Context(), userID, orgID); err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to unfollow organization")
			return
		}

		response.OK(w, map[string]string{"status": "unfollowed"})
	}
}

func handleListFollowing(repos *repository.All) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
		if err != nil {
			response.ValidationError(w, "user_id", "required")
			return
		}

		orgs, err := repos.Organization.ListByUser(r.Context(), userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to list organizations")
			return
		}

		if orgs == nil {
			orgs = []domain.Organization{}
		}

		response.OK(w, orgs)
	}
}
