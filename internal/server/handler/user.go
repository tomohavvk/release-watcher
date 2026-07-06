package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/shadowy-pycoder/release-watcher/internal/repository"
	"github.com/shadowy-pycoder/release-watcher/internal/server/response"
)

type CodeSender interface {
	SendCode(ctx context.Context, chatID int64, code string) error
}

type authCode struct {
	code      string
	chatID    int64
	expiresAt time.Time
}

var (
	codes   = map[string]*authCode{}
	codesMu sync.Mutex
)

func RegisterUser(mux *http.ServeMux, repos *repository.All, sender CodeSender) {
	mux.HandleFunc("POST /api/v1/auth/request", handleAuthRequest(repos, sender))
	mux.HandleFunc("POST /api/v1/auth/verify", handleAuthVerify(repos))
	mux.HandleFunc("GET /api/v1/users/{name}", handleGetUser(repos))
}

func handleAuthRequest(repos *repository.All, sender CodeSender) http.HandlerFunc {
	type request struct {
		Username string `json:"username"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		username := strings.TrimPrefix(strings.TrimSpace(req.Username), "@")
		if username == "" {
			response.ValidationError(w, "username", "required")
			return
		}

		user, err := repos.User.GetByName(r.Context(), username)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				response.Error(w, http.StatusNotFound, "Спершу напиши /start боту @ReleaseWatcherBot")
				return
			}
			response.Error(w, http.StatusInternalServerError, "failed to find user")
			return
		}

		if user.TelegramChatID == nil {
			response.Error(w, http.StatusNotFound, "Спершу напиши /start боту @ReleaseWatcherBot")
			return
		}

		code := generateCode()

		codesMu.Lock()
		codes[username] = &authCode{
			code:      code,
			chatID:    *user.TelegramChatID,
			expiresAt: time.Now().Add(5 * time.Minute),
		}
		codesMu.Unlock()

		if err := sender.SendCode(r.Context(), *user.TelegramChatID, code); err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to send code")
			return
		}

		response.OK(w, map[string]string{"status": "code_sent"})
	}
}

func handleAuthVerify(repos *repository.All) http.HandlerFunc {
	type request struct {
		Username string `json:"username"`
		Code     string `json:"code"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		username := strings.TrimPrefix(strings.TrimSpace(req.Username), "@")
		code := strings.TrimSpace(req.Code)

		codesMu.Lock()
		stored, ok := codes[username]
		if ok {
			delete(codes, username)
		}
		codesMu.Unlock()

		if !ok || time.Now().After(stored.expiresAt) || stored.code != code {
			response.Error(w, http.StatusUnauthorized, "Невірний або прострочений код")
			return
		}

		user, err := repos.User.GetByTelegramChatID(r.Context(), stored.chatID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to get user")
			return
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

func generateCode() string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("%06d", int(b[0])<<16|int(b[1])<<8|int(b[2])%1000000)
}
