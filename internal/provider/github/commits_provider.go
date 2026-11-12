package github

import (
	"context"

	"github.com/google/go-github/v57/github"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
)

func (p *Provider) ListCommits(ctx context.Context, identifier domain.PRIdentifier) ([]domain.Commit, error) {
	owner, repo, err := common.ParseGitHubRepository(identifier.Repository)
	if err != nil {
		return nil, err
	}
	ghCommits, err := p.client.ListPullRequestCommits(ctx, owner, repo, identifier.Number)
	if err != nil {
		return nil, err
	}
	commits := make([]domain.Commit, 0, len(ghCommits))
	for _, ghCommit := range ghCommits {
		commits = append(commits, convertCommit(ghCommit))
	}
	return commits, nil
}

func convertCommit(ghCommit *github.RepositoryCommit) domain.Commit {
	details := ghCommit.GetCommit()
	return domain.Commit{
		ID:      ghCommit.GetSHA(),
		Message: details.GetMessage(),
		Author:  domain.User{Username: details.GetAuthor().GetName(), Email: details.GetAuthor().GetEmail()},
		Date:    details.GetAuthor().GetDate().Time,
	}
}

func (p *Provider) GetCommitDiff(ctx context.Context, identifier domain.PRIdentifier, commitID string) (*domain.Diff, error) {
	owner, repo, err := common.ParseGitHubRepository(identifier.Repository)
	if err != nil {
		return nil, err
	}
	diffText, err := p.client.GetCommitDiff(ctx, owner, repo, commitID)
	if err != nil {
		return nil, err
	}
	return common.ParseUnifiedDiff(diffText), nil
}
