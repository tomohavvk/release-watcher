package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/shadowy-pycoder/release-watcher/internal/poller"
	"github.com/shadowy-pycoder/release-watcher/internal/repository"
	"github.com/shadowy-pycoder/release-watcher/internal/server/response"
)

var defaultOrgs = []string{"singnet", "trueagi-io", "asi-alliance", "tomohavvk"}

func RegisterUser(mux *http.ServeMux, repos *repository.All, p *poller.Poller) {
	mux.HandleFunc("POST /api/v1/auth", handleAuth(repos, p))
	mux.HandleFunc("GET /api/v1/users/{name}", handleGetUser(repos))
}

func handleAuth(repos *repository.All, p *poller.Poller) http.HandlerFunc {
	type request struct {
		Name string `json:"name"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Name == "" {
			response.ValidationError(w, "name", "required")
			return
		}

		user, err := repos.User.Create(r.Context(), req.Name)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to create user")
			return
		}

		following, _ := repos.Organization.ListByUser(r.Context(), user.ID)
		if len(following) == 0 {
			for _, orgName := range defaultOrgs {
				org, err := repos.Organization.FindOrCreate(r.Context(), orgName)
				if err != nil {
					slog.Error("failed to create default org", "org", orgName, "error", err)
					continue
				}
				if err := repos.Organization.Follow(r.Context(), user.ID, org.ID); err != nil {
					slog.Error("failed to follow default org", "org", orgName, "error", err)
				}
			}
			p.Trigger()
		}

		response.OK(w, user)
	}
}

func handleGetUser(repos *repository.All) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		user, err := repos.User.GetByName(r.Context(), name)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				response.Error(w, http.StatusNotFound, "user not found")
				return
			}
			response.Error(w, http.StatusInternalServerError, "failed to get user")
			return
		}
		response.OK(w, user)
	}
}
