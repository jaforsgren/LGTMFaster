package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func mergedListPR() domain.PullRequest {
	return domain.PullRequest{Number: 7, Title: "Squashed", Status: domain.PRStatusMerged, MergeCommitID: "0123456789abcdef"}
}

func TestMergeRunsView_LoadingThenEmpty(t *testing.T) {
	m := NewMergeRunsView()
	m.SetSize(120, 30)
	m.Activate(mergedListPR())

	if view := m.View(); !strings.Contains(view, loadingMergeRunsList) || !strings.Contains(view, "merged as 01234567") {
		t.Errorf("expected the loading text and the short merge commit, got %q", view)
	}
	m.SetRuns(nil)
	if !strings.Contains(m.View(), noMergeRunsList) {
		t.Error("expected the empty text")
	}
	if m.HandleKey(tea.KeyMsg{Type: tea.KeyEnter}) != MergeRunsActionNone {
		t.Error("expected enter to do nothing without runs")
	}
}

func TestMergeRunsView_MovesAndOpensTheSelectedRun(t *testing.T) {
	m := NewMergeRunsView()
	m.SetSize(120, 30)
	m.Activate(mergedListPR())
	m.SetRuns([]domain.PipelineRun{{ID: 1, PipelineName: "CI"}, {ID: 2, PipelineName: "Deploy"}})

	m.HandleKey(tea.KeyMsg{Type: tea.KeyDown})

	if m.Selected().ID != 2 || m.HandleKey(tea.KeyMsg{Type: tea.KeyEnter}) != MergeRunsActionOpen {
		t.Error("expected enter to open the second run")
	}
	if m.HandleKey(tea.KeyMsg{Type: tea.KeyEsc}) != MergeRunsActionCancel {
		t.Error("expected esc to close")
	}
}

func TestPendingMergeRunsRequest_OnlyForAShownMergedPROnce(t *testing.T) {
	m := NewPRListView()
	m.SetClosedPRs([]domain.PullRequest{mergedListPR()})

	if m.PendingMergeRunsRequest() != nil {
		t.Error("expected no request while the open PRs are listed")
	}
	m.ShowClosed(true)
	if m.PendingMergeRunsRequest() != nil {
		t.Error("expected no request while the Info pane is closed")
	}
	m.TogglePreview()
	if m.PendingMergeRunsRequest() == nil {
		t.Fatal("expected a request for the shown merged PR")
	}
	if m.PendingMergeRunsRequest() != nil {
		t.Error("expected no second request while the first is in flight")
	}
}

func TestStartMergeRunsRequest_SkipsAbandonedPRs(t *testing.T) {
	m := NewPRListView()
	abandoned := mergedListPR()
	abandoned.Status = domain.PRStatusClosed

	if m.StartMergeRunsRequest(abandoned) {
		t.Error("expected no request for an abandoned PR")
	}
}
