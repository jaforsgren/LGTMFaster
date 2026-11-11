package azuredevops

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

const (
	branchRefPrefix = "refs/heads/"
	// deletedRefObjectID is the new object ID of a ref update that deleted the branch.
	deletedRefObjectID = "0000000000000000000000000000000000000000"
)

// ListRecentlyPushedBranches asks every repository for the user's pushes since the given time and
// returns each pushed branch once, with its latest push.
func (p *Provider) ListRecentlyPushedBranches(ctx context.Context, since time.Time) ([]domain.PushedBranch, error) {
	latest := make(map[string]domain.PushedBranch)
	var mu sync.Mutex

	err := p.visitRepositories(ctx, func(projectID, projectName, repoID, repoName string) {
		pushes, err := p.client.ListRecentPushes(ctx, projectID, repoID, since)
		if err != nil {
			logger.Log("AzureDevOps: Skipping pushes of %s/%s: %v", projectName, repoName, err)
			return
		}

		repo := domain.Repo{
			ID:       repoID,
			Name:     repoName,
			FullName: buildRepositoryIdentifier(projectName, repoName),
			Owner:    projectName,
		}
		mu.Lock()
		defer mu.Unlock()
		for _, branch := range pushedBranches(pushes, repo) {
			if existing, seen := latest[branch.Key()]; !seen || branch.PushedAt.After(existing.PushedAt) {
				latest[branch.Key()] = branch
			}
		}
	})

	branches := make([]domain.PushedBranch, 0, len(latest))
	for _, branch := range latest {
		branches = append(branches, branch)
	}
	return branches, err
}

// pushedBranches lists the branches the pushes created or updated; deleted branches are skipped.
func pushedBranches(pushes []git.GitPush, repo domain.Repo) []domain.PushedBranch {
	var branches []domain.PushedBranch
	for _, push := range pushes {
		if push.RefUpdates == nil || push.Date == nil {
			continue
		}
		for _, update := range *push.RefUpdates {
			name := common.GetString(update.Name)
			if !strings.HasPrefix(name, branchRefPrefix) || common.GetString(update.NewObjectId) == deletedRefObjectID {
				continue
			}
			branches = append(branches, domain.PushedBranch{
				Repository: repo,
				Branch:     strings.TrimPrefix(name, branchRefPrefix),
				PushedAt:   push.Date.Time,
			})
		}
	}
	return branches
}

func (p *Provider) ListBranches(ctx context.Context, repositoryFullName string) (domain.RepositoryBranches, error) {
	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, repositoryFullName)
	if err != nil {
		return domain.RepositoryBranches{}, err
	}

	repo, err := p.client.GetRepository(ctx, projectID, repoID)
	if err != nil {
		return domain.RepositoryBranches{}, err
	}
	refs, err := p.client.ListBranchRefs(ctx, projectID, repoID)
	if err != nil {
		return domain.RepositoryBranches{}, err
	}

	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		names = append(names, extractBranchName(ref.Name))
	}
	sort.Strings(names)
	return domain.RepositoryBranches{Default: extractBranchName(repo.DefaultBranch), Names: names}, nil
}

func (p *Provider) CreatePullRequest(ctx context.Context, pr domain.NewPullRequest) (*domain.PullRequest, error) {
	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, pr.RepositoryFullName)
	if err != nil {
		return nil, err
	}
	projectName, repoName, err := parseRepositoryIdentifier(pr.RepositoryFullName)
	if err != nil {
		return nil, err
	}

	sourceRef := branchRefPrefix + pr.SourceBranch
	targetRef := branchRefPrefix + pr.TargetBranch
	created, err := p.client.CreatePullRequest(ctx, projectID, repoID, &git.GitPullRequest{
		SourceRefName: &sourceRef,
		TargetRefName: &targetRef,
		Title:         &pr.Title,
		Description:   &pr.Description,
		IsDraft:       &pr.IsDraft,
	})
	if err != nil {
		return nil, err
	}

	domainPR := p.toDomainPullRequest(created, projectName, repoName, p.client.username)
	return &domainPR, nil
}
