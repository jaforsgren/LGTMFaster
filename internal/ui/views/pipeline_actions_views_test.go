package views

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	logViewTestWidth  = 120
	logViewTestHeight = 20
)

func numberedLines(count int) []string {
	lines := make([]string, count)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}
	return lines
}

func openLogView(run domain.PipelineRun, logs []domain.PipelineLog, stage string) *PipelineLogViewModel {
	m := NewPipelineLogView()
	m.SetSize(logViewTestWidth, logViewTestHeight)
	m.Open(run, stage)
	m.SetLogs(logs)
	return m
}

func TestPipelineLogView_PendingLinesRequestReadsOnlyWhatIsMissing(t *testing.T) {
	logs := []domain.PipelineLog{{ID: 1, Name: "Build", Status: domain.PipelineStatusRunning}}
	m := openLogView(domain.PipelineRun{ID: 1, Status: domain.PipelineStatusRunning}, logs, "")

	request, needed := m.PendingLinesRequest()
	if !needed || request.Offset != 0 || request.Finished {
		t.Fatalf("expected a first read of the running step, got %+v", request)
	}
	if _, needed := m.PendingLinesRequest(); needed {
		t.Error("expected no second read while one is in flight")
	}
	m.AppendLines(request, []string{"a", "b"})

	request, _ = m.PendingLinesRequest()
	if request.Offset != 2 {
		t.Errorf("expected the next read after 2 lines, got %d", request.Offset)
	}
	m.AppendLines(LogLinesRequest{Log: logs[0], Offset: 0}, []string{"stale"})
	if got := m.currentLines(); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("expected a read at an outdated offset to be ignored, got %v", got)
	}

	m.AppendLines(LogLinesRequest{Log: logs[0], Offset: 2, Finished: true}, []string{"c"})
	if _, needed := m.PendingLinesRequest(); needed {
		t.Error("expected a step read after it finished not to be read again")
	}
	m.LinesFailed(1)
	m.ReloadLines()
	if request, needed := m.PendingLinesRequest(); !needed || request.Offset != 0 {
		t.Errorf("expected a reload to read from the start, got %+v", request)
	}
}

func TestPipelineLogView_FollowingMovesToTheNewestStepAndEnd(t *testing.T) {
	running := domain.PipelineRun{ID: 1, Status: domain.PipelineStatusRunning}
	m := openLogView(running, []domain.PipelineLog{{ID: 1, Status: domain.PipelineStatusRunning}}, "")
	request, _ := m.PendingLinesRequest()
	m.AppendLines(request, numberedLines(100))
	if !strings.Contains(m.View(), "line 100") || !strings.Contains(m.View(), "[following]") {
		t.Fatal("expected a followed log to show its end")
	}

	m.SetLogs([]domain.PipelineLog{{ID: 1, Status: domain.PipelineStatusSucceeded}, {ID: 2, Status: domain.PipelineStatusRunning}})
	if m.SelectedLog().ID != 2 {
		t.Errorf("expected following to move to the running step, got %d", m.SelectedLog().ID)
	}

	m.ScrollPane(-1)
	if m.IsFollowing() {
		t.Error("expected scrolling up to stop following")
	}
	m.SetLogs([]domain.PipelineLog{{ID: 1}, {ID: 2}, {ID: 3, Status: domain.PipelineStatusRunning}})
	if m.SelectedLog().ID != 2 {
		t.Errorf("expected the step to stay selected when not following, got %d", m.SelectedLog().ID)
	}
}

func TestPipelineLogView_FailedStepOpensOnItsError(t *testing.T) {
	failed := domain.PipelineLog{ID: 1, Status: domain.PipelineStatusFailed}
	m := openLogView(domain.PipelineRun{ID: 1, Status: domain.PipelineStatusFailed}, []domain.PipelineLog{failed}, "")
	lines := numberedLines(200)
	lines[150] = "##[error]boom"
	request, _ := m.PendingLinesRequest()
	m.AppendLines(request, lines)

	if !strings.Contains(m.View(), "##[error]boom") || strings.Contains(m.View(), "line 1 ") {
		t.Error("expected the failed log to open around its error")
	}
}

func TestPipelineLogView_SearchCountsMatchesPerStep(t *testing.T) {
	logs := []domain.PipelineLog{{ID: 1, Name: "Build", Status: domain.PipelineStatusSucceeded}}
	m := openLogView(domain.PipelineRun{ID: 1, Status: domain.PipelineStatusSucceeded}, logs, "")
	request, _ := m.PendingLinesRequest()
	m.AppendLines(request, []string{"Restore", "restore again", "build"})

	m.ActivateSearch()
	for _, r := range "RESTORE" {
		m.UpdateSearch(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.DeactivateSearch()

	view := m.View()
	if !strings.Contains(view, "Build (2)") || !strings.Contains(view, "1/2") {
		t.Errorf("expected the case-insensitive match counts, got:\n%s", view)
	}
	if !m.NextMatch() || m.NextMatch() {
		t.Error("expected exactly one more match after the first")
	}
	if !m.PrevMatch() {
		t.Error("expected to move back to the first match")
	}
}

func TestPipelineLogView_TabHidesTheSteps(t *testing.T) {
	logs := []domain.PipelineLog{{ID: 1, Name: "Compile"}}
	m := openLogView(domain.PipelineRun{ID: 1, Status: domain.PipelineStatusSucceeded}, logs, "")
	m.TogglePreview()
	if !m.IsPaneFocused() || strings.Contains(m.View(), "Steps") {
		t.Error("expected tab to hide the steps and focus the log")
	}
	m.FocusPrevPane()
	if m.IsPaneFocused() || !strings.Contains(m.View(), "Steps") {
		t.Error("expected ← to bring the steps back")
	}
}

func TestPipelineRetryView_OffersTheRetriesTheRunAllows(t *testing.T) {
	m := NewPipelineRetryView()
	m.SetSize(80, 20)
	stage := &domain.PipelineStage{Name: "Deploy", RefName: "deploy", Status: domain.PipelineStatusFailed}
	m.Activate(domain.PipelineRun{ID: 3, Number: "3", Status: domain.PipelineStatusFailed}, stage)

	view := m.View()
	for _, label := range []string{"Retry failed jobs", "Retry stage 'Deploy'", "Rerun the whole pipeline"} {
		if !strings.Contains(view, label) {
			t.Errorf("expected %q in:\n%s", label, view)
		}
	}
	m.MoveDown()
	m.MoveDown()
	m.MoveDown()
	if retry := m.GetRetry(); retry.Scope != domain.PipelineRetryRerun || retry.Run.RunID != 3 {
		t.Errorf("expected the cursor to stop on the rerun, got %+v", retry)
	}
}

func TestBranchPicker_CompletesAndReplacesThePreselectedBranch(t *testing.T) {
	picker := newBranchPicker()
	picker.Reset("feature/x")
	picker.Focus()
	picker.SetBranches(domain.RepositoryBranches{Default: "main", Names: []string{"develop", "feature/x", "main", "release/dev"}})
	if picker.Value() != "main" {
		t.Fatalf("expected the default branch preselected, got %q", picker.Value())
	}

	picker.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("dev")})
	if got := picker.Matches(); !slices.Equal(got, []string{"develop", "release/dev"}) {
		t.Errorf("expected prefix matches first, got %v", got)
	}
	picker.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	if accepted, _ := picker.HandleKey(tea.KeyMsg{Type: tea.KeyEnter}); !accepted || picker.Value() != "release/dev" {
		t.Errorf("expected enter to accept release/dev, got %q", picker.Value())
	}
	picker.SetValue("feature")
	if len(picker.Matches()) != 0 || !picker.IsUnknown("feature") || picker.IsUnknown("main") {
		t.Error("expected the excluded branch left out and unknown branches reported")
	}
}

func triggerTestView() *PipelineTriggerViewModel {
	m := NewPipelineTriggerView()
	m.SetSize(100, 30)
	m.Activate(domain.Pipeline{ID: 7, Name: "deploy", Project: "Proj"})
	m.SetOptions(domain.PipelineTriggerOptions{
		Branches: domain.RepositoryBranches{Default: "main", Names: []string{"main"}},
		Parameters: []domain.PipelineParameter{
			{Name: "retries", DisplayName: "Retries", Type: domain.PipelineParameterNumber, Default: "3"},
			{Name: "dryRun", Type: domain.PipelineParameterBoolean, Default: "false"},
		},
	})
	return m
}

func TestPipelineTriggerView_UsesDefaultsAndValidatesNumbers(t *testing.T) {
	m := triggerTestView()
	trigger, err := m.Trigger()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trigger.Branch != "main" || trigger.Parameters["retries"] != "3" || trigger.Parameters["dryRun"] != "false" {
		t.Errorf("expected the defaults, got %+v", trigger)
	}

	m.HandleKey(tea.KeyMsg{Type: tea.KeyTab})
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if _, err := m.Trigger(); err == nil || !strings.Contains(err.Error(), "Retries must be a number") {
		t.Errorf("expected a number error, got %v", err)
	}
	if action, _ := m.HandleKey(tea.KeyMsg{Type: tea.KeyEsc}); action != PipelineTriggerActionCancel {
		t.Error("expected esc to cancel")
	}
}

func TestPipelineTriggerView_ShowsLoadingUntilOptionsArrive(t *testing.T) {
	m := NewPipelineTriggerView()
	m.SetSize(100, 30)
	m.Activate(domain.Pipeline{ID: 7, Name: "deploy"})
	if !strings.Contains(m.View(), loadingParametersText) {
		t.Error("expected the loading text")
	}
	m.SetOptions(domain.PipelineTriggerOptions{})
	if !strings.Contains(m.View(), noParametersText) {
		t.Error("expected the no-parameters text")
	}
	if _, err := m.Trigger(); err == nil {
		t.Error("expected an error without a branch")
	}
}
