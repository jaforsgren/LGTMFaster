package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func closedPRsTestModel(t *testing.T) (Model, *mockProvider) {
	t.Helper()
	provider := &mockProvider{closedPRs: []domain.PullRequest{
		{Number: 7, Title: "Squashed feature", Status: domain.PRStatusMerged, ClosedAt: time.Now(), MergeCommitID: "abc123",
			Repository: domain.Repo{FullName: "Proj/api"}},
	}}
	m := newPRTestModel(t, provider)
	m.repository = &mockRepository{pats: map[string]*domain.PAT{
		testPATID: {ID: testPATID, Name: "ado", Provider: domain.ProviderAzureDevOps, IsSelected: true},
	}}
	m.prListView.SetPRs([]domain.PullRequest{{Number: 1, Title: "Open work", Status: domain.PRStatusOpen, PATID: testPATID}})
	return m, provider
}

func TestToggleClosedPRsKey_LoadsAndListsClosedPRsTaggedWithTheirPAT(t *testing.T) {
	m, _ := closedPRsTestModel(t)

	m, cmd := handleToggleClosedPRsKey(m)
	m, _ = m.handleClosedPRsLoaded(findMsg[ClosedPRsLoadedMsg](t, cmd))

	selected := m.prListView.GetSelectedPR()
	if !m.prListView.IsShowingClosed() || selected == nil || selected.Number != 7 {
		t.Fatalf("expected the closed PR to be listed, got %+v", selected)
	}
	if selected.PATID != testPATID || selected.ProviderType != domain.ProviderAzureDevOps {
		t.Errorf("expected the closed PR to carry its PAT, got %+v", selected)
	}
	if view := m.View(); !strings.Contains(view, "Squashed feature") || strings.Contains(view, "Open work") {
		t.Error("expected only the closed PR on screen")
	}
}

func TestToggleClosedPRsKey_SwitchesBackWithoutReloading(t *testing.T) {
	m, _ := closedPRsTestModel(t)
	m, cmd := handleToggleClosedPRsKey(m)
	m, _ = m.handleClosedPRsLoaded(findMsg[ClosedPRsLoadedMsg](t, cmd))

	m, _ = handleToggleClosedPRsKey(m)
	if m.prListView.IsShowingClosed() || m.prListView.GetSelectedPR().Number != 1 {
		t.Fatal("expected the open PRs again")
	}

	m, cmd = handleToggleClosedPRsKey(m)
	for _, msg := range runCmd(cmd) {
		if _, ok := msg.(ClosedPRsLoadedMsg); ok {
			t.Error("expected the loaded closed PRs to be reused")
		}
	}
}

func TestRefreshKey_ReloadsClosedPRsWhileShown(t *testing.T) {
	m, provider := closedPRsTestModel(t)
	m, cmd := handleToggleClosedPRsKey(m)
	m, _ = m.handleClosedPRsLoaded(findMsg[ClosedPRsLoadedMsg](t, cmd))
	provider.closedPRs = append(provider.closedPRs, domain.PullRequest{Number: 8, Status: domain.PRStatusClosed})

	_, cmd = handleRefreshKey(m)

	if loaded := findMsg[ClosedPRsLoadedMsg](t, cmd); len(loaded.prs) != 2 {
		t.Errorf("expected the closed PRs to be read again, got %d", len(loaded.prs))
	}
}

func TestPRCommand_ShowsOpenPRsAfterClosedOnes(t *testing.T) {
	m, _ := closedPRsTestModel(t)
	m, _ = handleClosedPRsCommand(m, nil)

	m, _ = handlePRCommand(m, nil)

	if m.prListView.IsShowingClosed() || m.state != ViewPRList {
		t.Error("expected :pr to list the open pull requests")
	}
}
