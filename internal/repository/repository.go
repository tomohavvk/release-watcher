package repository

import (
	"context"
	"database/sql"

	"github.com/shadowy-pycoder/release-watcher/internal/domain"
)

type RepositoryRepo struct {
	db *sql.DB
}

func (r *RepositoryRepo) Upsert(ctx context.Context, orgID int64, name, fullName string) (*domain.Repository, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO repositories (org_id, name, full_name) VALUES (?, ?, ?)
		 ON CONFLICT(full_name) DO NOTHING`, orgID, name, fullName)
	if err != nil {
		return nil, err
	}

	var repo domain.Repository
	err = r.db.QueryRowContext(ctx,
		`SELECT id, org_id, name, full_name, created_at FROM repositories WHERE full_name = ?`, fullName).
		Scan(&repo.ID, &repo.OrgID, &repo.Name, &repo.FullName, &repo.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (r *RepositoryRepo) ListByOrg(ctx context.Context, orgID int64) ([]domain.Repository, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, org_id, name, full_name, created_at
		FROM repositories WHERE org_id = ?
		ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []domain.Repository
	for rows.Next() {
		var repo domain.Repository
		if err := rows.Scan(&repo.ID, &repo.OrgID, &repo.Name, &repo.FullName, &repo.CreatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}
	return repos, rows.Err()
}

func (r *RepositoryRepo) ListByOrgWithMuteStatus(ctx context.Context, orgID, userID int64) ([]domain.Repository, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.org_id, r.name, r.full_name, r.created_at,
			   CASE WHEN mr.repo_id IS NOT NULL THEN 1 ELSE 0 END as muted
		FROM repositories r
		LEFT JOIN muted_repos mr ON mr.repo_id = r.id AND mr.user_id = ?
		WHERE r.org_id = ?
		ORDER BY r.name`, userID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []domain.Repository
	for rows.Next() {
		var repo domain.Repository
		if err := rows.Scan(&repo.ID, &repo.OrgID, &repo.Name, &repo.FullName, &repo.CreatedAt, &repo.Muted); err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}
	return repos, rows.Err()
}

func (r *RepositoryRepo) GetByID(ctx context.Context, id int64) (*domain.Repository, error) {
	var repo domain.Repository
	err := r.db.QueryRowContext(ctx,
		`SELECT id, org_id, name, full_name, created_at FROM repositories WHERE id = ?`, id).
		Scan(&repo.ID, &repo.OrgID, &repo.Name, &repo.FullName, &repo.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (r *RepositoryRepo) GetByFullName(ctx context.Context, fullName string) (*domain.Repository, error) {
	var repo domain.Repository
	err := r.db.QueryRowContext(ctx,
		`SELECT id, org_id, name, full_name, created_at FROM repositories WHERE full_name = ?`, fullName).
		Scan(&repo.ID, &repo.OrgID, &repo.Name, &repo.FullName, &repo.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (r *RepositoryRepo) FollowRepo(ctx context.Context, userID, repoID int64) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_repositories (user_id, repo_id) VALUES (?, ?) ON CONFLICT DO NOTHING`,
		userID, repoID)
	return err
}

func (r *RepositoryRepo) UnfollowRepo(ctx context.Context, userID, repoID int64) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM user_repositories WHERE user_id = ? AND repo_id = ?`, userID, repoID)
	return err
}

func (r *RepositoryRepo) ListFollowedByUser(ctx context.Context, userID int64) ([]domain.Repository, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.org_id, r.name, r.full_name, r.created_at
		FROM repositories r
		JOIN user_repositories ur ON ur.repo_id = r.id
		WHERE ur.user_id = ?
		ORDER BY r.full_name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []domain.Repository
	for rows.Next() {
		var repo domain.Repository
		if err := rows.Scan(&repo.ID, &repo.OrgID, &repo.Name, &repo.FullName, &repo.CreatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}
	return repos, rows.Err()
}
