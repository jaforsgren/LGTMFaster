package listing

import (
	"context"
	"errors"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

// listingProvider lists the pull requests it holds, or fails with err.
type listingProvider struct {
	domain.Provider
	prs []domain.PullRequest
	err error
}

func (p listingProvider) ListPullRequests(context.Context, string) ([]domain.PullRequest, error) {
	return p.prs, p.err
}

// pipelineListingProvider also has pipelines.
type pipelineListingProvider struct {
	domain.PipelineProvider
	listingProvider
	pipelines []domain.Pipeline
}

func (p pipelineListingProvider) ListPipelines(context.Context) ([]domain.Pipeline, error) {
	return p.pipelines, p.err
}

var errUnknownPAT = errors.New("unknown PAT")

// factoryOf builds the provider registered for each PAT ID.
func factoryOf(providers map[string]domain.Provider) domain.ProviderFactory {
	return func(pat domain.PAT) (domain.Provider, error) {
		if provider, ok := providers[pat.ID]; ok {
			return provider, nil
		}
		return nil, errUnknownPAT
	}
}

func selectedPAT(id string) domain.PAT {
	return domain.PAT{ID: id, Name: id, IsSelected: true}
}
