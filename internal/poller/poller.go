package poller

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/shadowy-pycoder/release-watcher/internal/domain"
	gh "github.com/shadowy-pycoder/release-watcher/internal/github"
	"github.com/shadowy-pycoder/release-watcher/internal/repository"
)

type Poller struct {
	repos    *repository.All
	github   *gh.Client
	interval time.Duration
	trigger  chan struct{}
	onNew    func(ctx context.Context, rel *domain.Release)
}

func New(repos *repository.All, github *gh.Client, interval time.Duration) *Poller {
	return &Poller{
		repos:    repos,
		github:   github,
		interval: interval,
		trigger:  make(chan struct{}, 1),
	}
}

func (p *Poller) OnNewRelease(fn func(ctx context.Context, rel *domain.Release)) {
	p.onNew = fn
}

func (p *Poller) Trigger() {
	select {
	case p.trigger <- struct{}{}:
	default:
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
		case <-p.trigger:
			slog.Info("poll triggered manually")
			p.poll(ctx)
			for {
				select {
				case <-p.trigger:
				default:
					goto done
				}
			}
		done:
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

		shouldPoll, _ := p.repos.Repository.ShouldPoll(ctx, repo.ID)
		if !shouldPoll {
			continue
		}

		latestRelease := p.pollReleases(ctx, repo, ghRepo.FullName)
		latestTag := p.pollTags(ctx, repo, ghRepo.FullName)

		p.repos.Repository.MarkPolled(ctx, repo.ID)

		latest := latestRelease
		if latestTag.After(latest) {
			latest = latestTag
		}
		if !latest.IsZero() {
			p.repos.Repository.MarkNewActivity(ctx, repo.ID, latest)
		}
	}

	if err := p.repos.Organization.UpdateLastPolled(ctx, org.ID); err != nil {
		slog.Error("failed to update last polled", "org", org.Name, "error", err)
	}

	slog.Info("polled organization", "org", org.Name, "repos", len(ghRepos))
}

func (p *Poller) pollReleases(ctx context.Context, repo *domain.Repository, fullName string) time.Time {
	var latest time.Time

	releases, err := p.github.ListReleases(ctx, fullName)
	if err != nil {
		slog.Debug("failed to list releases", "repo", fullName, "error", err)
		return latest
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

		release := &domain.Release{
			RepoID:       repo.ID,
			TagName:      rel.TagName,
			Name:         rel.Name,
			Body:         rel.Body,
			HTMLURL:      rel.HTMLURL,
			Type:         "release",
			Author:       author,
			AuthorAvatar: avatar,
			PublishedAt:  publishedAt,
			RepoName:     fullName,
			OrgName:      strings.SplitN(fullName, "/", 2)[0],
		}

		inserted, err := p.repos.Release.InsertIfNotExists(ctx, release)
		if err != nil {
			slog.Error("failed to insert release", "repo", fullName, "tag", rel.TagName, "error", err)
			continue
		}

		if inserted && p.onNew != nil {
			p.onNew(ctx, release)
		}

		if publishedAt.After(latest) {
			latest = publishedAt
		}
	}
	return latest
}

func (p *Poller) pollTags(ctx context.Context, repo *domain.Repository, fullName string) time.Time {
	var latest time.Time

	tags, err := p.github.ListTags(ctx, fullName)
	if err != nil {
		slog.Debug("failed to list tags", "repo", fullName, "error", err)
		return latest
	}

	for _, tag := range tags {
		release := &domain.Release{
			RepoID:      repo.ID,
			TagName:     tag.Name,
			Name:        tag.Name,
			HTMLURL:     "https://github.com/" + fullName + "/releases/tag/" + tag.Name,
			Type:        "tag",
			PublishedAt: time.Now().UTC(),
			RepoName:    fullName,
			OrgName:     strings.SplitN(fullName, "/", 2)[0],
		}

		inserted, err := p.repos.Release.InsertIfNotExists(ctx, release)
		if err != nil || !inserted {
			continue
		}

		tagDate := time.Now().UTC()
		if tag.Commit.URL != "" {
			if dateStr, err := p.github.GetCommitDate(ctx, tag.Commit.URL); err == nil {
				if parsed, err := time.Parse(time.RFC3339, dateStr); err == nil {
					tagDate = parsed
					p.repos.Release.UpdatePublishedAt(ctx, repo.ID, tag.Name, parsed)
				}
			}
		}

		release.PublishedAt = tagDate
		if p.onNew != nil {
			p.onNew(ctx, release)
		}

		if tagDate.After(latest) {
			latest = tagDate
		}
	}
	return latest
}
