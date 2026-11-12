package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

// runCmd runs a command and every command batched into it, returning the messages they produce.
func runCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, inner := range batch {
			msgs = append(msgs, runCmd(inner)...)
		}
		return msgs
	}
	return []tea.Msg{msg}
}

func findMsg[T tea.Msg](t *testing.T, cmd tea.Cmd) T {
	t.Helper()
	for _, msg := range runCmd(cmd) {
		if found, ok := msg.(T); ok {
			return found
		}
	}
	var zero T
	t.Fatalf("expected a %T from the command", zero)
	return zero
}

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
	m, _ = m.handleDashboardChecksLoaded(findMsg[DashboardChecksLoadedMsg](t, cmd))

	if !strings.Contains(m.View(), "Failing checks (1)") || !strings.Contains(m.View(), mine.Title) {
		t.Error("expected the PR under failing checks")
	}
}

func TestDashboard_InspectingAPullRequestReturnsToTheDashboard(t *testing.T) {
	m, mine := dashboardTestModel(t)
	m, cmd := m.openDashboard()
	m, _ = m.handleDashboardChecksLoaded(findMsg[DashboardChecksLoadedMsg](t, cmd))

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

func TestDashboard_PRCommandOpensThePRList(t *testing.T) {
	m, mine := dashboardTestModel(t)
	m, _ = m.openDashboard()

	m, _ = handlePRCommand(m, nil)
	if m.state != ViewPRList {
		t.Fatalf("expected :pr to open the PR list, got %v", m.state)
	}
	updated, _ := m.Update(PRsLoadedMsg{prs: []domain.PullRequest{mine}})

	if updated.(Model).state != ViewPRList {
		t.Errorf("expected the PR list to stay open once PRs load, got %v", updated.(Model).state)
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

func activeRunsTestModel(t *testing.T) (Model, *mockPipelineProvider) {
	t.Helper()
	running := testRun(domain.PipelineStatusRunning)
	running.PipelineName = "deploy"
	running.SourceBranch = "feature/login"
	provider := &mockPipelineProvider{activeRuns: []domain.PipelineRun{running}, run: &running}
	m := sizedTestModel(t, layoutTestWidth, layoutTestHeight)
	m.providers = map[string]domain.Provider{testPATID: provider}
	m.primaryPATID = testPATID
	m.prListView.SetPRs([]domain.PullRequest{{Number: 1, Title: "Mine", Category: domain.PRCategoryAuthored, Status: domain.PRStatusOpen, PATID: testPATID}})
	return m, provider
}

func TestOpenDashboard_ShowsTheUsersActivePipelineRuns(t *testing.T) {
	m, _ := activeRunsTestModel(t)

	m, cmd := m.openDashboard()
	m, _ = m.handleDashboardRunsLoaded(findMsg[DashboardRunsLoadedMsg](t, cmd))

	runs := m.dashboardView.ActiveRuns()
	if len(runs) != 1 || runs[0].PATID != testPATID || runs[0].ProviderType != domain.ProviderAzureDevOps {
		t.Fatalf("expected the active run tagged with its PAT, got %+v", runs)
	}
	view := m.View()
	for _, text := range []string{"Your active pipeline runs (1)", "deploy", "feature/login"} {
		if !strings.Contains(view, text) {
			t.Errorf("expected %q on the dashboard", text)
		}
	}
}

func TestDashboard_ActiveRunsWithoutPipelineProvidersAreEmpty(t *testing.T) {
	m, _ := dashboardTestModel(t)

	m, cmd := m.openDashboard()
	m, _ = m.handleDashboardRunsLoaded(findMsg[DashboardRunsLoadedMsg](t, cmd))

	if !strings.Contains(m.View(), "Your active pipeline runs (0)") {
		t.Error("expected an empty runs pane for providers without pipelines")
	}
}

func TestDashboard_FailedActiveRunsAreReported(t *testing.T) {
	m, provider := activeRunsTestModel(t)
	provider.activeRunsErr = errors.New("forbidden")

	m, cmd := m.openDashboard()
	m, _ = m.handleDashboardRunsLoaded(findMsg[DashboardRunsLoadedMsg](t, cmd))

	if !strings.Contains(m.View(), "Failed to load pipeline runs for 1 PAT(s)") {
		t.Error("expected the failure in the status bar")
	}
}

func TestDashboard_OpeningARunReturnsToTheDashboard(t *testing.T) {
	m, _ := activeRunsTestModel(t)
	m.pipelineRunsView.SetPipeline(domain.Pipeline{ID: 99, Name: "other", Project: "Proj", PATID: testPATID})
	m, cmd := m.openDashboard()
	m, _ = m.handleDashboardRunsLoaded(findMsg[DashboardRunsLoadedMsg](t, cmd))
	for m.dashboardView.GetSelectedRun() == nil {
		m, _ = handleDashboardNextPaneKey(m)
	}

	m, _ = handleEnterKey(m)
	if m.state != ViewPipelineRun || m.pipelineRunView.GetRun().ID != testRun(domain.PipelineStatusRunning).ID {
		t.Fatalf("expected enter to open the run, got %v", m.state)
	}
	if strings.Contains(m.View(), "other") {
		t.Error("expected the top bar to name the run's pipeline, not the runs view's last one")
	}
	m, _ = handleQuitKey(m)

	if m.state != ViewDashboard {
		t.Errorf("expected back to return to the dashboard, got %v", m.state)
	}
}

func TestDashboard_WatchTickReloadsActiveRuns(t *testing.T) {
	m, _ := activeRunsTestModel(t)
	m, _ = m.openDashboard()

	_, cmd := m.handlePipelineWatchTick(PipelineWatchTickMsg{watchID: m.pipelineWatchID})

	if cmd == nil {
		t.Error("expected the tick to reload the active runs")
	}
}
