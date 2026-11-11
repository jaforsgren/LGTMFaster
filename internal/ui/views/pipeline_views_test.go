package views

import (
	"strings"
	"testing"

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
		{ID: 3, Name: "deploy", Project: "Beta", Repository: "api", PATID: "pat"},
		{ID: 1, Name: "ci", Project: "alpha", Repository: "web", PATID: "pat"},
		{ID: 2, Name: "ci", Folder: `\infra`, Project: "Alpha", Repository: "api", PATID: "pat"},
	}
}

func TestPipelineList_SortsByProjectRepoName(t *testing.T) {
	m := NewPipelineListView()
	m.SetSize(viewTestWidth, viewTestHeight)

	m.SetPipelines(testPipelines())

	var order []int
	for range testPipelines() {
		order = append(order, m.GetSelectedPipeline().ID)
		m.MoveDown()
	}
	if order[0] != 2 || order[1] != 1 || order[2] != 3 {
		t.Errorf("expected order [2 1 3], got %v", order)
	}
}

func TestPipelineList_FilterMatchesProjectRepoAndName(t *testing.T) {
	tests := []struct {
		filter     string
		expectedID int
	}{
		{"beta", 3},
		{"web", 1},
		{`infra\ci`, 2},
	}

	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			m := NewPipelineListView()
			m.SetSize(viewTestWidth, viewTestHeight)
			m.SetPipelines(testPipelines())

			m.ActivateFilter()
			typeIntoFilter(t, m.UpdateFilter, tt.filter)

			selected := m.GetSelectedPipeline()
			if selected == nil || selected.ID != tt.expectedID {
				t.Fatalf("expected pipeline %d, got %+v", tt.expectedID, selected)
			}
			m.MoveDown()
			if m.GetSelectedPipeline().ID != tt.expectedID {
				t.Error("expected filter to yield a single pipeline")
			}
		})
	}
}

func TestPipelineList_ClearFilterRestoresAll(t *testing.T) {
	m := NewPipelineListView()
	m.SetPipelines(testPipelines())
	m.ActivateFilter()
	typeIntoFilter(t, m.UpdateFilter, "nomatch")

	if m.GetSelectedPipeline() != nil {
		t.Fatal("expected no pipelines while filtered")
	}

	m.ClearFilter()

	if m.GetSelectedPipeline() == nil || m.GetFilterText() != "" {
		t.Error("expected pipelines visible after clearing filter")
	}
}

func TestPipelineList_RefreshKeepsSelection(t *testing.T) {
	m := NewPipelineListView()
	m.SetPipelines(testPipelines())
	m.MoveDown()
	selectedID := m.GetSelectedPipeline().ID

	m.SetPipelines(testPipelines())

	if m.GetSelectedPipeline().ID != selectedID {
		t.Errorf("expected selection %d preserved, got %d", selectedID, m.GetSelectedPipeline().ID)
	}
}

func TestPipelineList_ViewRendersRows(t *testing.T) {
	m := NewPipelineListView()
	m.SetSize(viewTestWidth, viewTestHeight)
	m.SetPipelines(testPipelines())

	view := m.View()

	for _, expected := range []string{"Pipelines", "Project", "deploy", `infra\ci`} {
		if !strings.Contains(view, expected) {
			t.Errorf("expected view to contain %q", expected)
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

func testStages() []domain.PipelineStage {
	return []domain.PipelineStage{
		{ID: "build", Name: "Build", Status: domain.PipelineStatusSucceeded},
		{ID: "test", Name: "Test", Status: domain.PipelineStatusSucceeded},
		{ID: "prod", Name: "Prod", Status: domain.PipelineStatusWaitingForApproval, PendingApprovalID: "approval-1"},
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

	for _, expected := range []string{"20260101.1", "Prod", approvalHint, "auto-refreshing"} {
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
