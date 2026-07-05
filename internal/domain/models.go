package domain

import "time"

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Organization struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	LastPolledAt *time.Time `json:"last_polled_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Repository struct {
	ID        int64     `json:"id"`
	OrgID     int64     `json:"org_id"`
	Name      string    `json:"name"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
	Muted     bool      `json:"muted,omitempty"`
}

type Release struct {
	ID           int64     `json:"id"`
	RepoID       int64     `json:"repo_id"`
	TagName      string    `json:"tag_name"`
	Name         string    `json:"name"`
	Body         string    `json:"body"`
	HTMLURL      string    `json:"html_url"`
	Type         string    `json:"type"`
	Author       string    `json:"author"`
	AuthorAvatar string    `json:"author_avatar"`
	PublishedAt  time.Time `json:"published_at"`
	CreatedAt    time.Time `json:"created_at"`
	RepoName     string    `json:"repo_name,omitempty"`
	OrgName      string    `json:"org_name,omitempty"`
}

type MutedRepo struct {
	UserID    int64     `json:"user_id"`
	RepoID    int64     `json:"repo_id"`
	RepoName  string    `json:"repo_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
