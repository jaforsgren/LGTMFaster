package azuredevops

import (
	"context"
	"io"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

// GitClientInterface defines the subset of git.Client methods we use
// This allows us to mock the Azure DevOps Git client for testing
type GitClientInterface interface {
	GetRepositories(ctx context.Context, args git.GetRepositoriesArgs) (*[]git.GitRepository, error)
	GetPullRequests(ctx context.Context, args git.GetPullRequestsArgs) (*[]git.GitPullRequest, error)
	GetPullRequest(ctx context.Context, args git.GetPullRequestArgs) (*git.GitPullRequest, error)
	GetPullRequestCommits(ctx context.Context, args git.GetPullRequestCommitsArgs) (*git.GetPullRequestCommitsResponseValue, error)
	GetPullRequestIterations(ctx context.Context, args git.GetPullRequestIterationsArgs) (*[]git.GitPullRequestIteration, error)
	GetPullRequestIterationChanges(ctx context.Context, args git.GetPullRequestIterationChangesArgs) (*git.GitPullRequestIterationChanges, error)
	GetBlobContent(ctx context.Context, args git.GetBlobContentArgs) (io.ReadCloser, error)
	GetThreads(ctx context.Context, args git.GetThreadsArgs) (*[]git.GitPullRequestCommentThread, error)
	CreateThread(ctx context.Context, args git.CreateThreadArgs) (*git.GitPullRequestCommentThread, error)
	CreatePullRequestReviewer(ctx context.Context, args git.CreatePullRequestReviewerArgs) (*git.IdentityRefWithVote, error)
	UpdatePullRequest(ctx context.Context, args git.UpdatePullRequestArgs) (*git.GitPullRequest, error)
}
