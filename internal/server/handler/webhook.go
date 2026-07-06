package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
)

func RegisterWebhook(mux *http.ServeMux, secret string) {
	mux.HandleFunc("POST /api/v1/webhook/github", handleGitHubWebhook(secret))
}

func handleGitHubWebhook(secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if secret != "" {
			sig := r.Header.Get("X-Hub-Signature-256")
			if !verifySignature(body, sig, secret) {
				slog.Warn("webhook: invalid signature")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}

		event := r.Header.Get("X-GitHub-Event")
		if event != "push" {
			w.WriteHeader(http.StatusOK)
			return
		}

		slog.Info("webhook: deploying")
		cmd := exec.Command("systemd-run", "--no-block", "/opt/release-watcher/deploy.sh")
		if err := cmd.Start(); err != nil {
			slog.Error("webhook: failed to start deploy", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func verifySignature(payload []byte, signature, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	sig, err := hex.DecodeString(signature[7:])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hmac.Equal(sig, mac.Sum(nil))
}
