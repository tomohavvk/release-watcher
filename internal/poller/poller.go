package poller

import (
	"context"
	"log/slog"
	"time"

	"github.com/shadowy-pycoder/release-watcher/internal/domain"
	gh "github.com/shadowy-pycoder/release-watcher/internal/github"
	"github.com/shadowy-pycoder/release-watcher/internal/repository"
)

type Poller struct {
	repos    *repository.All
	github   *gh.Client
	interval time.Duration
}

func New(repos *repository.All, github *gh.Client, interval time.Duration) *Poller {
	return &Poller{
		repos:    repos,
		github:   github,
		interval: interval,
	}
}

func (p *Poller) Start(ctx context.Context) {
	slog.Info("poller started", "interval", p.interval)

	p.poll(ctx)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("poller stopped")
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	orgs, err := p.repos.Organization.ListAll(ctx)
	if err != nil {
		slog.Error("failed to list organizations", "error", err)
		return
	}

	if len(orgs) == 0 {
		return
	}

	slog.Info("polling organizations", "count", len(orgs))

	for _, org := range orgs {
		if ctx.Err() != nil {
			return
		}
		p.pollOrg(ctx, org)
	}
}

func (p *Poller) pollOrg(ctx context.Context, org domain.Organization) {
	ghRepos, err := p.github.ListRepos(ctx, org.Name)
	if err != nil {
		slog.Error("failed to list repos", "org", org.Name, "error", err)
		return
	}

	for _, ghRepo := range ghRepos {
		if ctx.Err() != nil {
			return
		}

		repo, err := p.repos.Repository.Upsert(ctx, org.ID, ghRepo.Name, ghRepo.FullName)
		if err != nil {
			slog.Error("failed to upsert repo", "repo", ghRepo.FullName, "error", err)
			continue
		}

		p.pollReleases(ctx, repo, ghRepo.FullName)
		p.pollTags(ctx, repo, ghRepo.FullName)
	}

	if err := p.repos.Organization.UpdateLastPolled(ctx, org.ID); err != nil {
		slog.Error("failed to update last polled", "org", org.Name, "error", err)
	}

	slog.Info("polled organization", "org", org.Name, "repos", len(ghRepos))
}

func (p *Poller) pollReleases(ctx context.Context, repo *domain.Repository, fullName string) {
	releases, err := p.github.ListReleases(ctx, fullName)
	if err != nil {
		slog.Debug("failed to list releases", "repo", fullName, "error", err)
		return
	}

	for _, rel := range releases {
		publishedAt, err := time.Parse(time.RFC3339, rel.PublishedAt)
		if err != nil {
			continue
		}

		author := ""
		avatar := ""
		if rel.Author != nil {
			author = rel.Author.Login
			avatar = rel.Author.AvatarURL
		}

		if err := p.repos.Release.Upsert(ctx, &domain.Release{
			RepoID:       repo.ID,
			TagName:      rel.TagName,
			Name:         rel.Name,
			Body:         rel.Body,
			HTMLURL:      rel.HTMLURL,
			Type:         "release",
			Author:       author,
			AuthorAvatar: avatar,
			PublishedAt:  publishedAt,
		}); err != nil {
			slog.Error("failed to upsert release", "repo", fullName, "tag", rel.TagName, "error", err)
		}
	}
}

func (p *Poller) pollTags(ctx context.Context, repo *domain.Repository, fullName string) {
	tags, err := p.github.ListTags(ctx, fullName)
	if err != nil {
		slog.Debug("failed to list tags", "repo", fullName, "error", err)
		return
	}

	for _, tag := range tags {
		if err := p.repos.Release.InsertIfNotExists(ctx, &domain.Release{
			RepoID:      repo.ID,
			TagName:     tag.Name,
			Name:        tag.Name,
			HTMLURL:     "https://github.com/" + fullName + "/releases/tag/" + tag.Name,
			Type:        "tag",
			PublishedAt: time.Now().UTC(),
		}); err != nil {
			// ignore
		}
	}
}
