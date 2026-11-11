package domain

import (
	"context"
	"time"
)

type Provider interface {
	GetType() ProviderType

	ListPullRequests(ctx context.Context, username string) ([]PullRequest, error)

	GetPullRequest(ctx context.Context, identifier PRIdentifier) (*PullRequest, error)

	GetDiff(ctx context.Context, identifier PRIdentifier) (*Diff, error)

	GetComments(ctx context.Context, identifier PRIdentifier) ([]Comment, error)

	AddComment(ctx context.Context, identifier PRIdentifier, body string, filePath string, line int) error

	SubmitReview(ctx context.Context, review Review) error

	MergePullRequest(ctx context.Context, identifier PRIdentifier, mergeMethod string, deleteBranch bool) error

	UpdatePullRequestDescription(ctx context.Context, identifier PRIdentifier, description string) error

	// ListRecentlyPushedBranches returns the branches the user pushed since the given time.
	ListRecentlyPushedBranches(ctx context.Context, since time.Time) ([]PushedBranch, error)

	ListBranches(ctx context.Context, repositoryFullName string) (RepositoryBranches, error)

	CreatePullRequest(ctx context.Context, pr NewPullRequest) (*PullRequest, error)

	ValidateCredentials(ctx context.Context) error
}

// PipelineProvider is optional: providers without CI/CD support (currently GitHub) do not implement it,
// so callers must type-assert a Provider before use.
type PipelineProvider interface {
	ListPipelines(ctx context.Context) ([]Pipeline, error)

	ListPipelineRuns(ctx context.Context, identifier PipelineIdentifier) ([]PipelineRun, error)

	GetPipelineRun(ctx context.Context, identifier PipelineRunIdentifier) (*PipelineRun, error)

	GetPipelineRunStages(ctx context.Context, identifier PipelineRunIdentifier) ([]PipelineStage, error)

	SubmitStageApproval(ctx context.Context, approval StageApproval) error
}
