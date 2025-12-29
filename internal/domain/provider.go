package domain

import "context"

type Provider interface {
	GetType() ProviderType

	ListPullRequests(ctx context.Context, username string) ([]PullRequest, error)

	GetPullRequest(ctx context.Context, identifier PRIdentifier) (*PullRequest, error)

	GetDiff(ctx context.Context, identifier PRIdentifier) (*Diff, error)

	GetComments(ctx context.Context, identifier PRIdentifier) ([]Comment, error)

	AddComment(ctx context.Context, identifier PRIdentifier, body string, filePath string, line int) error

	SubmitReview(ctx context.Context, review Review) error

	ValidateCredentials(ctx context.Context) error
}
