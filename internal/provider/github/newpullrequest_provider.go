package github

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
)

const (
	pushEventType   = "PushEvent"
	branchRefPrefix = "refs/heads/"
)

func (p *Provider) ListRecentlyPushedBranches(ctx context.Context, since time.Time) ([]domain.PushedBranch, error) {
	events, err := p.client.ListEventsSince(ctx, since)
	if err != nil {
		logger.LogError("GITHUB_LIST_PUSHES", "events", err)
		return nil, err
	}
	return pushedBranches(events, since), nil
}

// pushedBranches turns the push events since the given time into one branch each, with its
// latest push.
func pushedBranches(events []*github.Event, since time.Time) []domain.PushedBranch {
	latest := make(map[string]domain.PushedBranch)
	var order []string
	for _, event := range events {
		if event.GetType() != pushEventType || event.GetCreatedAt().Before(since) {
			continue
		}
		payload, err := event.ParsePayload()
		if err != nil {
			continue
		}
		push, ok := payload.(*github.PushEvent)
		if !ok || !strings.HasPrefix(push.GetRef(), branchRefPrefix) {
			continue
		}

		fullName := event.GetRepo().GetName()
		owner, name, _ := strings.Cut(fullName, "/")
		branch := domain.PushedBranch{
			Repository: domain.Repo{Name: name, FullName: fullName, Owner: owner},
			Branch:     strings.TrimPrefix(push.GetRef(), branchRefPrefix),
			PushedAt:   event.GetCreatedAt().Time,
		}
		existing, seen := latest[branch.Key()]
		if !seen {
			order = append(order, branch.Key())
		}
		if !seen || branch.PushedAt.After(existing.PushedAt) {
			latest[branch.Key()] = branch
		}
	}

	branches := make([]domain.PushedBranch, 0, len(order))
	for _, key := range order {
		branches = append(branches, latest[key])
	}
	return branches
}

func (p *Provider) ListBranches(ctx context.Context, repositoryFullName string) (domain.RepositoryBranches, error) {
	owner, repo, err := common.ParseGitHubRepository(repositoryFullName)
	if err != nil {
		return domain.RepositoryBranches{}, err
	}

	repository, err := p.client.GetRepository(ctx, owner, repo)
	if err != nil {
		return domain.RepositoryBranches{}, err
	}
	branches, err := p.client.ListBranches(ctx, owner, repo)
	if err != nil {
		return domain.RepositoryBranches{}, err
	}

	names := make([]string, 0, len(branches))
	for _, branch := range branches {
		names = append(names, branch.GetName())
	}
	sort.Strings(names)
	return domain.RepositoryBranches{Default: repository.GetDefaultBranch(), Names: names}, nil
}

func (p *Provider) CreatePullRequest(ctx context.Context, pr domain.NewPullRequest) (*domain.PullRequest, error) {
	owner, repo, err := common.ParseGitHubRepository(pr.RepositoryFullName)
	if err != nil {
		return nil, err
	}

	created, err := p.client.CreatePullRequest(ctx, owner, repo, &github.NewPullRequest{
		Title: github.String(pr.Title),
		Head:  github.String(pr.SourceBranch),
		Base:  github.String(pr.TargetBranch),
		Body:  github.String(pr.Description),
		Draft: github.Bool(pr.IsDraft),
	})
	if err != nil {
		logger.LogError("GITHUB_CREATE_PR", pr.RepositoryFullName, err)
		return nil, err
	}

	domainPR := p.convertPullRequest(created, p.username)
	return &domainPR, nil
}
