package views

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const viewTestWidth, viewTestHeight = 160, 40

func typeIntoFilter(t *testing.T, update func(tea.Msg) tea.Cmd, text string) {
	t.Helper()
	for _, r := range text {
		update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestListFilter_MatchesAnyFieldCaseInsensitive(t *testing.T) {
	f := newListFilter("")
	f.Activate()
	typeIntoFilter(t, f.Update, "API")

	if !f.Matches("web", "my-api-repo") {
		t.Error("expected match on second field")
	}
	if f.Matches("web", "frontend") {
		t.Error("expected no match")
	}
}

func TestListFilter_EmptyMatchesEverything(t *testing.T) {
	f := newListFilter("")
	if !f.Matches("anything") {
		t.Error("expected empty filter to match")
	}
}

func TestListFilter_ClearResetsTextAndDeactivates(t *testing.T) {
	f := newListFilter("")
	f.Activate()
	typeIntoFilter(t, f.Update, "abc")

	f.Clear()

	if f.Text() != "" || f.IsActive() {
		t.Errorf("expected cleared inactive filter, got text=%q active=%v", f.Text(), f.IsActive())
	}
}

func TestListCursor_ScrollsWithinVisibleWindow(t *testing.T) {
	c := newListCursor()
	c.SetVisibleRows(3)
	const count = 10

	for range 5 {
		c.MoveDown(count)
	}

	start, end := c.VisibleRange(count)
	if c.Index() != 5 || start != 3 || end != 6 {
		t.Errorf("expected index 5 in window [3,6), got index %d window [%d,%d)", c.Index(), start, end)
	}

	for range 10 {
		c.MoveDown(count)
	}
	if c.Index() != count-1 {
		t.Errorf("expected cursor to stop at last item, got %d", c.Index())
	}
}

func TestListCursor_ClampAfterShrink(t *testing.T) {
	c := newListCursor()
	c.SetIndex(8, 10)

	c.Clamp(2)

	if c.Index() != 1 {
		t.Errorf("expected cursor clamped to 1, got %d", c.Index())
	}
	if start, end := c.VisibleRange(2); start != 0 || end != 2 {
		t.Errorf("expected window [0,2), got [%d,%d)", start, end)
	}
}

func TestListCursor_EmptyList(t *testing.T) {
	c := newListCursor()
	c.Clamp(0)

	if start, end := c.VisibleRange(0); start != 0 || end != 0 {
		t.Errorf("expected empty window, got [%d,%d)", start, end)
	}
}

func testPipelines() []domain.Pipeline {
	return []domain.Pipeline{
		{ID: 3, Name: "deploy", Project: "Beta", Repository: "api", PATID: "pat",
			LatestRun: &domain.PipelineRun{Status: domain.PipelineStatusSucceeded}},
		{ID: 1, Name: "ci", Project: "Alpha", Repository: "web", PATID: "pat",
			LatestRun: &domain.PipelineRun{Status: domain.PipelineStatusSucceeded}},
		{ID: 2, Name: "ci", Folder: `\infra`, Project: "Alpha", Repository: "api", PATID: "pat",
			LatestRun: &domain.PipelineRun{Status: domain.PipelineStatusWaitingForApproval}},
	}
}

func newTestPipelineList() *PipelineListViewModel {
	m := NewPipelineListView()
	m.SetSize(viewTestWidth, viewTestHeight)
	m.SetPipelines(testPipelines())
	return m
}

func visibleNodeNames(m *PipelineListViewModel) []string {
	names := make([]string, len(m.visibleNodes))
	for i, node := range m.visibleNodes {
		names[i] = node.Name
	}
	return names
}

func assertVisible(t *testing.T, m *PipelineListViewModel, expected ...string) {
	t.Helper()
	got := visibleNodeNames(m)
	if strings.Join(got, ",") != strings.Join(expected, ",") {
		t.Errorf("visible nodes = %v, want %v", got, expected)
	}
}

func TestPipelineList_StartsWithProjectsFolded(t *testing.T) {
	m := newTestPipelineList()

	assertVisible(t, m, "Alpha", "Beta")
	if m.GetSelectedPipeline() != nil {
		t.Error("expected a group, not a pipeline, to be selected")
	}
}

func TestPipelineList_ToggleSelectedUnfoldsGroupsAndPipelineRuns(t *testing.T) {
	m := newTestPipelineList()

	m.ToggleSelected()
	assertVisible(t, m, "Alpha", "api", "web", "Beta")

	m.MoveDown()
	m.ToggleSelected()
	assertVisible(t, m, "Alpha", "api", `infra\ci`, "web", "Beta")

	m.MoveDown()
	if m.GetSelectedPipeline() == nil || m.GetSelectedPipeline().ID != 2 {
		t.Fatalf("expected pipeline 2 selected, got %+v", m.GetSelectedNode())
	}

	m.ToggleSelected()
	assertVisible(t, m, "Alpha", "api", `infra\ci`, loadingRunsText, "web", "Beta")

	m.MoveDown()
	m.ToggleSelected()
	assertVisible(t, m, "Alpha", "api", `infra\ci`, "web", "Beta")
	if m.GetSelectedPipeline() == nil || m.GetSelectedPipeline().ID != 2 {
		t.Errorf("expected folding from a run row to select its pipeline, got %+v", m.GetSelectedNode())
	}
}

func TestPipelineList_ToggleAllExpandsThenFolds(t *testing.T) {
	m := newTestPipelineList()

	m.ToggleAll()
	assertVisible(t, m, "Alpha", "api", `infra\ci`, "web", "ci", "Beta", "api", "deploy")

	m.MoveDown()
	m.MoveDown()
	m.ToggleAll()
	assertVisible(t, m, "Alpha", "Beta")
	if m.GetSelectedNode().Name != "Alpha" {
		t.Errorf("expected selection to move to the root project, got %s", m.GetSelectedNode().Name)
	}
}

func TestPipelineList_FavoritesSortFirstAtEachLevel(t *testing.T) {
	m := newTestPipelineList()

	m.SetFavorites(map[string]bool{
		domain.ProjectFavoriteKey("pat", "Beta"):            true,
		domain.RepositoryFavoriteKey("pat", "Alpha", "web"): true,
	})
	m.ToggleAll()

	assertVisible(t, m,
		"Alpha/web", "ci", "Beta", "api", "deploy",
		"Beta", "api", "deploy", "Alpha", "web", "ci", "api", `infra\ci`)
	if !strings.Contains(m.View(), favoriteMarker+"Beta") {
		t.Error("expected favorite marker rendered")
	}
}

func sortTestPipelines() []domain.Pipeline {
	return []domain.Pipeline{
		{ID: 1, Name: "zeta", Project: "Alpha", Repository: "web", PATID: "pat",
			LatestRun: &domain.PipelineRun{Status: domain.PipelineStatusSucceeded, QueuedAt: time.Now().Add(-1 * time.Hour)}},
		{ID: 2, Name: "alpha", Project: "Beta", Repository: "api", PATID: "pat",
			LatestRun: &domain.PipelineRun{Status: domain.PipelineStatusFailed, QueuedAt: time.Now().Add(-3 * time.Hour)}},
		{ID: 3, Name: "mid", Project: "Alpha", Repository: "api", PATID: "pat",
			LatestRun: &domain.PipelineRun{Status: domain.PipelineStatusWaitingForApproval, QueuedAt: time.Now().Add(-2 * time.Hour)}},
		{ID: 4, Name: "never", Project: "Beta", Repository: "api", PATID: "pat"},
	}
}

func TestPipelineList_SortModes(t *testing.T) {
	tests := []struct {
		mode     domain.PipelineSortMode
		expected []string
	}{
		{domain.PipelineSortProject, []string{"Alpha", "api", "mid", "web", "zeta", "Beta", "api", "alpha", "never"}},
		{domain.PipelineSortRepository, []string{"Alpha/api", "mid", "Alpha/web", "zeta", "Beta/api", "alpha", "never"}},
		{domain.PipelineSortPipeline, []string{"alpha  (Beta/api)", "mid  (Alpha/api)", "never  (Beta/api)", "zeta  (Alpha/web)"}},
		{domain.PipelineSortLastRun, []string{"zeta  (Alpha/web)", "mid  (Alpha/api)", "alpha  (Beta/api)", "never  (Beta/api)"}},
		{domain.PipelineSortStatus, []string{"mid  (Alpha/api)", "alpha  (Beta/api)", "zeta  (Alpha/web)", "never  (Beta/api)"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			m := NewPipelineListView()
			m.SetPipelines(sortTestPipelines())
			m.SetSortMode(tt.mode)
			m.ToggleAll()

			assertVisible(t, m, tt.expected...)
		})
	}
}

func TestPipelineList_FavoritesModeOrdersByLastViewedAfterFavorites(t *testing.T) {
	m := NewPipelineListView()
	m.SetPipelines(sortTestPipelines())
	m.SetFavorites(map[string]bool{domain.PipelineFavoriteKey("pat", "Alpha", 1): true})
	m.SetLastViewed(map[string]time.Time{
		domain.PipelineFavoriteKey("pat", "Beta", 4):  time.Now(),
		domain.PipelineFavoriteKey("pat", "Alpha", 3): time.Now().Add(-time.Hour),
	})
	m.ToggleAll()

	assertVisible(t, m, "zeta  (Alpha/web)", "Beta", "api", "never", "alpha", "Alpha", "api", "mid", "web", "zeta")
}

func TestPipelineList_SetSortModeFallsBackToFavorites(t *testing.T) {
	m := NewPipelineListView()

	m.SetSortMode("bogus")

	if m.SortMode() != domain.PipelineSortFavorites {
		t.Errorf("expected favorites fallback, got %s", m.SortMode())
	}
	if !strings.Contains(m.View(), "sorted by favorites & recently viewed") {
		t.Error("expected sort label in title")
	}
}

func TestPipelineList_FlatModeFoldKeysAreSafe(t *testing.T) {
	m := NewPipelineListView()
	m.SetPipelines(sortTestPipelines())
	m.SetSortMode(domain.PipelineSortPipeline)

	m.ToggleSelected()
	m.ToggleAll()

	if len(m.visibleNodes) != 4 {
		t.Errorf("expected flat list unaffected by folding, got %d nodes", len(m.visibleNodes))
	}
}

func TestPipelineList_GroupsSurfaceMostUrgentStatus(t *testing.T) {
	m := newTestPipelineList()

	if m.roots[0].Status != domain.PipelineStatusWaitingForApproval {
		t.Errorf("expected Alpha to surface pending approval, got %s", m.roots[0].Status)
	}
	if m.roots[1].Status != domain.PipelineStatusSucceeded {
		t.Errorf("expected Beta to be succeeded, got %s", m.roots[1].Status)
	}
	if m.roots[0].PipelineCount != 2 {
		t.Errorf("expected Alpha pipeline count 2, got %d", m.roots[0].PipelineCount)
	}
}

func TestPipelineList_FilterUnfoldsMatches(t *testing.T) {
	tests := []struct {
		filter   string
		expected []string
	}{
		{"beta", []string{"Beta", "api", "deploy"}},
		{"web", []string{"Alpha", "web", "ci"}},
		{`infra\ci`, []string{"Alpha", "api", `infra\ci`}},
	}

	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			m := newTestPipelineList()

			m.ActivateFilter()
			typeIntoFilter(t, m.UpdateFilter, tt.filter)

			assertVisible(t, m, tt.expected...)
		})
	}
}

func TestPipelineList_ClearFilterRestoresFolding(t *testing.T) {
	m := newTestPipelineList()
	m.ActivateFilter()
	typeIntoFilter(t, m.UpdateFilter, "nomatch")

	if m.GetSelectedNode() != nil {
		t.Fatal("expected nothing visible while filtered")
	}

	m.ClearFilter()

	assertVisible(t, m, "Alpha", "Beta")
}

func TestPipelineList_RefreshKeepsSelection(t *testing.T) {
	m := newTestPipelineList()
	m.ToggleAll()
	m.MoveDown()
	m.MoveDown()
	selected := m.GetSelectedNode().Key

	m.SetPipelines(testPipelines())

	if m.GetSelectedNode().Key != selected {
		t.Errorf("expected selection %s preserved, got %s", selected, m.GetSelectedNode().Key)
	}
}

func TestPipelineList_PreviewRequestsRunsOncePerPipeline(t *testing.T) {
	m := newTestPipelineList()
	m.ToggleAll()
	m.MoveDown()
	m.MoveDown()

	if requests := m.PendingRunRequests(); len(requests) != 0 {
		t.Fatalf("expected no preview request while pane is closed, got %+v", requests)
	}

	m.TogglePreview()
	requests := m.PendingRunRequests()
	if len(requests) != 1 || requests[0].ID != 2 {
		t.Fatalf("expected preview request for pipeline 2, got %+v", requests)
	}
	pipeline := requests[0]
	if len(m.PendingRunRequests()) != 0 {
		t.Error("expected in-flight request not to be repeated")
	}

	m.SetPreviewRuns(pipeline.FavoriteKey(), []domain.PipelineRun{{Number: "20260101.7"}})
	if !strings.Contains(m.View(), "20260101.7") {
		t.Error("expected preview pane to list runs")
	}

	m.ResetPreview()
	if len(m.PendingRunRequests()) == 0 {
		t.Error("expected reset to allow refetching")
	}
}

func TestPipelineList_PreviewForGroupsListsPipelines(t *testing.T) {
	m := newTestPipelineList()
	m.TogglePreview()

	if requests := m.PendingRunRequests(); len(requests) != 0 {
		t.Errorf("expected no fetch for a project preview, got %+v", requests)
	}
	view := m.View()
	for _, expected := range []string{"Pipelines in Alpha", `infra\ci`, "web (1)"} {
		if !strings.Contains(view, expected) {
			t.Errorf("expected preview to contain %q", expected)
		}
	}
}

func TestPipelineDisplayName(t *testing.T) {
	tests := []struct {
		folder   string
		expected string
	}{
		{"", "build"},
		{`\`, "build"},
		{`\team\infra`, `team\infra\build`},
	}
	for _, tt := range tests {
		if got := PipelineDisplayName(domain.Pipeline{Name: "build", Folder: tt.folder}); got != tt.expected {
			t.Errorf("PipelineDisplayName(%q) = %q, want %q", tt.folder, got, tt.expected)
		}
	}
}

func TestPipelineRuns_SetPipelineResetsState(t *testing.T) {
	m := NewPipelineRunsView()
	m.SetPipeline(domain.Pipeline{ID: 1})
	m.SetRuns([]domain.PipelineRun{{ID: 10}})

	m.SetPipeline(domain.Pipeline{ID: 2})

	if m.GetSelectedRun() != nil {
		t.Error("expected runs cleared for new pipeline")
	}
	if !strings.Contains(m.View(), "Loading runs") {
		t.Error("expected loading state for new pipeline")
	}
}

func TestPipelineRuns_RefreshKeepsSelectedRunWhenNewRunsArrive(t *testing.T) {
	m := NewPipelineRunsView()
	m.SetPipeline(domain.Pipeline{ID: 1})
	m.SetRuns([]domain.PipelineRun{{ID: 10}, {ID: 9}})
	m.MoveDown()

	m.SetRuns([]domain.PipelineRun{{ID: 11}, {ID: 10}, {ID: 9}})

	if m.GetSelectedRun().ID != 9 {
		t.Errorf("expected run 9 to stay selected, got %d", m.GetSelectedRun().ID)
	}
}

func TestPipelineRuns_FilterByBranch(t *testing.T) {
	m := NewPipelineRunsView()
	m.SetPipeline(domain.Pipeline{ID: 1})
	m.SetRuns([]domain.PipelineRun{{ID: 10, SourceBranch: "main"}, {ID: 9, SourceBranch: "feature/x"}})

	m.ActivateFilter()
	typeIntoFilter(t, m.UpdateFilter, "feature")

	if m.GetSelectedRun() == nil || m.GetSelectedRun().ID != 9 {
		t.Errorf("expected run 9, got %+v", m.GetSelectedRun())
	}
}

func TestPipelineRuns_FavoritesFloatToTop(t *testing.T) {
	m := NewPipelineRunsView()
	m.SetPipeline(domain.Pipeline{ID: 1})
	m.SetRuns([]domain.PipelineRun{{ID: 10, PATID: "pat"}, {ID: 9, PATID: "pat"}, {ID: 8, PATID: "pat"}})

	m.SetFavorites(map[string]bool{domain.PipelineRunFavoriteKey("pat", "", 8): true})

	if m.visibleRuns[0].ID != 8 || m.visibleRuns[1].ID != 10 {
		t.Errorf("expected favorite run first then queue order, got %v", []int{m.visibleRuns[0].ID, m.visibleRuns[1].ID, m.visibleRuns[2].ID})
	}
	if m.GetSelectedRun().ID != 10 {
		t.Errorf("expected selection to stay on run 10, got %d", m.GetSelectedRun().ID)
	}
}

func TestPipelineRuns_PreviewShowsStages(t *testing.T) {
	m := NewPipelineRunsView()
	m.SetSize(viewTestWidth, viewTestHeight)
	m.SetPipeline(domain.Pipeline{ID: 1})
	m.SetRuns([]domain.PipelineRun{{ID: 10, Number: "run-10"}})
	m.TogglePreview()

	run := m.NextPreviewRequest()
	if run == nil || run.ID != 10 {
		t.Fatalf("expected preview request for run 10, got %+v", run)
	}
	m.SetPreviewStages(run.FavoriteKey(), testStages())

	view := m.View()
	for _, expected := range []string{"Stages of run-10", "Prod", "needs approval"} {
		if !strings.Contains(view, expected) {
			t.Errorf("expected preview to contain %q", expected)
		}
	}
}

func TestPipelineRuns_SetPipelineResetsPreview(t *testing.T) {
	m := NewPipelineRunsView()
	m.SetPipeline(domain.Pipeline{ID: 1})
	m.SetRuns([]domain.PipelineRun{{ID: 10}})
	m.TogglePreview()
	m.SetPreviewStages(m.NextPreviewRequest().FavoriteKey(), testStages())

	m.SetPipeline(domain.Pipeline{ID: 1})
	m.SetRuns([]domain.PipelineRun{{ID: 10}})

	if m.NextPreviewRequest() == nil {
		t.Error("expected stages refetched after re-entering the pipeline")
	}
}

func testStages() []domain.PipelineStage {
	return []domain.PipelineStage{
		{ID: "build", Name: "Build", Status: domain.PipelineStatusSucceeded},
		{ID: "test", Name: "Test", Status: domain.PipelineStatusSucceeded},
		{ID: "prod", Name: "Prod", Status: domain.PipelineStatusWaitingForApproval, PendingApprovalID: "approval-1"},
	}
}

func TestPipelineRun_StagesShowHowLongTheyRan(t *testing.T) {
	m := NewPipelineRunView()
	m.SetSize(160, 30)
	m.SetRun(domain.PipelineRun{ID: 1})
	started := time.Now().Add(-time.Hour)

	m.SetStages([]domain.PipelineStage{
		{ID: "build", Name: "Build", Status: domain.PipelineStatusSucceeded, StartedAt: started, FinishedAt: started.Add(95 * time.Second)},
		{ID: "test", Name: "Test", Status: domain.PipelineStatusRunning, StartedAt: time.Now().Add(-2 * time.Minute)},
		{ID: "prod", Name: "Prod", Status: domain.PipelineStatusQueued},
	})

	view := m.View()
	for _, duration := range []string{"1m35s", "2m0s"} {
		if !strings.Contains(view, duration) {
			t.Errorf("expected stage duration %q in the run view", duration)
		}
	}
}

func TestPipelineRun_FirstLoadSelectsPendingApproval(t *testing.T) {
	m := NewPipelineRunView()
	m.SetRun(domain.PipelineRun{ID: 1})

	m.SetStages(testStages())

	if m.GetSelectedStage().ID != "prod" {
		t.Errorf("expected pending approval stage selected, got %s", m.GetSelectedStage().ID)
	}
	if m.PendingApprovalCount() != 1 {
		t.Errorf("expected 1 pending approval, got %d", m.PendingApprovalCount())
	}
}

func TestPipelineRun_RefreshKeepsSelectedStage(t *testing.T) {
	m := NewPipelineRunView()
	m.SetRun(domain.PipelineRun{ID: 1})
	m.SetStages(testStages())
	m.MoveUp()

	m.SetStages(testStages())

	if m.GetSelectedStage().ID != "test" {
		t.Errorf("expected Test stage to stay selected, got %s", m.GetSelectedStage().ID)
	}
}

func TestPipelineRun_SameRunUpdateKeepsStages(t *testing.T) {
	m := NewPipelineRunView()
	m.SetRun(domain.PipelineRun{ID: 1, Status: domain.PipelineStatusRunning})
	m.SetStages(testStages())

	m.SetRun(domain.PipelineRun{ID: 1, Status: domain.PipelineStatusSucceeded})

	if m.GetSelectedStage() == nil {
		t.Error("expected stages kept when refreshing the same run")
	}
	if m.GetRun().Status != domain.PipelineStatusSucceeded {
		t.Error("expected run status updated")
	}
}

func TestPipelineRun_DifferentRunResetsStages(t *testing.T) {
	m := NewPipelineRunView()
	m.SetRun(domain.PipelineRun{ID: 1})
	m.SetStages(testStages())

	m.SetRun(domain.PipelineRun{ID: 2})

	if m.GetSelectedStage() != nil {
		t.Error("expected stages cleared for a different run")
	}
}

func TestPipelineRun_ViewShowsApprovalHint(t *testing.T) {
	m := NewPipelineRunView()
	m.SetSize(viewTestWidth, viewTestHeight)
	m.SetRun(domain.PipelineRun{ID: 1, Number: "20260101.1", Status: domain.PipelineStatusRunning})
	m.SetStages(testStages())

	view := m.View()

	for _, expected := range []string{"20260101.1", "Prod", approvalHint} {
		if !strings.Contains(view, expected) {
			t.Errorf("expected view to contain %q", expected)
		}
	}
}

func TestStageApproval_GetApprovalCarriesContext(t *testing.T) {
	m := NewStageApprovalView()
	stage := testStages()[2]
	m.Activate("Project", stage, domain.ApprovalDecisionReject)
	typeIntoFilter(t, m.Update, "  not yet  ")

	approval := m.GetApproval()

	expected := domain.StageApproval{
		Project:    "Project",
		ApprovalID: "approval-1",
		Decision:   domain.ApprovalDecisionReject,
		Comment:    "not yet",
	}
	if approval != expected {
		t.Errorf("GetApproval() = %+v, want %+v", approval, expected)
	}
}

func TestStageApproval_DeactivateClearsComment(t *testing.T) {
	m := NewStageApprovalView()
	m.Activate("Project", testStages()[2], domain.ApprovalDecisionApprove)
	typeIntoFilter(t, m.Update, "ok")

	m.Deactivate()
	m.Activate("Project", testStages()[2], domain.ApprovalDecisionApprove)

	if m.GetApproval().Comment != "" {
		t.Error("expected comment cleared between activations")
	}
}

func TestPipelineList_FavoritesSectionIsDividedFromTheRest(t *testing.T) {
	m := newTestPipelineList()
	m.SetFavorites(map[string]bool{domain.PipelineFavoriteKey("pat", "Beta", 3): true})

	assertVisible(t, m, "deploy  (Beta/api)", "Alpha", "Beta")
	if !hasListDivider(m.View()) {
		t.Error("expected a divider between the favorites and the rest")
	}

	m.SetFavorites(map[string]bool{})
	if hasListDivider(m.View()) {
		t.Error("expected no divider without favorites")
	}
}

// hasListDivider looks for a divider row inside the frame, whose own border is drawn with the
// same glyph.
func hasListDivider(view string) bool {
	for line := range strings.SplitSeq(view, "\n") {
		inner := strings.TrimSpace(strings.Trim(strings.TrimSpace(line), "│"))
		if strings.HasPrefix(inner, strings.Repeat(listDividerGlyph, 10)) {
			return true
		}
	}
	return false
}

func TestPipelineList_FavoriteCopiesFoldIndependently(t *testing.T) {
	m := newTestPipelineList()
	m.SetFavorites(map[string]bool{domain.ProjectFavoriteKey("pat", "Alpha"): true})
	if m.GetSelectedNode().ID != domain.ProjectFavoriteKey("pat", "Alpha") {
		t.Fatalf("expected the selection to stay on the main tree's Alpha, got %s", m.GetSelectedNode().ID)
	}

	m.ToggleSelected()

	assertVisible(t, m, "Alpha", "Alpha", "api", "web", "Beta")
}

func TestPipelineList_FavoritesOnlyHidesTheRest(t *testing.T) {
	m := newTestPipelineList()
	m.SetFavorites(map[string]bool{domain.PipelineFavoriteKey("pat", "Beta", 3): true})

	m.ToggleFavoritesOnly()
	assertVisible(t, m, "deploy  (Beta/api)")
	if !strings.Contains(m.View(), favoritesOnlyHint) {
		t.Error("expected the title to say only favorites are shown")
	}

	m.SetFavorites(map[string]bool{})
	if !strings.Contains(m.View(), noFavoritesText) {
		t.Error("expected a hint when there are no favorites")
	}

	m.ToggleFavoritesOnly()
	assertVisible(t, m, "Alpha", "Beta")
}

func TestPipelineList_UnfoldedPipelineRequestsAndShowsItsRuns(t *testing.T) {
	m := NewPipelineListView()
	m.SetSize(viewTestWidth, viewTestHeight)
	m.SetSortMode(domain.PipelineSortPipeline)
	pipeline := domain.Pipeline{ID: 1, Name: "ci", Project: "Alpha", Repository: "web", PATID: "pat"}
	m.SetPipelines([]domain.Pipeline{pipeline})

	m.ToggleSelected()
	requests := m.PendingRunRequests()
	if len(requests) != 1 || requests[0].ID != pipeline.ID {
		t.Fatalf("expected the unfolded pipeline's runs to be requested, got %+v", requests)
	}

	m.SetPreviewRuns(pipeline.FavoriteKey(), []domain.PipelineRun{
		{ID: 10, Number: "run-10", PipelineID: 1, Project: "Alpha", PATID: "pat", Status: domain.PipelineStatusFailed},
	})
	assertVisible(t, m, "ci  (Alpha/web)", "run-10")

	m.MoveDown()
	if run := m.GetSelectedRun(); run == nil || run.ID != 10 {
		t.Fatalf("expected the run row to be selectable, got %+v", m.GetSelectedNode())
	}
	if !strings.Contains(m.View(), "failed") {
		t.Error("expected the run's status in its row")
	}
}

func TestPipelineList_FailedRunsRequestIsRetried(t *testing.T) {
	m := newTestPipelineList()
	m.SetSortMode(domain.PipelineSortPipeline)
	m.TogglePreview()
	pipeline := m.PendingRunRequests()[0]

	m.PreviewRunsFailed(pipeline.FavoriteKey())

	if requests := m.PendingRunRequests(); len(requests) != 1 {
		t.Errorf("expected the failed request to be retried, got %+v", requests)
	}
}
