package azuredevops

import (
	"context"
	"fmt"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type Provider struct {
	client *Client
}

func NewProvider(token string, organization string, username string) *Provider {
	return &Provider{
		client: NewClient(token, organization, username),
	}
}

func (p *Provider) GetType() domain.ProviderType {
	return domain.ProviderAzureDevOps
}

func (p *Provider) ListPullRequests(ctx context.Context, username string) ([]domain.PullRequest, error) {
	return nil, fmt.Errorf("azure devops provider not yet implemented")
}

func (p *Provider) GetPullRequest(ctx context.Context, identifier domain.PRIdentifier) (*domain.PullRequest, error) {
	return nil, fmt.Errorf("azure devops provider not yet implemented")
}

func (p *Provider) GetDiff(ctx context.Context, identifier domain.PRIdentifier) (*domain.Diff, error) {
	return nil, fmt.Errorf("azure devops provider not yet implemented")
}

func (p *Provider) GetComments(ctx context.Context, identifier domain.PRIdentifier) ([]domain.Comment, error) {
	return nil, fmt.Errorf("azure devops provider not yet implemented")
}

func (p *Provider) AddComment(ctx context.Context, identifier domain.PRIdentifier, body string, filePath string, line int) error {
	return fmt.Errorf("azure devops provider not yet implemented")
}

func (p *Provider) SubmitReview(ctx context.Context, review domain.Review) error {
	return fmt.Errorf("azure devops provider not yet implemented")
}

func (p *Provider) ValidateCredentials(ctx context.Context) error {
	return p.client.ValidateCredentials(ctx)
}
