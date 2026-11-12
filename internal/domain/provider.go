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

	// ListCommits returns the pull request's commits, oldest first.
	ListCommits(ctx context.Context, identifier PRIdentifier) ([]Commit, error)

	GetCommitDiff(ctx context.Context, identifier PRIdentifier, commitID string) (*Diff, error)

	// GetChecks returns the latest builds or checks that ran for the pull request.
	GetChecks(ctx context.Context, identifier PRIdentifier) (Checks, error)

	// ListLinkedWorkItems returns the work items linked to the pull request (GitHub: the issues its
	// description closes).
	ListLinkedWorkItems(ctx context.Context, identifier PRIdentifier) ([]WorkItem, error)

	// ListAssignedWorkItems returns the user's open bugs, tasks and user stories (GitHub: the open
	// issues assigned to the user in the pull request's repository).
	ListAssignedWorkItems(ctx context.Context, identifier PRIdentifier) ([]WorkItem, error)

	LinkWorkItem(ctx context.Context, identifier PRIdentifier, workItemID int) error

	ValidateCredentials(ctx context.Context) error
}

// PipelineProvider is optional: providers without CI/CD support (currently GitHub) do not implement it,
// so callers must type-assert a Provider before use.
type PipelineProvider interface {
	ListPipelines(ctx context.Context) ([]Pipeline, error)

	ListPipelineRuns(ctx context.Context, identifier PipelineIdentifier) ([]PipelineRun, error)

	GetPipelineRun(ctx context.Context, identifier PipelineRunIdentifier) (*PipelineRun, error)

	// ListActivePipelineRuns returns the unfinished runs requested for the authenticated user,
	// across every project.
	ListActivePipelineRuns(ctx context.Context) ([]PipelineRun, error)

	GetPipelineRunStages(ctx context.Context, identifier PipelineRunIdentifier) ([]PipelineStage, error)

	SubmitStageApproval(ctx context.Context, approval StageApproval) error
}
