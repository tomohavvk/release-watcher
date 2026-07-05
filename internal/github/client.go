package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Client struct {
	http  *http.Client
	token string
}

func NewClient(token string) *Client {
	return &Client{
		http:  &http.Client{Timeout: 30 * time.Second},
		token: token,
	}
}

type GHRelease struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	HTMLURL     string  `json:"html_url"`
	PublishedAt string  `json:"published_at"`
	Author      *GHUser `json:"author"`
}

type GHTag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
}

type GHCommit struct {
	Commit struct {
		Committer struct {
			Date string `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}

type GHRepo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

type GHUser struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

func (c *Client) get(ctx context.Context, url string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	slog.Debug("github request", "url", url)

	resp, err := c.http.Do(req)
	if err != nil {
		slog.Error("github request failed", "url", url, "error", err)
		return err
	}
	defer resp.Body.Close()

	slog.Info("github response", "url", url, "status", resp.StatusCode)

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found: %s", url)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github api error %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) ListRepos(ctx context.Context, owner string) ([]GHRepo, error) {
	var allRepos []GHRepo

	for page := 1; page <= 10; page++ {
		var repos []GHRepo
		url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&page=%d", owner, page)
		err := c.get(ctx, url, &repos)
		if err != nil {
			slog.Debug("org repos failed, trying user repos", "owner", owner, "error", err)
			return c.listUserRepos(ctx, owner)
		}
		allRepos = append(allRepos, repos...)
		if len(repos) < 100 {
			break
		}
	}

	return allRepos, nil
}

func (c *Client) listUserRepos(ctx context.Context, owner string) ([]GHRepo, error) {
	var allRepos []GHRepo

	for page := 1; page <= 10; page++ {
		var repos []GHRepo
		url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100&page=%d", owner, page)
		if err := c.get(ctx, url, &repos); err != nil {
			return nil, fmt.Errorf("list repos for %s: %w", owner, err)
		}
		allRepos = append(allRepos, repos...)
		if len(repos) < 100 {
			break
		}
	}

	return allRepos, nil
}

func (c *Client) ListReleases(ctx context.Context, fullName string) ([]GHRelease, error) {
	var releases []GHRelease
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=1", fullName)
	if err := c.get(ctx, url, &releases); err != nil {
		return nil, err
	}
	return releases, nil
}

func (c *Client) ListTags(ctx context.Context, fullName string) ([]GHTag, error) {
	var tags []GHTag
	url := fmt.Sprintf("https://api.github.com/repos/%s/tags?per_page=1", fullName)
	if err := c.get(ctx, url, &tags); err != nil {
		return nil, err
	}
	return tags, nil
}

func (c *Client) GetCommitDate(ctx context.Context, commitURL string) (string, error) {
	var commit GHCommit
	if err := c.get(ctx, commitURL, &commit); err != nil {
		return "", err
	}
	return commit.Commit.Committer.Date, nil
}

func (c *Client) ValidateOwner(ctx context.Context, owner string) bool {
	var result json.RawMessage
	url := fmt.Sprintf("https://api.github.com/orgs/%s", owner)
	if err := c.get(ctx, url, &result); err == nil {
		return true
	}
	url = fmt.Sprintf("https://api.github.com/users/%s", owner)
	return c.get(ctx, url, &result) == nil
}

func (c *Client) ValidateRepo(ctx context.Context, fullName string) bool {
	var result json.RawMessage
	url := fmt.Sprintf("https://api.github.com/repos/%s", fullName)
	return c.get(ctx, url, &result) == nil
}
