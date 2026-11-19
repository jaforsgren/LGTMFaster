package views

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	sectionReview = iota
	sectionCreated
	sectionNews
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
	if m.focus != sectionFailing || m.GetSelectedPR().Title != "Broken build" {
		t.Errorf("expected → to move to the Failing checks pane beside it, got %d", m.focus)
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

	if m.focus != sectionNews {
		t.Errorf("expected the chosen pane to stay focused, got %d", m.focus)
	}
}

func TestDashboard_EmptyPanesShrinkToTheirTitle(t *testing.T) {
	m := loadedDashboard()
	m.SetPRs(dashboardTestPRs()[4:])
	m.SetActiveRuns(nil)

	layout := m.layout()
	for _, section := range []int{sectionReview, sectionFailing} {
		if got := layout.sectionHeight(section); got != dashboardEmptyPaneRows {
			t.Errorf("expected empty section %d to be %d row high, got %d", section, dashboardEmptyPaneRows, got)
		}
	}
	if layout.runs != collapsedPaneHeight {
		t.Errorf("expected the empty runs pane collapsed, got %+v", layout.runs)
	}
	if layout.sectionHeight(sectionCreated) <= layout.sectionHeight(sectionReview) {
		t.Error("expected the filled pane to take the room of the empty one")
	}

	view := m.View()
	for _, title := range []string{"Waiting for your review (0)", "Failing checks (0)", dashboardRunsTitle + " (0)"} {
		if !strings.Contains(view, title) {
			t.Errorf("expected %q in the dashboard", title)
		}
	}
	main := RenderMainWindow("", m.width, m.height)
	if lipgloss.Height(view) != lipgloss.Height(main) {
		t.Errorf("expected the panes to still fill %d rows, got %d", lipgloss.Height(main), lipgloss.Height(view))
	}
}

func TestDashboard_AllEmptyPanesStillFillTheWindow(t *testing.T) {
	m := NewDashboardView()
	m.SetSize(160, 40)
	m.SetActiveRuns(nil)

	main := RenderMainWindow("", m.width, m.height)
	if got := lipgloss.Height(m.View()); got != lipgloss.Height(main) {
		t.Errorf("expected the panes to fill %d rows, got %d", lipgloss.Height(main), got)
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

func newsItem(kind domain.ActivityKind, author, summary string, ago time.Duration, byMe bool) domain.Activity {
	return domain.Activity{Kind: kind, Author: domain.User{Username: author}, Summary: summary, At: time.Now().Add(-ago), ByMe: byMe}
}

// newsDashboard has Anna's approval on Needs my eyes, Bob's two comments and the user's own push on
// Broken build, and only a comment from before the news window on Good to go.
func newsDashboard(t *testing.T) *DashboardViewModel {
	t.Helper()
	m := loadedDashboard()
	pending := m.PendingActivity()
	if len(pending) != 6 {
		t.Fatalf("expected activity for the six authored and assigned PRs, got %d", len(pending))
	}
	if len(m.PendingActivity()) != 0 {
		t.Error("expected activity in flight not to be requested again")
	}
	if !strings.Contains(m.View(), "News (0)"+fmt.Sprintf(dashboardActivityLoading, 6)) {
		t.Error("expected the News pane to say it is reading activity")
	}
	activity := map[int][]domain.Activity{
		1: {newsItem(domain.ActivityKindReview, "Anna", "approved", 5*time.Minute, false)},
		4: {
			newsItem(domain.ActivityKindComment, "Bob", "commented: Why?", 2*time.Hour, false),
			newsItem(domain.ActivityKindPush, "Johan", "pushed", 40*time.Minute, true),
			newsItem(domain.ActivityKindComment, "Bob", "commented: Thanks", 30*time.Minute, false),
		},
		5: {newsItem(domain.ActivityKindComment, "Bob", "commented: Old", 48*time.Hour, false)},
	}
	for _, pr := range pending {
		m.SetActivity(pr.Key(), activity[pr.Number])
	}
	return m
}

func TestDashboard_NewsShowsWhatOthersDidLatelyNewestFirst(t *testing.T) {
	m := newsDashboard(t)

	if got := sectionTitles(m, sectionNews); got != "Needs my eyes,Broken build" {
		t.Errorf("expected the PRs with recent news, latest first, got %s", got)
	}
	view := m.View()
	for _, text := range []string{"News (2)", "Anna approved", "Bob commented: Thanks, +1 more"} {
		if !strings.Contains(view, text) {
			t.Errorf("expected %q in the News pane", text)
		}
	}
}

func TestDashboard_NewsOnlyCountsWhatHappenedSinceThePRWasOpened(t *testing.T) {
	m := newsDashboard(t)
	broken := m.sections[sectionNews][1]

	m.SetLastViewed(map[string]time.Time{broken.Key(): time.Now().Add(-time.Hour)})
	if !strings.Contains(m.View(), "Bob commented: Thanks)") {
		t.Error("expected only the comment after the last view to count")
	}

	m.MarkViewed(broken.Key(), time.Now())
	if got := sectionTitles(m, sectionNews); got != "Needs my eyes" {
		t.Errorf("expected an opened PR to leave the News pane, got %s", got)
	}
}

func TestDashboard_NewsWindowBoundsUnopenedPRs(t *testing.T) {
	m := newsDashboard(t)

	m.SetNewsWindow(72 * time.Hour)

	if got := sectionTitles(m, sectionNews); got != "Needs my eyes,Broken build,Good to go" {
		t.Errorf("expected a wider window to bring in the older comment, got %s", got)
	}
}

func TestDashboard_ResetDetailsRequestsActivityAgain(t *testing.T) {
	m := newsDashboard(t)

	m.ResetDetails()

	if len(m.sections[sectionNews]) != 0 || len(m.PendingActivity()) != 6 {
		t.Error("expected a refresh to drop the news and fetch the activity again")
	}
}

func TestDashboard_NewsSitsAboveFailingChecks(t *testing.T) {
	m := loadedDashboard()

	m.FocusRight()
	if m.focus != sectionNews {
		t.Errorf("expected → from the review pane to reach News, got %d", m.focus)
	}
	m.FocusNext()
	if m.focus != sectionFailing {
		t.Errorf("expected tab from News to reach Failing checks, got %d", m.focus)
	}
}

func TestDashboard_PanesTakeOnlyTheRowsTheirListNeeds(t *testing.T) {
	m := loadedDashboard()
	m.SetActiveRuns([]domain.PipelineRun{dashboardRun(1, "only", 5)})

	layout := m.layout()
	for _, section := range []int{sectionFailing, sectionReady} {
		if got := layout.sectionHeight(section); got != fittedListRows(1) {
			t.Errorf("expected section %d to fit its one PR in %d rows, got %d", section, fittedListRows(1), got)
		}
	}
	if layout.runs.inner != fittedListRows(1) {
		t.Errorf("expected the runs pane to fit its one run in %d rows, got %d", fittedListRows(1), layout.runs.inner)
	}
	if layout.sectionHeight(sectionReview) != fittedListRows(1) || layout.spareRows == 0 {
		t.Errorf("expected every pane fitted and the spare rows left blank below, got %+v", layout)
	}
	main := RenderMainWindow("", m.width, m.height)
	if got := lipgloss.Height(m.View()); got != lipgloss.Height(main) {
		t.Errorf("expected the panes to still fill %d rows, got %d", lipgloss.Height(main), got)
	}
}

func TestDashboard_RunsPaneGrowsIntoRoomTheGridDoesNotNeed(t *testing.T) {
	m := loadedDashboard()
	runs := make([]domain.PipelineRun, 30)
	for i := range runs {
		runs[i] = dashboardRun(i, fmt.Sprintf("run %d", i), i)
	}
	defaultRuns := m.layout().runs.inner

	m.SetActiveRuns(runs)

	if got := m.layout().runs.inner; got <= defaultRuns {
		t.Errorf("expected the runs pane to grow past its share of %d rows, got %d", defaultRuns, got)
	}
	main := RenderMainWindow("", m.width, m.height)
	if got := lipgloss.Height(m.View()); got != lipgloss.Height(main) {
		t.Errorf("expected the panes to still fill %d rows, got %d", lipgloss.Height(main), got)
	}
}

func TestStackPanes_ShortListsLeaveTheRestToLongOnes(t *testing.T) {
	heights := stackPanes(30, []int{fittedListRows(50), 0, fittedListRows(2)})

	if heights[2].inner != fittedListRows(2) || heights[1] != collapsedPaneHeight {
		t.Errorf("expected the short list fitted and the empty pane collapsed, got %+v", heights)
	}
	if fitted := stackPanes(30, []int{fittedListRows(1), 0}); fitted[0].inner != fittedListRows(1) {
		t.Errorf("expected a short list to leave the rows it does not need, got %+v", fitted)
	}
	total := 0
	for _, height := range heights {
		total += height.outer()
	}
	if total != 30 {
		t.Errorf("expected the stack to fill 30 rows, got %d", total)
	}
}
