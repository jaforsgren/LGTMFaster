package ui

import (
	"slices"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestAutoRefreshKey_AvailableInPRList(t *testing.T) {
	registry := NewCommandRegistry()

	for _, binding := range registry.keyBindings {
		if slices.Contains(binding.Keys, keyAutoRefresh) && slices.Contains(binding.AvailableIn, ViewPRList) {
			return
		}
	}
	t.Errorf("expected %q to toggle auto-refresh in the PR list", keyAutoRefresh)
}

func TestPipelineWatchTick_KeepsTickingOutsidePipelineViews(t *testing.T) {
	m := newPRTestModel(t, &mockProvider{})
	m.state = ViewPRInspect
	m.pipelineWatchID = 3

	_, cmd := m.handlePipelineWatchTick(PipelineWatchTickMsg{watchID: 3})

	if cmd == nil {
		t.Error("expected the watch to keep ticking so the PR list refreshes once it is back on screen")
	}
}

func TestBackgroundPRsLoaded_UpdatesTheListWithoutLeavingTheView(t *testing.T) {
	m := newPRTestModel(t, &mockProvider{})
	m.state = ViewPRInspect
	refreshed := domain.PullRequest{Number: 9, Title: "Fresh", PATID: testPATID, Repository: domain.Repo{FullName: "Proj/api"}}

	updated, _ := m.Update(PRsLoadedMsg{prs: []domain.PullRequest{refreshed}, background: true})
	m = updated.(Model)

	if m.state != ViewPRInspect {
		t.Errorf("expected a background reload to stay in the current view, got %v", m.state)
	}
	if prs := m.prListView.AllPRs(); len(prs) != 1 || prs[0].Title != "Fresh" {
		t.Errorf("expected the PR list to hold the reloaded PRs, got %+v", prs)
	}
}

func TestBackgroundPRsLoaded_KeepsTheListWhenAPATFailed(t *testing.T) {
	m := newPRTestModel(t, &mockProvider{})
	kept := domain.PullRequest{Number: 4, Title: "Kept", PATID: testPATID, Repository: domain.Repo{FullName: "Proj/api"}}
	m.prListView.SetPRs([]domain.PullRequest{kept})

	updated, _ := m.Update(PRsLoadedMsg{background: true, failedPATCount: 1})
	m = updated.(Model)

	if prs := m.prListView.AllPRs(); len(prs) != 1 || prs[0].Title != "Kept" {
		t.Errorf("expected the PR list to keep its PRs after a failed refresh, got %+v", prs)
	}
}
