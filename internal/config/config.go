package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port          string
	DBPath        string
	GithubToken   string
	TelegramToken string
	PollInterval  time.Duration
	LogLevel      string
}

func Load() *Config {
	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		DBPath:        getEnv("DB_PATH", "./data/release-watcher.db"),
		GithubToken:   getEnv("GITHUB_TOKEN", ""),
		TelegramToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		PollInterval:  getDurationEnv("POLL_INTERVAL_SECONDS", 300),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
	}

	if cfg.GithubToken == "" {
		log.Println("WARN: GITHUB_TOKEN not set, API rate limits will be very low (60 req/hour)")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getDurationEnv(key string, fallbackSeconds int) time.Duration {
	if val := os.Getenv(key); val != "" {
		if sec, err := strconv.Atoi(val); err == nil {
			return time.Duration(sec) * time.Second
		}
	}
	return time.Duration(fallbackSeconds) * time.Second
}
