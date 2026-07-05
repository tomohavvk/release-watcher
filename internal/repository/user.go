package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/shadowy-pycoder/release-watcher/internal/domain"
)

type UserRepo struct {
	db *sql.DB
}

func (r *UserRepo) Create(ctx context.Context, name string) (*domain.User, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (name) VALUES (?) ON CONFLICT(name) DO NOTHING`, name)
	if err != nil {
		return nil, err
	}

	id, _ := res.LastInsertId()
	if id == 0 {
		return r.GetByName(ctx, name)
	}

	return r.GetByID(ctx, id)
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	var chatID sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, telegram_chat_id, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Name, &chatID, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	if chatID.Valid {
		u.TelegramChatID = &chatID.Int64
	}
	return &u, nil
}

func (r *UserRepo) GetByName(ctx context.Context, name string) (*domain.User, error) {
	var u domain.User
	var chatID sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, telegram_chat_id, created_at FROM users WHERE name = ?`, name).
		Scan(&u.ID, &u.Name, &chatID, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	if chatID.Valid {
		u.TelegramChatID = &chatID.Int64
	}
	return &u, nil
}

func (r *UserRepo) GetOrCreateByTelegram(ctx context.Context, telegramChatID int64, name string) (*domain.User, error) {
	var u domain.User
	var chatID sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, telegram_chat_id, created_at FROM users WHERE telegram_chat_id = ?`, telegramChatID).
		Scan(&u.ID, &u.Name, &chatID, &u.CreatedAt)
	if err == nil {
		if chatID.Valid {
			u.TelegramChatID = &chatID.Int64
		}
		return &u, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	userName := fmt.Sprintf("tg_%d", telegramChatID)
	if name != "" {
		userName = name
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO users (name, telegram_chat_id) VALUES (?, ?)
		 ON CONFLICT(name) DO UPDATE SET telegram_chat_id = excluded.telegram_chat_id`,
		userName, telegramChatID)
	if err != nil {
		return nil, err
	}

	return r.GetByTelegramChatID(ctx, telegramChatID)
}

func (r *UserRepo) GetByTelegramChatID(ctx context.Context, chatID int64) (*domain.User, error) {
	var u domain.User
	var cid sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, telegram_chat_id, created_at FROM users WHERE telegram_chat_id = ?`, chatID).
		Scan(&u.ID, &u.Name, &cid, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	if cid.Valid {
		u.TelegramChatID = &cid.Int64
	}
	return &u, nil
}

func (r *UserRepo) GetTelegramRecipientsForRepo(ctx context.Context, repoID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT u.telegram_chat_id
		FROM users u
		WHERE u.telegram_chat_id IS NOT NULL
		AND (
			EXISTS (
				SELECT 1 FROM user_organizations uo
				JOIN repositories r ON r.org_id = uo.org_id
				WHERE uo.user_id = u.id AND r.id = ?
			)
			OR EXISTS (
				SELECT 1 FROM user_repositories ur
				WHERE ur.user_id = u.id AND ur.repo_id = ?
			)
		)
		AND NOT EXISTS (
			SELECT 1 FROM muted_repos mr
			WHERE mr.user_id = u.id AND mr.repo_id = ?
		)`, repoID, repoID, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chatIDs []int64
	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			return nil, err
		}
		chatIDs = append(chatIDs, chatID)
	}
	return chatIDs, rows.Err()
}
