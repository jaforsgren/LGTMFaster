package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	testBuildLogID  = 11
	testDeployLogID = 12
)

func testLogs(deployStatus domain.PipelineStatus) []domain.PipelineLog {
	return []domain.PipelineLog{
		{ID: testBuildLogID, Stage: "Build", Job: "Linux", Name: "Compile", Status: domain.PipelineStatusSucceeded},
		{ID: testDeployLogID, Stage: "Deploy", Job: "Release", Name: "Upload", Status: deployStatus},
	}
}

func logLinesWithError(count, errorAt int) []string {
	lines := make([]string, count)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}
	lines[errorAt] = "##[error]upload failed"
	return lines
}

// sizedPipelineTestModel is a fully built model at the layout test size, with provider as its
// only PAT.
func sizedPipelineTestModel(t *testing.T, provider domain.Provider) Model {
	t.Helper()
	m := sizedTestModel(t, layoutTestWidth, layoutTestHeight)
	m.ctx = context.Background()
	m.providers = map[string]domain.Provider{testPATID: provider}
	return m
}

// openedLogsModel opens the logs of a run from the run view and delivers the loaded logs.
func openedLogsModel(t *testing.T, provider *mockPipelineProvider, run domain.PipelineRun) Model {
	t.Helper()
	m := sizedPipelineTestModel(t, provider)
	m.state = ViewPipelineRun
	m.pipelineRunView.SetRun(run)
	m.pipelineRunView.SetStages([]domain.PipelineStage{{Name: "Deploy", Status: run.Status, RefName: "deploy"}})

	m, _ = handleLogsKey(m)
	if m.state != ViewPipelineLog {
		t.Fatalf("expected the logs view, got %v", m.state)
	}
	m, cmd := m.handlePipelineLogsLoaded(m.loadPipelineLogs(run)().(PipelineLogsLoadedMsg))
	if cmd != nil {
		m, _ = m.handlePipelineLogLinesLoaded(cmd().(PipelineLogLinesLoadedMsg))
	}
	return m
}

func TestOpenPipelineLogs_OpensFailedStepOnItsFirstError(t *testing.T) {
	failedRun := testRun(domain.PipelineStatusFailed)
	provider := &mockPipelineProvider{
		run:      &failedRun,
		logs:     testLogs(domain.PipelineStatusFailed),
		logLines: map[int][]string{testDeployLogID: logLinesWithError(200, 150)},
	}

	m := openedLogsModel(t, provider, failedRun)

	if log := m.pipelineLogView.SelectedLog(); log == nil || log.ID != testDeployLogID {
		t.Fatalf("expected the failed Deploy step, got %+v", log)
	}
	if m.pipelineLogView.IsFollowing() {
		t.Error("expected a finished run not to be followed")
	}
	if !strings.Contains(m.View(), "##[error]upload failed") {
		t.Error("expected the log to open on its first error")
	}
	if cmd := m.startPipelineLogFollow(); cmd != nil {
		t.Error("expected no follow loop for a finished run")
	}
}

func TestPipelineLogs_FollowReadsOnlyNewLines(t *testing.T) {
	runningRun := testRun(domain.PipelineStatusRunning)
	provider := &mockPipelineProvider{
		run:      &runningRun,
		logs:     testLogs(domain.PipelineStatusRunning),
		logLines: map[int][]string{testDeployLogID: {"one", "two"}},
	}
	m := openedLogsModel(t, provider, runningRun)
	if !m.pipelineLogView.IsFollowing() {
		t.Fatal("expected an unfinished run to be followed")
	}

	provider.logLines[testDeployLogID] = append(provider.logLines[testDeployLogID], "three")
	m, cmd := m.handlePipelineLogTick(PipelineLogTickMsg{watchID: m.pipelineLogWatchID})
	if cmd == nil {
		t.Fatal("expected a tick to reload the followed run")
	}
	m, cmd = m.handlePipelineLogsLoaded(m.loadPipelineLogs(runningRun)().(PipelineLogsLoadedMsg))
	m, _ = m.handlePipelineLogLinesLoaded(cmd().(PipelineLogLinesLoadedMsg))

	if provider.lastLogOffset != 2 {
		t.Errorf("expected the follow to read after the 2 lines shown, got offset %d", provider.lastLogOffset)
	}
	if !strings.Contains(m.View(), "three") {
		t.Error("expected the new line to be shown")
	}
}

func TestPipelineLogs_StaleOrUnfollowedTicksAreDropped(t *testing.T) {
	runningRun := testRun(domain.PipelineStatusRunning)
	provider := &mockPipelineProvider{run: &runningRun, logs: testLogs(domain.PipelineStatusRunning)}
	m := openedLogsModel(t, provider, runningRun)

	if _, cmd := m.handlePipelineLogTick(PipelineLogTickMsg{watchID: m.pipelineLogWatchID - 1}); cmd != nil {
		t.Error("expected a stale tick to be dropped")
	}
	m, _ = handleFollowKey(m)
	if m.pipelineLogView.IsFollowing() {
		t.Fatal("expected F to stop following")
	}
	if _, cmd := m.handlePipelineLogTick(PipelineLogTickMsg{watchID: m.pipelineLogWatchID}); cmd != nil {
		t.Error("expected no reload while not following")
	}
}

func TestPipelineLogs_SearchAndErrorJumps(t *testing.T) {
	failedRun := testRun(domain.PipelineStatusFailed)
	lines := logLinesWithError(300, 250)
	lines[20] = "##[warning]deprecated task"
	provider := &mockPipelineProvider{run: &failedRun, logs: testLogs(domain.PipelineStatusSucceeded),
		logLines: map[int][]string{testDeployLogID: lines}}
	m := openedLogsModel(t, provider, failedRun)

	m, _ = handleSearchKey(m)
	if !m.isInInputMode() {
		t.Fatal("expected the search to take the keys")
	}
	for _, r := range "line 12" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.isInInputMode() || !strings.Contains(m.View(), "Search: line 12") {
		t.Fatal("expected enter to keep the search and leave input mode")
	}

	m, _ = handleNextMatchKey(m)
	if !strings.Contains(m.View(), "2/11") {
		t.Errorf("expected the second of 11 matches (line 12, 120-129), got:\n%s", m.View())
	}

	m, _ = logLevelJumpHandler(domain.LogLineError, true)(m)
	if !strings.Contains(m.View(), "##[error]upload failed") {
		t.Error("expected e to jump to the error")
	}
	m, _ = logLevelJumpHandler(domain.LogLineError, true)(m)
	if !strings.Contains(m.View(), noErrorLineMessage) {
		t.Error("expected a message when there is no further error")
	}
	m, _ = logLevelJumpHandler(domain.LogLineWarning, false)(m)
	if !strings.Contains(m.View(), "##[warning]deprecated task") {
		t.Error("expected W to jump back to the warning")
	}

	m, _ = handleEscKey(m)
	if m.pipelineLogView.SearchQuery() != "" || m.state != ViewPipelineLog {
		t.Error("expected esc to clear the search before leaving")
	}
	m, _ = handleEscKey(m)
	if m.state != ViewPipelineRun {
		t.Errorf("expected esc to go back to the run, got %v", m.state)
	}
}

func TestPipelineLogs_ChangingStepReadsItsLog(t *testing.T) {
	failedRun := testRun(domain.PipelineStatusFailed)
	provider := &mockPipelineProvider{run: &failedRun, logs: testLogs(domain.PipelineStatusFailed),
		logLines: map[int][]string{testBuildLogID: {"compiled"}, testDeployLogID: {"x"}}}
	m := openedLogsModel(t, provider, failedRun)

	m, cmd := handleUpKey(m)
	if cmd == nil {
		t.Fatal("expected the Compile step's log to be read")
	}
	m, _ = m.handlePipelineLogLinesLoaded(cmd().(PipelineLogLinesLoadedMsg))
	if !strings.Contains(m.View(), "compiled") {
		t.Error("expected the Compile log")
	}
	if _, cmd := handleDownKey(m); cmd != nil {
		t.Error("expected a finished step that was read not to be read again")
	}
}

func TestPipelineLogs_FillTheTerminal(t *testing.T) {
	failedRun := testRun(domain.PipelineStatusFailed)
	provider := &mockPipelineProvider{run: &failedRun, logs: testLogs(domain.PipelineStatusFailed),
		logLines: map[int][]string{testDeployLogID: logLinesWithError(500, 10)}}
	m := openedLogsModel(t, provider, failedRun)
	assertFillsTerminal(t, "logs", m, layoutTestWidth, layoutTestHeight)

	m, _ = handleTogglePreviewKey(m)
	assertFillsTerminal(t, "logs without steps", m, layoutTestWidth, layoutTestHeight)

	m, _ = handleSearchKey(m)
	assertFillsTerminal(t, "logs searching", m, layoutTestWidth, layoutTestHeight)
	m.pipelineLogView.ClearSearch()

	m, _ = handleRetryKey(m)
	assertFillsTerminal(t, "retry", m, layoutTestWidth, layoutTestHeight)
	m.pipelineRetryView.Deactivate()

	m.state = ViewPipelineRuns
	m.pipelineRunsView.SetPipeline(testPipeline())
	provider.options = &domain.PipelineTriggerOptions{
		Branches:   domain.RepositoryBranches{Default: "main", Names: []string{"main"}},
		Parameters: []domain.PipelineParameter{{Name: "env", Values: []string{"dev", "prod"}}},
	}
	m, cmd := handleTriggerKey(m)
	m, _ = m.handlePipelineTriggerOptionsLoaded(cmd().(PipelineTriggerOptionsLoadedMsg))
	assertFillsTerminal(t, "trigger", m, layoutTestWidth, layoutTestHeight)
}

func TestRetry_FailedJobsRefreshesTheRun(t *testing.T) {
	failedRun := testRun(domain.PipelineStatusFailed)
	retried := testRun(domain.PipelineStatusRunning)
	provider := &mockPipelineProvider{run: &retried, retried: &retried}
	m := sizedPipelineTestModel(t, provider)
	m.state = ViewPipelineRun
	m.pipelineRunView.SetRun(failedRun)
	m.pipelineRunView.SetStages([]domain.PipelineStage{{Name: "Deploy", RefName: "deploy", Status: domain.PipelineStatusFailed}})

	m, _ = handleRetryKey(m)
	if !m.pipelineRetryView.IsActive() {
		t.Fatal("expected the retry dialog")
	}
	m, _ = m.handlePipelineRetryKey(tea.KeyMsg{Type: tea.KeyDown})
	m, cmd := m.handlePipelineRetryKey(tea.KeyMsg{Type: tea.KeyEnter})
	msg, ok := cmd().(PipelineRetriedMsg)
	if !ok {
		t.Fatalf("expected PipelineRetriedMsg")
	}
	if provider.lastRetry.Scope != domain.PipelineRetryStage || provider.lastRetry.Stage.RefName != "deploy" {
		t.Errorf("expected the selected stage to be retried, got %+v", provider.lastRetry)
	}

	m, _ = m.handlePipelineRetried(msg)
	if m.state != ViewPipelineRun || !strings.Contains(m.View(), "Retry stage 'Deploy'") {
		t.Errorf("expected to stay on the run with a message, got %v", m.state)
	}
}

func TestRetry_RerunOpensTheNewRun(t *testing.T) {
	newRun := testRun(domain.PipelineStatusQueued)
	newRun.ID, newRun.Number = 101, "20260101.2"
	provider := &mockPipelineProvider{run: &newRun, retried: &newRun}
	m := sizedPipelineTestModel(t, provider)
	m.state = ViewPipelineRun
	m.pipelineRunView.SetRun(testRun(domain.PipelineStatusSucceeded))

	m, _ = handleRetryKey(m)
	m, cmd := m.handlePipelineRetryKey(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = m.handlePipelineRetried(cmd().(PipelineRetriedMsg))

	if provider.lastRetry.Scope != domain.PipelineRetryRerun {
		t.Errorf("expected a rerun to be the only choice for a succeeded run, got %s", provider.lastRetry.Scope)
	}
	if run := m.pipelineRunView.GetRun(); m.state != ViewPipelineRun || run.ID != 101 || run.PATID != testPATID {
		t.Errorf("expected the new run to be opened and tagged, got %+v", run)
	}
}

func TestRetry_ProviderError(t *testing.T) {
	provider := &mockPipelineProvider{actionErr: errors.New("not allowed")}
	m := sizedPipelineTestModel(t, provider)
	m.state = ViewPipelineRuns
	if _, cmd := handleRetryKey(m); cmd != nil {
		t.Error("expected nothing to retry without a selected run")
	}
	m.state = ViewPipelineRun
	m.pipelineRunView.SetRun(testRun(domain.PipelineStatusFailed))

	m, _ = handleRetryKey(m)
	_, cmd := m.handlePipelineRetryKey(tea.KeyMsg{Type: tea.KeyEnter})
	if errMsg, ok := cmd().(ErrorMsg); !ok || !errors.Is(errMsg.err, provider.actionErr) {
		t.Errorf("expected the wrapped provider error, got %#v", errMsg)
	}
}

func TestTrigger_StartsTheChosenBranchAndParametersAndOpensTheRun(t *testing.T) {
	queued := testRun(domain.PipelineStatusQueued)
	provider := &mockPipelineProvider{
		run:       &queued,
		triggered: &queued,
		options: &domain.PipelineTriggerOptions{
			Branches: domain.RepositoryBranches{Default: "main", Names: []string{"feature/login", "main"}},
			Parameters: []domain.PipelineParameter{
				{Name: "env", Values: []string{"dev", "prod"}, Default: "dev"},
				{Name: "dryRun", Type: domain.PipelineParameterBoolean, Default: "true"},
			},
		},
	}
	m := sizedPipelineTestModel(t, provider)
	m.state = ViewPipelineRuns
	m.pipelineRunsView.SetPipeline(testPipeline())

	m, cmd := handleTriggerKey(m)
	if !m.pipelineTriggerView.IsActive() || !m.isInInputMode() {
		t.Fatal("expected the trigger form to take the keys")
	}
	m, _ = m.handlePipelineTriggerOptionsLoaded(cmd().(PipelineTriggerOptionsLoadedMsg))

	keys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("feat")},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRight},
		{Type: tea.KeyTab},
		{Type: tea.KeySpace, Runes: []rune(" ")},
		{Type: tea.KeyCtrlS},
	}
	for _, key := range keys {
		m, cmd = m.handlePipelineTriggerKey(key)
	}
	if cmd == nil {
		t.Fatalf("expected ctrl+s to start the run, got:\n%s", m.View())
	}
	msg, ok := cmd().(PipelineTriggeredMsg)
	if !ok {
		t.Fatalf("expected PipelineTriggeredMsg")
	}
	trigger := provider.lastTrigger
	if trigger.Branch != "feature/login" || trigger.Parameters["env"] != "prod" || trigger.Parameters["dryRun"] != "false" {
		t.Errorf("unexpected trigger: %+v", trigger)
	}

	m, _ = m.handlePipelineTriggered(msg)
	if m.pipelineTriggerView.IsActive() || m.state != ViewPipelineRun || m.pipelineRunView.GetRun().PATID != testPATID {
		t.Errorf("expected the form closed and the tagged run opened, got %v", m.state)
	}
}

func TestTrigger_RejectsUnknownBranch(t *testing.T) {
	provider := &mockPipelineProvider{options: &domain.PipelineTriggerOptions{
		Branches: domain.RepositoryBranches{Default: "main", Names: []string{"main"}},
	}}
	m := sizedPipelineTestModel(t, provider)
	m.state = ViewPipelineRuns
	m.pipelineRunsView.SetPipeline(testPipeline())
	m, cmd := handleTriggerKey(m)
	m, _ = m.handlePipelineTriggerOptionsLoaded(cmd().(PipelineTriggerOptionsLoadedMsg))

	m, _ = m.handlePipelineTriggerKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("nope")})
	m, cmd = m.handlePipelineTriggerKey(tea.KeyMsg{Type: tea.KeyCtrlS})

	if cmd != nil || provider.lastTrigger != nil || !m.pipelineTriggerView.IsActive() {
		t.Error("expected an unknown branch to keep the form open without starting a run")
	}
}
