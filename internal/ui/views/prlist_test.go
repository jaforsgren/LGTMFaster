package views

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

func TestPendingCommentRequest_OnlyWhileTheInfoPaneIsOpen(t *testing.T) {
	m := NewPRListView()
	m.SetPRs([]domain.PullRequest{{Number: 1}})

	if m.PendingCommentRequest() != nil {
		t.Fatal("expected no request while the pane is closed")
	}
	m.TogglePreview()
	if m.PendingCommentRequest() == nil {
		t.Fatal("expected the selected PR to be requested")
	}
	if m.PendingCommentRequest() != nil {
		t.Error("expected an in-flight request not to be repeated")
	}
	m.CommentThreadsFailed(m.GetSelectedPR().Key())
	if m.PendingCommentRequest() == nil {
		t.Error("expected a failed request to be retried")
	}
}

func closedTestPRs() []domain.PullRequest {
	now := time.Now()
	return []domain.PullRequest{
		{Number: 1, Title: "Older merge", Status: domain.PRStatusMerged, ClosedAt: now.Add(-48 * time.Hour), MergeCommitID: "abc123"},
		{Number: 2, Title: "Recent abandon", Status: domain.PRStatusClosed, ClosedAt: now.Add(-time.Hour)},
	}
}

func TestShowClosed_ListsClosedPRsMostRecentlyClosedFirst(t *testing.T) {
	m := NewPRListView()
	m.SetSize(120, 30)
	m.SetPRs([]domain.PullRequest{{Number: 9, Title: "Still open", Status: domain.PRStatusOpen}})
	m.SetClosedPRs(closedTestPRs())

	m.ShowClosed(true)

	if len(m.visiblePRs) != 2 || m.visiblePRs[0].Number != 2 || m.visiblePRs[1].Number != 1 {
		t.Fatalf("expected the closed PRs, most recently closed first, got %+v", m.visiblePRs)
	}
	view := m.View()
	if !strings.Contains(view, "Pull Requests / Closed") || strings.Contains(view, "Still open") {
		t.Error("expected the closed list title and no open PRs")
	}
	if len(m.AllPRs()) != 1 || m.AllPRs()[0].Number != 9 {
		t.Error("expected AllPRs to keep only the open PRs")
	}
}

func TestShowClosed_FalseListsOpenPRsAgain(t *testing.T) {
	m := NewPRListView()
	m.SetPRs([]domain.PullRequest{{Number: 9, Status: domain.PRStatusOpen}})
	m.SetClosedPRs(closedTestPRs())
	m.ShowClosed(true)

	m.ShowClosed(false)

	if m.IsShowingClosed() || len(m.visiblePRs) != 1 || m.visiblePRs[0].Number != 9 {
		t.Fatalf("expected the open PRs, got %+v", m.visiblePRs)
	}
}

func TestShowClosed_FilterAppliesToClosedPRs(t *testing.T) {
	m := NewPRListView()
	m.SetClosedPRs(closedTestPRs())
	m.ShowClosed(true)

	m.ActivateFilter()
	for _, r := range "abandon" {
		m.UpdateFilter(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	if len(m.visiblePRs) != 1 || m.visiblePRs[0].Number != 2 {
		t.Fatalf("expected only the matching closed PR, got %+v", m.visiblePRs)
	}
}

func TestPRInfoLines_ShowsClosingAndMergeCommitOfClosedPRs(t *testing.T) {
	m := NewPRListView()
	merged := closedTestPRs()[0]

	info := strings.Join(m.prInfoLines(merged), "\n")

	if !strings.Contains(info, "Closed:") || !strings.Contains(info, "abc123") {
		t.Errorf("expected the closing age and merge commit, got %q", info)
	}
	open := strings.Join(m.prInfoLines(domain.PullRequest{Status: domain.PRStatusOpen}), "\n")
	if strings.Contains(open, "Merged as") {
		t.Error("expected no merge commit row for an open PR")
	}
}

func TestPRList_MarksDrafts(t *testing.T) {
	m := NewPRListView()
	m.SetSize(120, 40)
	m.SetPRs([]domain.PullRequest{
		{Number: 1, Title: "Half done", IsDraft: true, Status: domain.PRStatusOpen},
		{Number: 2, Title: "Finished", Status: domain.PRStatusOpen},
	})

	view := m.View()

	if !strings.Contains(view, draftTitlePrefix+"Half done") {
		t.Error("expected the draft marked")
	}
	if strings.Contains(view, draftTitlePrefix+"Finished") {
		t.Error("expected only drafts marked")
	}
}
