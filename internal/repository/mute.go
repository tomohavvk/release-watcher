package repository

import (
	"context"
	"database/sql"

	"github.com/shadowy-pycoder/release-watcher/internal/domain"
)

type MuteRepo struct {
	db *sql.DB
}

func (r *MuteRepo) Mute(ctx context.Context, userID, repoID int64) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO muted_repos (user_id, repo_id) VALUES (?, ?) ON CONFLICT DO NOTHING`,
		userID, repoID)
	return err
}

func (r *MuteRepo) Unmute(ctx context.Context, userID, repoID int64) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM muted_repos WHERE user_id = ? AND repo_id = ?`, userID, repoID)
	return err
}

func (r *MuteRepo) ListByUser(ctx context.Context, userID int64) ([]domain.MutedRepo, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT mr.user_id, mr.repo_id, r.full_name, mr.created_at
		FROM muted_repos mr
		JOIN repositories r ON r.id = mr.repo_id
		WHERE mr.user_id = ?
		ORDER BY r.full_name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mutes []domain.MutedRepo
	for rows.Next() {
		var m domain.MutedRepo
		if err := rows.Scan(&m.UserID, &m.RepoID, &m.RepoName, &m.CreatedAt); err != nil {
			return nil, err
		}
		mutes = append(mutes, m)
	}
	return mutes, rows.Err()
}
