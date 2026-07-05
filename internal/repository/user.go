package repository

import (
	"context"
	"database/sql"

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
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Name, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) GetByName(ctx context.Context, name string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, created_at FROM users WHERE name = ?`, name).
		Scan(&u.ID, &u.Name, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
