package views

import (
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestSetSettingsHidesOldOtherPRs(t *testing.T) {
	m := NewPRListView()

	prs := []domain.PullRequest{
		{
			ID:        "old-other",
			Category:  domain.PRCategoryOther,
			CreatedAt: time.Now().Add(-40 * 24 * time.Hour),
		},
		{
			ID:        "recent-other",
			Category:  domain.PRCategoryOther,
			CreatedAt: time.Now().Add(-1 * 24 * time.Hour),
		},
		{
			ID:        "old-authored",
			Category:  domain.PRCategoryAuthored,
			CreatedAt: time.Now().Add(-40 * 24 * time.Hour),
		},
	}

	m.SetPRs(prs)
	if len(m.visiblePRs) != 3 {
		t.Fatalf("Expected 3 visible PRs before filtering, got %d", len(m.visiblePRs))
	}

	m.SetSettings(domain.Settings{HideOldOtherPRs: true, HideOldOtherPRsDays: 30})

	if len(m.visiblePRs) != 2 {
		t.Fatalf("Expected 2 visible PRs after filtering, got %d", len(m.visiblePRs))
	}

	for _, pr := range m.visiblePRs {
		if pr.ID == "old-other" {
			t.Error("Expected old other-person PR to be filtered out")
		}
	}
}

func TestSetSettingsDisabledShowsAllPRs(t *testing.T) {
	m := NewPRListView()

	prs := []domain.PullRequest{
		{
			ID:        "old-other",
			Category:  domain.PRCategoryOther,
			CreatedAt: time.Now().Add(-40 * 24 * time.Hour),
		},
	}

	m.SetPRs(prs)
	m.SetSettings(domain.Settings{HideOldOtherPRs: false, HideOldOtherPRsDays: 30})

	if len(m.visiblePRs) != 1 {
		t.Fatalf("Expected 1 visible PR when filter disabled, got %d", len(m.visiblePRs))
	}
}
