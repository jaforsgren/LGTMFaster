package views

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	sectionReview = iota
	sectionCreated
	sectionFailing
	sectionReady
)

func dashboardPR(number int, title string, category domain.PRCategory, edit func(pr *domain.PullRequest)) domain.PullRequest {
	pr := domain.PullRequest{
		Number: number, Title: title, Category: category, Status: domain.PRStatusOpen,
		Repository: domain.Repo{FullName: "Proj/api"}, UpdatedAt: time.Now().Add(-time.Duration(number) * time.Minute),
	}
	if edit != nil {
		edit(&pr)
	}
	return pr
}

func dashboardTestPRs() []domain.PullRequest {
	approvedAndMergeable := func(pr *domain.PullRequest) {
		pr.ApprovalStatus = domain.ApprovalStatusApproved
		pr.Mergeable = true
	}
	return []domain.PullRequest{
		dashboardPR(1, "Needs my eyes", domain.PRCategoryAssigned, nil),
		dashboardPR(2, "Already reviewed", domain.PRCategoryAssigned, func(pr *domain.PullRequest) { pr.ReviewedByMe = true }),
		dashboardPR(3, "Draft for me", domain.PRCategoryAssigned, func(pr *domain.PullRequest) { pr.IsDraft = true }),
		dashboardPR(4, "Broken build", domain.PRCategoryAuthored, approvedAndMergeable),
		dashboardPR(5, "Good to go", domain.PRCategoryAuthored, approvedAndMergeable),
		dashboardPR(6, "Not approved", domain.PRCategoryAuthored, nil),
		dashboardPR(7, "Someone else's", domain.PRCategoryOther, nil),
	}
}

func sectionTitles(m *DashboardViewModel, section int) string {
	var titles []string
	for _, pr := range m.sections[section] {
		titles = append(titles, pr.Title)
	}
	return strings.Join(titles, ",")
}

// loadedDashboard has the checks of Broken build failing and of the other own PRs passing.
func loadedDashboard() *DashboardViewModel {
	m := NewDashboardView()
	m.SetSize(160, 40)
	m.SetPRs(dashboardTestPRs())
	for _, pr := range m.PendingChecks() {
		state := domain.CheckStatePassing
		if pr.Number == 4 {
			state = domain.CheckStateFailing
		}
		m.SetChecks(pr.Key(), domain.SummarizeChecks([]domain.Check{{Name: "ci", State: state}}))
	}
	return m
}

func TestDashboard_SortsPullRequestsIntoPanes(t *testing.T) {
	m := NewDashboardView()
	m.SetSize(160, 40)
	m.SetPRs(dashboardTestPRs())

	if pending := m.PendingChecks(); len(pending) != 3 {
		t.Fatalf("expected checks for the three own PRs only, got %d", len(pending))
	}
	if len(m.PendingChecks()) != 0 {
		t.Error("expected checks in flight not to be requested again")
	}
	if sectionTitles(m, sectionCreated) != "Broken build,Good to go,Not approved" {
		t.Errorf("expected every own PR under Created by you, got %s", sectionTitles(m, sectionCreated))
	}

	m = loadedDashboard()
	expected := map[int]string{
		sectionReview:  "Needs my eyes",
		sectionCreated: "Broken build,Good to go,Not approved",
		sectionFailing: "Broken build",
		sectionReady:   "Good to go",
	}
	for section, titles := range expected {
		if got := sectionTitles(m, section); got != titles {
			t.Errorf("section %d: expected %s, got %s", section, titles, got)
		}
	}
	view := m.View()
	for _, title := range []string{"Waiting for your review (1)", "Created by you (3)", "Failing checks (1)", "Ready to merge (1)", "(failing: ci)"} {
		if !strings.Contains(view, title) {
			t.Errorf("expected %q in the dashboard", title)
		}
	}
}

func TestDashboard_FillsTheMainWindowExactly(t *testing.T) {
	m := loadedDashboard()
	main := RenderMainWindow("", m.width, m.height)

	view := m.View()

	if lipgloss.Width(view) != lipgloss.Width(main) || lipgloss.Height(view) != lipgloss.Height(main) {
		t.Errorf("expected the panes to be %dx%d, got %dx%d",
			lipgloss.Width(main), lipgloss.Height(main), lipgloss.Width(view), lipgloss.Height(view))
	}
}

func TestDashboard_EachPaneHasItsOwnCursor(t *testing.T) {
	m := loadedDashboard()
	if m.focus != sectionReview {
		t.Fatalf("expected the first filled pane focused, got %d", m.focus)
	}

	m.FocusNext()
	m.MoveDown()
	if m.GetSelectedPR().Title != "Good to go" {
		t.Fatalf("expected the second own PR selected, got %s", m.GetSelectedPR().Title)
	}

	m.FocusRight()
	if m.focus != sectionReady || m.GetSelectedPR().Title != "Good to go" {
		t.Errorf("expected → to move to the Ready to merge pane, got %d", m.focus)
	}
	m.FocusLeft()
	m.FocusPrev()
	if m.focus != sectionReview || m.GetSelectedPR().Title != "Needs my eyes" {
		t.Errorf("expected focus back on the review pane, got %d", m.focus)
	}
	m.FocusNext()
	if m.GetSelectedPR().Title != "Good to go" {
		t.Error("expected the Created by you pane to keep its cursor")
	}
}

func TestDashboard_FocusStaysWhereTheUserPutIt(t *testing.T) {
	m := loadedDashboard()
	m.FocusRight()

	m.SetPRs(dashboardTestPRs()[3:])

	if m.focus != sectionFailing {
		t.Errorf("expected the chosen pane to stay focused, got %d", m.focus)
	}
}

func TestDashboard_EmptyPanesSaySo(t *testing.T) {
	m := NewDashboardView()
	m.SetSize(160, 40)

	m.SetActiveRuns(nil)

	if strings.Count(m.View(), dashboardEmptySection) != dashboardPaneCount() {
		t.Error("expected every empty pane to say so")
	}
}

func dashboardRun(id int, number string, queuedMinutesAgo int) domain.PipelineRun {
	return domain.PipelineRun{
		ID: id, Number: number, PipelineID: 7, PipelineName: "deploy", Project: "Proj",
		Status: domain.PipelineStatusRunning, QueuedAt: time.Now().Add(-time.Duration(queuedMinutesAgo) * time.Minute),
	}
}

func TestDashboard_ShowsActiveRunsNewestFirst(t *testing.T) {
	m := loadedDashboard()
	if !strings.Contains(m.View(), dashboardRunsLoading) {
		t.Error("expected the runs pane to say it is loading")
	}

	m.SetActiveRuns([]domain.PipelineRun{dashboardRun(1, "older", 30), dashboardRun(2, "newer", 5)})

	if runs := m.ActiveRuns(); runs[0].Number != "newer" || runs[1].Number != "older" {
		t.Errorf("expected the newest run first, got %+v", runs)
	}
	view := m.View()
	for _, text := range []string{dashboardRunsTitle + " (2)", "deploy", "newer", "older"} {
		if !strings.Contains(view, text) {
			t.Errorf("expected %q in the dashboard", text)
		}
	}
}

func TestDashboard_RunsPaneHasItsOwnCursor(t *testing.T) {
	m := loadedDashboard()
	m.SetActiveRuns([]domain.PipelineRun{dashboardRun(1, "newer", 5), dashboardRun(2, "older", 30)})
	if m.GetSelectedRun() != nil {
		t.Fatal("expected no run selected while a pull request pane is focused")
	}

	m.FocusPrev()
	if m.focus != dashboardRunsPane || m.GetSelectedPR() != nil {
		t.Fatalf("expected shift+tab from the first pane to reach the runs pane, got %d", m.focus)
	}
	m.MoveDown()
	if m.GetSelectedRun().Number != "older" {
		t.Errorf("expected the second run selected, got %s", m.GetSelectedRun().Number)
	}
	m.FocusRight()
	m.FocusLeft()
	if m.focus != dashboardRunsPane {
		t.Errorf("expected ←/→ to keep the full-width runs pane focused, got %d", m.focus)
	}

	m.SetActiveRuns([]domain.PipelineRun{dashboardRun(3, "newest", 1), dashboardRun(1, "newer", 5), dashboardRun(2, "older", 30)})
	if m.GetSelectedRun().Number != "older" {
		t.Errorf("expected a reload to keep the cursor on the same run, got %s", m.GetSelectedRun().Number)
	}
	m.FocusNext()
	if m.focus != sectionReview {
		t.Errorf("expected tab from the runs pane to wrap to the first pane, got %d", m.focus)
	}
}

func TestDashboard_RunRowsShowTheStageAndCommit(t *testing.T) {
	m := loadedDashboard()
	run := dashboardRun(1, "20260101.1", 5)
	run.Status = domain.PipelineStatusWaitingForApproval
	run.CurrentStage = "Prod"
	run.CommitMessage = "feature: add login " + strings.Repeat("x", runCommitColumnWidth)

	m.SetActiveRuns([]domain.PipelineRun{run})

	view := m.View()
	for _, text := range []string{"Prod", "waiting for approval", "feature: add login"} {
		if !strings.Contains(view, text) {
			t.Errorf("expected %q in the runs pane", text)
		}
	}
	if strings.Contains(view, run.CommitMessage) {
		t.Error("expected the commit message to be capped")
	}
}
