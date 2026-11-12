package azuredevops

import (
	"context"
	"sort"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func (p *Provider) ListCommits(ctx context.Context, identifier domain.PRIdentifier) ([]domain.Commit, error) {
	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, identifier.Repository)
	if err != nil {
		return nil, err
	}
	refs, err := p.client.GetPullRequestCommits(ctx, projectID, repoID, identifier.Number)
	if err != nil {
		return nil, err
	}

	commits := make([]domain.Commit, 0, len(*refs))
	for _, ref := range *refs {
		commits = append(commits, convertCommit(ref))
	}
	sort.SliceStable(commits, func(i, j int) bool { return commits[i].Date.Before(commits[j].Date) })
	return commits, nil
}

func convertCommit(ref git.GitCommitRef) domain.Commit {
	commit := domain.Commit{ID: common.GetString(ref.CommitId), Message: common.GetString(ref.Comment)}
	if ref.Author != nil {
		commit.Author = domain.User{Username: common.GetString(ref.Author.Name), Email: common.GetString(ref.Author.Email)}
		if ref.Author.Date != nil {
			commit.Date = ref.Author.Date.Time
		}
	}
	return commit
}

func (p *Provider) GetCommitDiff(ctx context.Context, identifier domain.PRIdentifier, commitID string) (*domain.Diff, error) {
	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, identifier.Repository)
	if err != nil {
		return nil, err
	}
	diffText, err := p.client.GetCommitDiff(ctx, projectID, repoID, commitID)
	if err != nil {
		return nil, err
	}
	return common.ParseUnifiedDiff(diffText), nil
}
