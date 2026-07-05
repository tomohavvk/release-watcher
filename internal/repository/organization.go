package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/shadowy-pycoder/release-watcher/internal/domain"
)

type OrganizationRepo struct {
	db *sql.DB
}

func (r *OrganizationRepo) FindOrCreate(ctx context.Context, name string) (*domain.Organization, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO organizations (name) VALUES (?) ON CONFLICT(name) DO NOTHING`, name)
	if err != nil {
		return nil, err
	}

	var org domain.Organization
	err = r.db.QueryRowContext(ctx,
		`SELECT id, name, last_polled_at, created_at FROM organizations WHERE name = ?`, name).
		Scan(&org.ID, &org.Name, &org.LastPolledAt, &org.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *OrganizationRepo) Follow(ctx context.Context, userID, orgID int64) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_organizations (user_id, org_id) VALUES (?, ?) ON CONFLICT DO NOTHING`,
		userID, orgID)
	return err
}

func (r *OrganizationRepo) Unfollow(ctx context.Context, userID, orgID int64) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM user_organizations WHERE user_id = ? AND org_id = ?`, userID, orgID)
	return err
}

func (r *OrganizationRepo) ListByUser(ctx context.Context, userID int64) ([]domain.Organization, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT o.id, o.name, o.last_polled_at, o.created_at
		FROM organizations o
		JOIN user_organizations uo ON uo.org_id = o.id
		WHERE uo.user_id = ?
		ORDER BY o.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []domain.Organization
	for rows.Next() {
		var o domain.Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.LastPolledAt, &o.CreatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

func (r *OrganizationRepo) ListAll(ctx context.Context) ([]domain.Organization, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT o.id, o.name, o.last_polled_at, o.created_at
		FROM organizations o
		JOIN user_organizations uo ON uo.org_id = o.id
		ORDER BY o.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []domain.Organization
	for rows.Next() {
		var o domain.Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.LastPolledAt, &o.CreatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

func (r *OrganizationRepo) UpdateLastPolled(ctx context.Context, orgID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE organizations SET last_polled_at = ? WHERE id = ?`, time.Now().UTC(), orgID)
	return err
}
