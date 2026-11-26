package domain

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestTitleFromBranch(t *testing.T) {
	tests := map[string]string{
		"feature/add-login_page": "Add login page",
		"fix-typo":               "Fix typo",
		"main":                   "Main",
		"feature/":               "",
	}
	for branch, expected := range tests {
		if got := TitleFromBranch(branch); got != expected {
			t.Errorf("TitleFromBranch(%q) = %q, want %q", branch, got, expected)
		}
	}
}

// linkingProvider records the work items linked and fails the ones in fail.
type linkingProvider struct {
	Provider
	linked []int
	fail   map[int]error
}

func (p *linkingProvider) LinkWorkItem(_ context.Context, _ PRIdentifier, workItemID int) error {
	p.linked = append(p.linked, workItemID)
	return p.fail[workItemID]
}

func TestLinkWorkItems_TriesEveryWorkItemAndReturnsTheFailures(t *testing.T) {
	provider := &linkingProvider{fail: map[int]error{1: errors.New("does not exist")}}

	failures := LinkWorkItems(context.Background(), provider, PRIdentifier{}, []int{1, 2})

	if !slices.Equal(provider.linked, []int{1, 2}) {
		t.Errorf("expected both linked, got %v", provider.linked)
	}
	if got := DescribeLinkFailures(failures); got != "could not link #1: does not exist" {
		t.Errorf("unexpected failures %q", got)
	}
}
