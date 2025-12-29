package domain

import "context"

type Provider interface {
	GetType() ProviderType

	ListPullRequests(ctx context.Context, username string) ([]PullRequest, error)

	GetPullRequest(ctx context.Context, identifier PRIdentifier) (*PullRequest, error)

	GetDiff(ctx context.Context, identifier PRIdentifier) (*Diff, error)

	GetComments(ctx context.Context, identifier PRIdentifier) ([]Comment, error)

	AddComment(ctx context.Context, identifier PRIdentifier, body string, filePath string, line int) error

	// SubmitReview submits a pull request review with comments and an action.
	// Expected behavior: The operation should be atomic or implement best-effort rollback.
	// - GitHub: Truly atomic operation (all-or-nothing via single API call)
	// - Azure DevOps: Sequential operations with best-effort rollback on error
	//   (creates comment threads, then vote; attempts cleanup on failure)
	// If an error occurs, the provider should return a descriptive error indicating
	// whether the submission was fully rejected or partially applied.
	SubmitReview(ctx context.Context, review Review) error

	ValidateCredentials(ctx context.Context) error
}
