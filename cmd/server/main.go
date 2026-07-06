package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shadowy-pycoder/release-watcher/internal/config"
	"github.com/shadowy-pycoder/release-watcher/internal/db"
	"github.com/shadowy-pycoder/release-watcher/internal/github"
	"github.com/shadowy-pycoder/release-watcher/internal/poller"
	"github.com/shadowy-pycoder/release-watcher/internal/repository"
	"github.com/shadowy-pycoder/release-watcher/internal/server/handler"
	"github.com/shadowy-pycoder/release-watcher/internal/server/middleware"
	"github.com/shadowy-pycoder/release-watcher/internal/telegram"
)

func main() {
	cfg := config.Load()

	setupLogger(cfg.LogLevel)

	database, err := db.Init(cfg.DBPath)
	if err != nil {
		slog.Error("failed to init database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	repos := repository.NewAll(database)
	ghClient := github.NewClient(cfg.GithubToken)
	p := poller.New(repos, ghClient, cfg.PollInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var bot *telegram.Bot
	if cfg.TelegramToken != "" {
		bot = telegram.New(cfg.TelegramToken, repos, ghClient, p)
		p.OnNewRelease(bot.NotifyNewRelease)
		go bot.Start(ctx)
	}

	go p.Start(ctx)

	mux := http.NewServeMux()

	handler.RegisterHealth(mux)
	handler.RegisterUser(mux, repos, bot)
	handler.RegisterOrganization(mux, repos, p)
	handler.RegisterFeed(mux, repos)
	handler.RegisterMute(mux, repos)
	handler.RegisterRepos(mux, repos)
	handler.RegisterWebhook(mux, cfg.WebhookSecret)
	registerStaticFiles(mux)

	var h http.Handler = mux
	h = middleware.Logging(h)
	h = middleware.Recovery(h)
	h = corsMiddleware(h)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server started", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	slog.Info("server stopped")
}

func registerStaticFiles(mux *http.ServeMux) {
	distDir := "./webapp/dist"
	fs := http.FileServer(http.Dir(distDir))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) > 1 {
			path := distDir + r.URL.Path
			if _, err := os.Stat(path); os.IsNotExist(err) {
				http.ServeFile(w, r, distDir+"/index.html")
				return
			}
		}
		fs.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func setupLogger(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})))
}
