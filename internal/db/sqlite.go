package db

import (
	"database/sql"
	"embed"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed migration/*.sql
var migrations embed.FS

func Init(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	slog.Info("database connected", "path", dbPath)
	return db, nil
}

func Migrate(db *sql.DB) error {
	entries, err := migrations.ReadDir("migration")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		data, err := migrations.ReadFile("migration/" + entry.Name())
		if err != nil {
			return err
		}

		if _, err := db.Exec(string(data)); err != nil {
			return err
		}

		slog.Info("migration applied", "file", entry.Name())
	}

	return nil
}
