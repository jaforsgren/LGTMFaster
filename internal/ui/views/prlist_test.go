package views

import (
	"strings"
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

func TestSetPRGroups_KeepsSelectionWhenMorePRsStreamIn(t *testing.T) {
	m := NewPRListView()
	m.SetSize(120, 40)
	first := domain.PRGroup{PATID: "a", PRs: []domain.PullRequest{
		{Number: 1, PATID: "a", Category: domain.PRCategoryOther, UpdatedAt: time.Now().Add(-time.Hour)},
		{Number: 2, PATID: "a", Category: domain.PRCategoryOther, UpdatedAt: time.Now().Add(-2 * time.Hour)},
	}}
	m.SetPRGroups([]domain.PRGroup{first})
	m.MoveDown()

	second := domain.PRGroup{PATID: "b", PRs: []domain.PullRequest{
		{Number: 3, PATID: "b", Category: domain.PRCategoryAuthored, UpdatedAt: time.Now()},
	}}
	m.SetPRGroups([]domain.PRGroup{first, second})

	if got := m.GetSelectedPR().Number; got != 2 {
		t.Errorf("expected the selection to stay on #2, got #%d", got)
	}
}

func TestDetailPanes_ShowInfoAndDescriptionOfSelectedPR(t *testing.T) {
	m := NewPRListView()
	m.SetSize(120, 40)
	m.SetPRs([]domain.PullRequest{{
		Number:       7,
		Title:        "Add feature",
		Description:  "Long story",
		Author:       domain.User{Username: "dev"},
		SourceBranch: "feature",
		TargetBranch: "main",
	}})

	panes := m.detailPanes()

	if len(panes) != 2 || panes[0].title != "#7 Add feature" || panes[1].title != prDescriptionTitle {
		t.Fatalf("expected info and description panes, got %+v", panes)
	}
	info := strings.Join(panes[0].lines, "\n")
	for _, expected := range []string{"dev", "feature → main"} {
		if !strings.Contains(info, expected) {
			t.Errorf("expected %q in the info pane", expected)
		}
	}
	if !strings.Contains(strings.Join(panes[1].lines, "\n"), "Long story") {
		t.Error("expected the description in the description pane")
	}
}

func TestDetailPanes_EmptyDescriptionSaysSo(t *testing.T) {
	m := NewPRListView()
	m.SetSize(120, 40)
	m.SetPRs([]domain.PullRequest{{Number: 1}})

	if !strings.Contains(strings.Join(m.detailPanes()[1].lines, "\n"), noDescriptionText) {
		t.Error("expected the empty description message")
	}
}
