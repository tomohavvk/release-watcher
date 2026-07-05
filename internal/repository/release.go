package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/shadowy-pycoder/release-watcher/internal/domain"
)

type ReleaseRepo struct {
	db *sql.DB
}

func (r *ReleaseRepo) Upsert(ctx context.Context, rel *domain.Release) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO releases (repo_id, tag_name, name, body, html_url, type, author, author_avatar, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_id, tag_name) DO UPDATE SET
			name = excluded.name,
			body = excluded.body,
			html_url = excluded.html_url,
			type = excluded.type,
			author = excluded.author,
			author_avatar = excluded.author_avatar,
			published_at = excluded.published_at`,
		rel.RepoID, rel.TagName, rel.Name, rel.Body, rel.HTMLURL,
		rel.Type, rel.Author, rel.AuthorAvatar, rel.PublishedAt)
	return err
}

func (r *ReleaseRepo) InsertIfNotExists(ctx context.Context, rel *domain.Release) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO releases (repo_id, tag_name, name, body, html_url, type, author, author_avatar, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_id, tag_name) DO NOTHING`,
		rel.RepoID, rel.TagName, rel.Name, rel.Body, rel.HTMLURL,
		rel.Type, rel.Author, rel.AuthorAvatar, rel.PublishedAt)
	if err != nil {
		return false, err
	}
	rows, _ := res.RowsAffected()
	return rows > 0, nil
}

func (r *ReleaseRepo) UpdatePublishedAt(ctx context.Context, repoID int64, tagName string, publishedAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE releases SET published_at = ? WHERE repo_id = ? AND tag_name = ?`,
		publishedAt, repoID, tagName)
	return err
}


func (r *ReleaseRepo) Feed(ctx context.Context, userID int64, limit, offset int) ([]domain.Release, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT rel.id, rel.repo_id, rel.tag_name, rel.name, rel.body, rel.html_url,
			   rel.type, rel.author, rel.author_avatar, rel.published_at, rel.created_at,
			   repo.full_name, o.name
		FROM releases rel
		JOIN repositories repo ON repo.id = rel.repo_id
		JOIN organizations o ON o.id = repo.org_id
		JOIN user_organizations uo ON uo.org_id = o.id AND uo.user_id = ?
		LEFT JOIN muted_repos mr ON mr.repo_id = rel.repo_id AND mr.user_id = ?
		WHERE mr.repo_id IS NULL
		ORDER BY rel.published_at DESC
		LIMIT ? OFFSET ?`, userID, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var releases []domain.Release
	for rows.Next() {
		var rel domain.Release
		if err := rows.Scan(
			&rel.ID, &rel.RepoID, &rel.TagName, &rel.Name, &rel.Body, &rel.HTMLURL,
			&rel.Type, &rel.Author, &rel.AuthorAvatar, &rel.PublishedAt, &rel.CreatedAt,
			&rel.RepoName, &rel.OrgName,
		); err != nil {
			return nil, err
		}
		releases = append(releases, rel)
	}
	return releases, rows.Err()
}
