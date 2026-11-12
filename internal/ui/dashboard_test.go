package ui

import (
	"strings"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func dashboardTestModel(t *testing.T) (Model, domain.PullRequest) {
	t.Helper()
	mine := domain.PullRequest{
		Number: 4, Title: "Broken build", Category: domain.PRCategoryAuthored, Status: domain.PRStatusOpen,
		ApprovalStatus: domain.ApprovalStatusApproved, Mergeable: true, PATID: testPATID,
		Repository: domain.Repo{FullName: "Proj/api"},
	}
	provider := &mockProvider{checks: map[int]domain.Checks{
		4: domain.SummarizeChecks([]domain.Check{{Name: "ci", State: domain.CheckStateFailing}}),
	}}
	m := newPRTestModel(t, provider)
	m.prListView.SetPRs([]domain.PullRequest{mine})
	return m, mine
}

func TestOpenDashboard_LoadsChecksOfOwnPullRequests(t *testing.T) {
	m, mine := dashboardTestModel(t)

	m, cmd := m.openDashboard()
	if m.state != ViewDashboard || cmd == nil {
		t.Fatal("expected the dashboard with a checks request")
	}
	m, _ = m.handleDashboardChecksLoaded(cmd().(DashboardChecksLoadedMsg))

	if !strings.Contains(m.View(), "Failing checks (1)") || !strings.Contains(m.View(), mine.Title) {
		t.Error("expected the PR under failing checks")
	}
}

func TestDashboard_InspectingAPullRequestReturnsToTheDashboard(t *testing.T) {
	m, mine := dashboardTestModel(t)
	m, cmd := m.openDashboard()
	m, _ = m.handleDashboardChecksLoaded(cmd().(DashboardChecksLoadedMsg))

	m, _ = handleEnterKey(m)
	if m.state != ViewPRInspect || m.topBar == nil {
		t.Fatalf("expected enter to inspect %s, got %v", mine.Title, m.state)
	}
	m, _ = handleQuitKey(m)

	if m.state != ViewDashboard {
		t.Errorf("expected back to return to the dashboard, got %v", m.state)
	}
}

func TestDashboard_StaysOpenWhenPullRequestsReload(t *testing.T) {
	m, mine := dashboardTestModel(t)
	m, _ = m.openDashboard()

	updated, _ := m.Update(PRsLoadedMsg{prs: []domain.PullRequest{mine}})
	m = updated.(Model)

	if m.state != ViewDashboard {
		t.Errorf("expected a reload not to leave the dashboard, got %v", m.state)
	}
	if m.captureSession().View != domain.SessionViewDashboard {
		t.Error("expected the dashboard to be saved in the session")
	}
}

func TestDashboardKeys_MoveBetweenPanes(t *testing.T) {
	m, _ := dashboardTestModel(t)
	m, _ = m.openDashboard()

	for _, key := range []string{keyTogglePane, keyFocusPaneIn, keyFocusPaneOut, keyPrevPane} {
		var handled bool
		m, _, handled = m.commandRegistry.HandleKey(m, key)
		if !handled {
			t.Errorf("expected %q to be handled on the dashboard", key)
		}
	}
	if help := m.commandRegistry.NavigationHelp(ViewDashboard); !strings.Contains(help, "tab: next pane") || !strings.Contains(help, "→/←: pane beside") {
		t.Errorf("expected the pane keys in the status bar help, got %q", help)
	}
}
