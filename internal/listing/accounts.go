package listing

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

var errNoSelectedPATs = errors.New("no PATs selected; select one with :pats first")

// accountListing is what one PAT's provider listed, or why it could not.
type accountListing[T any] struct {
	pat   domain.PAT
	items []T
	err   error
}

type listFunc[T any] func(ctx context.Context, pat domain.PAT, provider domain.Provider) ([]T, error)

// listSelectedAccounts lists with every selected PAT at once and returns the results in PAT order.
func listSelectedAccounts[T any](ctx context.Context, pats []domain.PAT, newProvider domain.ProviderFactory, list listFunc[T]) ([]accountListing[T], error) {
	selected := domain.SelectedPATs(pats)
	if len(selected) == 0 {
		return nil, errNoSelectedPATs
	}

	listings := make([]accountListing[T], len(selected))
	var wg sync.WaitGroup
	for i, pat := range selected {
		wg.Go(func() {
			listings[i].pat = pat
			provider, err := newProvider(pat)
			if err != nil {
				listings[i].err = err
				return
			}
			listings[i].items, listings[i].err = list(ctx, pat, provider)
		})
	}
	wg.Wait()
	return listings, nil
}

// accountErrors joins the failures of the listings that were not skipped, naming each PAT.
func accountErrors[T any](listings []accountListing[T], skipped error) error {
	var errs []error
	for _, listing := range listings {
		if listing.err != nil && !errors.Is(listing.err, skipped) {
			errs = append(errs, fmt.Errorf("%s: %w", patLabel(listing.pat), listing.err))
		}
	}
	return errors.Join(errs...)
}

func patLabel(pat domain.PAT) string {
	if pat.Name != "" {
		return pat.Name
	}
	return pat.Username
}
