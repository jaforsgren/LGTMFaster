package ui

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const testPATID = "ado-pat"

type mockPipelineProvider struct {
	mockProvider
	pipelines     []domain.Pipeline
	runs          []domain.PipelineRun
	run           *domain.PipelineRun
	stages        []domain.PipelineStage
	listErr       error
	approvalErr   error
	lastApproval  *domain.StageApproval
	lastRunsQuery domain.PipelineIdentifier
}

func (m *mockPipelineProvider) GetType() domain.ProviderType {
	return domain.ProviderAzureDevOps
}

func (m *mockPipelineProvider) ListPipelines(ctx context.Context) ([]domain.Pipeline, error) {
	return m.pipelines, m.listErr
}

func (m *mockPipelineProvider) ListPipelineRuns(ctx context.Context, identifier domain.PipelineIdentifier) ([]domain.PipelineRun, error) {
	m.lastRunsQuery = identifier
	return m.runs, nil
}

func (m *mockPipelineProvider) GetPipelineRun(ctx context.Context, identifier domain.PipelineRunIdentifier) (*domain.PipelineRun, error) {
	run := *m.run
	return &run, nil
}

func (m *mockPipelineProvider) GetPipelineRunStages(ctx context.Context, identifier domain.PipelineRunIdentifier) ([]domain.PipelineStage, error) {
	return m.stages, nil
}

func (m *mockPipelineProvider) SubmitStageApproval(ctx context.Context, approval domain.StageApproval) error {
	m.lastApproval = &approval
	return m.approvalErr
}

func testPipeline() domain.Pipeline {
	return domain.Pipeline{ID: 7, Name: "deploy", Project: "Proj", Repository: "api", URL: "https://pipeline", PATID: testPATID}
}

func testRun(status domain.PipelineStatus) domain.PipelineRun {
	return domain.PipelineRun{ID: 100, Number: "20260101.1", PipelineID: 7, Project: "Proj", Status: status, URL: "https://run", PATID: testPATID}
}

func pendingApprovalStage() domain.PipelineStage {
	return domain.PipelineStage{ID: "prod", Name: "Prod", Status: domain.PipelineStatusWaitingForApproval, PendingApprovalID: "approval-1"}
}

func createPipelineTestModel(provider domain.Provider) Model {
	m := createTestModel()
	m.ctx = context.Background()
	m.state = ViewPRList
	m.providers = map[string]domain.Provider{testPATID: provider}
	m.repository = &mockRepository{pats: map[string]*domain.PAT{}}
	return m
}

func TestOpenPipelineList_WithoutPipelineProvider_StaysInView(t *testing.T) {
	m := createPipelineTestModel(&mockProvider{})

	newModel, cmd := m.openPipelineList()

	if newModel.state != ViewPRList {
		t.Errorf("expected to stay in PR list, got %v", newModel.state)
	}
	if cmd != nil {
		t.Error("expected no load command without pipeline providers")
	}
}

func TestOpenPipelineList_LoadsAndTagsPipelines(t *testing.T) {
	provider := &mockPipelineProvider{
		pipelines: []domain.Pipeline{{ID: 1, Name: "ci", LatestRun: &domain.PipelineRun{ID: 5}}},
	}
	m := createPipelineTestModel(provider)

	newModel, cmd := m.openPipelineList()

	if newModel.state != ViewPipelineList {
		t.Fatalf("expected pipeline list view, got %v", newModel.state)
	}
	msg, ok := cmd().(PipelinesLoadedMsg)
	if !ok {
		t.Fatal("expected PipelinesLoadedMsg")
	}
	if len(msg.pipelines) != 1 || msg.failedPATCount != 0 {
		t.Fatalf("unexpected message: %+v", msg)
	}
	pipeline := msg.pipelines[0]
	if pipeline.PATID != testPATID || pipeline.ProviderType != domain.ProviderAzureDevOps {
		t.Errorf("expected pipeline tagged with PAT and provider, got %+v", pipeline)
	}
	if pipeline.LatestRun.PATID != testPATID {
		t.Error("expected latest run tagged with PAT")
	}
}

func TestOpenPipelineList_UsesLoadedPipelinesWithoutReload(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.pipelineListView.SetPipelines([]domain.Pipeline{testPipeline()})

	newModel, cmd := m.openPipelineList()

	if newModel.state != ViewPipelineList || cmd != nil {
		t.Error("expected cached pipelines shown without reload")
	}
}

func TestLoadPipelines_CountsFailedPATs(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{listErr: errors.New("boom")})

	msg := m.loadPipelines()().(PipelinesLoadedMsg)

	if msg.failedPATCount != 1 {
		t.Errorf("expected 1 failed PAT, got %d", msg.failedPATCount)
	}
}

func TestEnterKey_DrillsFromPipelineListToRunsToRun(t *testing.T) {
	provider := &mockPipelineProvider{
		runs: []domain.PipelineRun{testRun(domain.PipelineStatusRunning)},
	}
	m := createPipelineTestModel(provider)
	m.state = ViewPipelineList
	m.pipelineListView.SetPipelines([]domain.Pipeline{testPipeline()})

	m, _ = handleEnterKey(m)
	m.pipelineListView.MoveDown()
	m, _ = handleEnterKey(m)
	if m.state != ViewPipelineList {
		t.Fatalf("expected enter on groups to unfold in place, got %v", m.state)
	}
	m.pipelineListView.MoveDown()

	m, cmd := handleEnterKey(m)
	if m.state != ViewPipelineRuns {
		t.Fatalf("expected runs view, got %v", m.state)
	}
	if cmd == nil {
		t.Fatal("expected runs load command")
	}

	runsMsg := m.loadPipelineRuns(testPipeline())().(PipelineRunsLoadedMsg)
	if provider.lastRunsQuery != testPipeline().Identifier() {
		t.Errorf("expected runs queried for pipeline, got %+v", provider.lastRunsQuery)
	}
	m, _ = m.handlePipelineRunsLoaded(runsMsg)

	m, cmd = handleEnterKey(m)
	if m.state != ViewPipelineRun {
		t.Fatalf("expected run view, got %v", m.state)
	}
	if cmd == nil || m.pipelineRunView.GetRun().ID != 100 {
		t.Error("expected selected run loaded")
	}
}

func TestHandlePipelineRunsLoaded_IgnoresStalePipeline(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.pipelineRunsView.SetPipeline(testPipeline())

	m, _ = m.handlePipelineRunsLoaded(PipelineRunsLoadedMsg{
		pipeline: domain.PipelineIdentifier{Project: "Proj", PipelineID: 999},
		patID:    testPATID,
		runs:     []domain.PipelineRun{testRun(domain.PipelineStatusRunning)},
	})

	if m.pipelineRunsView.GetSelectedRun() != nil {
		t.Error("expected runs for another pipeline to be ignored")
	}
}

func TestHandlePipelineRunLoaded_UpdatesRunAndStages(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.state = ViewPipelineRun
	m.pipelineRunView.SetRun(testRun(domain.PipelineStatusRunning))

	m, _ = m.handlePipelineRunLoaded(PipelineRunLoadedMsg{
		run:    testRun(domain.PipelineStatusSucceeded),
		stages: []domain.PipelineStage{pendingApprovalStage()},
	})

	if m.pipelineRunView.GetRun().Status != domain.PipelineStatusSucceeded {
		t.Error("expected run status refreshed")
	}
	if m.pipelineRunView.GetSelectedStage() == nil {
		t.Error("expected stages set")
	}
}

func TestPipelineWatchTick_IgnoresStaleWatch(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.state = ViewPipelineRuns
	m.pipelineRunsView.SetPipeline(testPipeline())
	m.pipelineWatchID = 2

	_, cmd := m.handlePipelineWatchTick(PipelineWatchTickMsg{watchID: 1})

	if cmd != nil {
		t.Error("expected stale tick to be dropped")
	}
}

func TestPipelineWatchTick_RefreshesActiveRun(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.state = ViewPipelineRun
	m.pipelineRunView.SetRun(testRun(domain.PipelineStatusWaitingForApproval))
	m.pipelineWatchID = 3

	_, cmd := m.handlePipelineWatchTick(PipelineWatchTickMsg{watchID: 3})

	if cmd == nil {
		t.Error("expected refresh and next tick for unfinished run")
	}
}

func TestPipelineWatchTick_StopsForFinishedRun(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.state = ViewPipelineRun
	m.pipelineRunView.SetRun(testRun(domain.PipelineStatusSucceeded))
	m.pipelineWatchID = 3

	_, cmd := m.handlePipelineWatchTick(PipelineWatchTickMsg{watchID: 3})

	if cmd != nil {
		t.Error("expected watch to stop for finished run")
	}
}

func TestEnteringPipelineViewInvalidatesPreviousWatch(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.pipelineWatchID = 5

	m, _ = m.enterPipelineRuns(testPipeline())

	if m.pipelineWatchID == 5 {
		t.Error("expected new watch generation")
	}
}

func TestApproveStageKey_RequiresPendingApproval(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.state = ViewPipelineRun
	m.pipelineRunView.SetRun(testRun(domain.PipelineStatusRunning))
	m.pipelineRunView.SetStages([]domain.PipelineStage{{ID: "build", Name: "Build", Status: domain.PipelineStatusRunning}})

	m, _ = handleApproveStageKey(m)

	if m.stageApprovalView.IsActive() {
		t.Error("expected approval dialog not to open for stage without approval")
	}
}

func TestApproveAndRejectStageKeys_OpenDialogWithDecision(t *testing.T) {
	tests := []struct {
		name     string
		handler  KeyHandler
		decision domain.ApprovalDecision
	}{
		{"approve", handleApproveStageKey, domain.ApprovalDecisionApprove},
		{"reject", handleRejectStageKey, domain.ApprovalDecisionReject},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createPipelineTestModel(&mockPipelineProvider{})
			m.state = ViewPipelineRun
			m.pipelineRunView.SetRun(testRun(domain.PipelineStatusWaitingForApproval))
			m.pipelineRunView.SetStages([]domain.PipelineStage{pendingApprovalStage()})

			m, _ = tt.handler(m)

			if !m.stageApprovalView.IsActive() {
				t.Fatal("expected approval dialog to open")
			}
			approval := m.stageApprovalView.GetApproval()
			if approval.Decision != tt.decision || approval.ApprovalID != "approval-1" || approval.Project != "Proj" {
				t.Errorf("unexpected approval: %+v", approval)
			}
		})
	}
}

func TestSubmitStageApproval_CallsProviderAndReportsSuccess(t *testing.T) {
	provider := &mockPipelineProvider{}
	m := createPipelineTestModel(provider)
	m.state = ViewPipelineRun
	m.pipelineRunView.SetRun(testRun(domain.PipelineStatusWaitingForApproval))
	m.pipelineRunView.SetStages([]domain.PipelineStage{pendingApprovalStage()})
	m, _ = handleApproveStageKey(m)

	msg := m.submitStageApproval()()

	if m.stageApprovalView.IsActive() {
		t.Error("expected dialog closed after submit")
	}
	submitted, ok := msg.(StageApprovalSubmittedMsg)
	if !ok {
		t.Fatalf("expected StageApprovalSubmittedMsg, got %T", msg)
	}
	if submitted.stageName != "Prod" || submitted.decision != domain.ApprovalDecisionApprove {
		t.Errorf("unexpected submitted message: %+v", submitted)
	}
	if provider.lastApproval == nil || provider.lastApproval.ApprovalID != "approval-1" {
		t.Errorf("expected provider called with approval, got %+v", provider.lastApproval)
	}
}

func TestSubmitStageApproval_ProviderError(t *testing.T) {
	provider := &mockPipelineProvider{approvalErr: errors.New("not an approver")}
	m := createPipelineTestModel(provider)
	m.state = ViewPipelineRun
	m.pipelineRunView.SetRun(testRun(domain.PipelineStatusWaitingForApproval))
	m.pipelineRunView.SetStages([]domain.PipelineStage{pendingApprovalStage()})
	m, _ = handleRejectStageKey(m)

	msg := m.submitStageApproval()()

	errMsg, ok := msg.(ErrorMsg)
	if !ok || !errors.Is(errMsg.err, provider.approvalErr) {
		t.Errorf("expected wrapped provider error, got %#v", msg)
	}
}

func TestNavigateBack_ThroughPipelineViews(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.state = ViewPipelineList
	m.pipelineListView.SetPipelines([]domain.Pipeline{testPipeline()})
	m, _ = m.enterPipelineRuns(testPipeline())
	m, _ = m.enterPipelineRun(testRun(domain.PipelineStatusRunning))

	newModel, _ := m.navigateBack()
	m = newModel.(Model)
	if m.state != ViewPipelineRuns {
		t.Fatalf("expected runs view, got %v", m.state)
	}

	newModel, _ = m.navigateBack()
	m = newModel.(Model)
	if m.state != ViewPipelineList {
		t.Fatalf("expected pipeline list, got %v", m.state)
	}

	newModel, _ = m.navigateBack()
	m = newModel.(Model)
	if m.state != ViewPRList {
		t.Fatalf("expected PR list, got %v", m.state)
	}
}

func TestPipelineBrowserTarget(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.pipelineListView.SetPipelines([]domain.Pipeline{testPipeline()})
	m.pipelineListView.ToggleAll()
	m.pipelineListView.MoveDown()
	m.pipelineListView.MoveDown()
	m.pipelineRunsView.SetPipeline(testPipeline())
	m.pipelineRunView.SetRun(testRun(domain.PipelineStatusRunning))

	tests := []struct {
		state       ViewState
		expectedURL string
	}{
		{ViewPipelineList, "https://pipeline"},
		{ViewPipelineRuns, "https://pipeline"},
		{ViewPipelineRun, "https://run"},
	}
	for _, tt := range tests {
		m.state = tt.state
		if url, _ := m.pipelineBrowserTarget(); url != tt.expectedURL {
			t.Errorf("state %v: expected %s, got %s", tt.state, tt.expectedURL, url)
		}
	}

	m.state = ViewPipelineRuns
	m.pipelineRunsView.SetRuns([]domain.PipelineRun{testRun(domain.PipelineStatusRunning)})
	if url, _ := m.pipelineBrowserTarget(); url != "https://run" {
		t.Errorf("expected selected run URL in runs view, got %s", url)
	}
}

func TestPipelineKeyBindingsRegistered(t *testing.T) {
	registry := NewCommandRegistry()

	for _, key := range []string{"a", "x", "ctrl+o", "r", "enter"} {
		found := false
		for _, binding := range registry.keyBindings {
			for _, k := range binding.Keys {
				if k == key && (isInViews(ViewPipelineRun, binding.AvailableIn) || isInViews(ViewPipelineList, binding.AvailableIn)) {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("expected key %q bound in a pipeline view", key)
		}
	}

	for _, name := range []string{"pipelines", "pl", "pipes"} {
		if _, ok := registry.commands[name]; !ok {
			t.Errorf("expected command %q registered", name)
		}
	}
}

func TestPipelineShortcutsShowApprovalOnlyInRunView(t *testing.T) {
	registry := NewCommandRegistry()

	if !containsShortcut(registry.GetContextualShortcuts(ViewPipelineRun, false), "Approve stage") {
		t.Error("expected approve shortcut in run view")
	}
	if containsShortcut(registry.GetContextualShortcuts(ViewPipelineList, false), "Approve stage") {
		t.Error("expected no approve shortcut in pipeline list")
	}
}

func TestFavoriteKey_TogglesSelectedNodeAndPersists(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	repo := m.repository.(*mockRepository)
	m.state = ViewPipelineList
	m.pipelineListView.SetPipelines([]domain.Pipeline{testPipeline()})

	m, _ = handleFavoriteKey(m)

	projectKey := domain.ProjectFavoriteKey(testPATID, "Proj")
	if len(repo.pipelineFavorites) != 1 || repo.pipelineFavorites[0] != projectKey {
		t.Fatalf("expected project favorited, got %v", repo.pipelineFavorites)
	}

	m, _ = handleFavoriteKey(m)
	if len(repo.pipelineFavorites) != 0 {
		t.Errorf("expected favorite toggled off, got %v", repo.pipelineFavorites)
	}
}

func TestFavoriteKey_InRunView_FavoritesRun(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	repo := m.repository.(*mockRepository)
	m.state = ViewPipelineRun
	m.pipelineRunView.SetRun(testRun(domain.PipelineStatusRunning))

	handleFavoriteKey(m)

	if len(repo.pipelineFavorites) != 1 || repo.pipelineFavorites[0] != testRun(domain.PipelineStatusRunning).FavoriteKey() {
		t.Errorf("expected run favorited, got %v", repo.pipelineFavorites)
	}
}

func TestTogglePreviewKey_LoadsRunsForSelectedPipeline(t *testing.T) {
	provider := &mockPipelineProvider{runs: []domain.PipelineRun{testRun(domain.PipelineStatusRunning)}}
	m := createPipelineTestModel(provider)
	m.state = ViewPipelineList
	m.pipelineListView.SetPipelines([]domain.Pipeline{testPipeline()})
	m.pipelineListView.ToggleAll()
	m.pipelineListView.MoveDown()
	m.pipelineListView.MoveDown()

	m, cmd := handleTogglePreviewKey(m)
	if !m.pipelineListView.IsPreviewOpen() || cmd == nil {
		t.Fatal("expected pane open and runs requested")
	}

	m, _ = m.handlePipelineRunsLoaded(cmd().(PipelineRunsLoadedMsg))
	if requests := m.pipelineListView.PendingRunRequests(); len(requests) != 0 {
		t.Errorf("expected preview runs cached after load, got %+v", requests)
	}
}

func TestFoldKeys_OnlyActInPipelineList(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.pipelineListView.SetPipelines([]domain.Pipeline{testPipeline()})
	m.state = ViewPipelineRuns

	handleFoldAllKey(m)
	if m.pipelineListView.GetSelectedPipeline() != nil {
		t.Error("expected fold keys ignored outside pipeline list")
	}

	m.state = ViewPipelineList
	handleFoldAllKey(m)
	m.pipelineListView.MoveDown()
	m.pipelineListView.MoveDown()
	if m.pipelineListView.GetSelectedPipeline() == nil {
		t.Error("expected Z to unfold all in pipeline list")
	}
}

func TestPipelineShortcutsIncludeNewKeys(t *testing.T) {
	registry := NewCommandRegistry()

	listShortcuts := registry.GetContextualShortcuts(ViewPipelineList, false)
	for _, description := range []string{"Fold/unfold", "Fold/unfold all", "Toggle favorite", "Toggle details pane", "Open in browser"} {
		if !containsShortcut(listShortcuts, description) {
			t.Errorf("expected %q in pipeline list shortcuts", description)
		}
	}
	if !containsShortcut(registry.GetContextualShortcuts(ViewPipelineRuns, false), "Toggle details pane") {
		t.Error("expected details pane shortcut in runs view")
	}
	if !containsShortcut(registry.GetContextualShortcuts(ViewPipelineRun, false), "Toggle favorite") {
		t.Error("expected favorite shortcut in run view")
	}
}

func TestSortKeys_SetAndPersistSortMode(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	repo := m.repository.(*mockRepository)
	m.state = ViewPipelineList

	for i, mode := range domain.PipelineSortModes {
		newModel, _, handled := m.commandRegistry.HandleKey(m, strconv.Itoa(i))
		if !handled {
			t.Fatalf("expected key %d handled", i)
		}
		m = newModel
		if m.pipelineListView.SortMode() != mode || repo.settings.PipelineSortMode != mode {
			t.Errorf("key %d: expected mode %s, got view=%s saved=%s", i, mode, m.pipelineListView.SortMode(), repo.settings.PipelineSortMode)
		}
	}
}

func TestSortKeys_IgnoredOutsidePipelineList(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.state = ViewPipelineRuns

	m, _ = m.setPipelineSortMode(domain.PipelineSortStatus)

	if m.pipelineListView.SortMode() != domain.PipelineSortFavorites {
		t.Error("expected sort mode unchanged outside pipeline list")
	}
}

func TestOpenPipelineList_AppliesSavedSortMode(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.repository.(*mockRepository).settings.PipelineSortMode = domain.PipelineSortLastRun
	m.pipelineListView.SetPipelines([]domain.Pipeline{testPipeline()})

	m, _ = m.openPipelineList()

	if m.pipelineListView.SortMode() != domain.PipelineSortLastRun {
		t.Errorf("expected saved sort mode applied, got %s", m.pipelineListView.SortMode())
	}
}

func TestEnterPipelineRuns_MarksPipelineViewed(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	repo := m.repository.(*mockRepository)

	m.enterPipelineRuns(testPipeline())

	if _, ok := repo.pipelineLastViewed[testPipeline().FavoriteKey()]; !ok {
		t.Error("expected pipeline marked as viewed")
	}
}

func TestSortShortcut_ShownOnceInTopBar(t *testing.T) {
	shortcuts := NewCommandRegistry().GetContextualShortcuts(ViewPipelineList, false)

	count := 0
	for _, shortcut := range shortcuts {
		if contains(shortcut, "Sort") {
			count++
		}
	}
	if count != 1 || !containsShortcut(shortcuts, "<0-5>") {
		t.Errorf("expected a single <0-5> sort hint, got %v", shortcuts)
	}
}

func containsShortcut(shortcuts []string, description string) bool {
	for _, shortcut := range shortcuts {
		if contains(shortcut, description) {
			return true
		}
	}
	return false
}

func TestRefreshWithDetailsPaneOpen_RequestsRunsAgain(t *testing.T) {
	pipeline := testPipeline()
	run := testRun(domain.PipelineStatusSucceeded)
	provider := &mockPipelineProvider{pipelines: []domain.Pipeline{pipeline}, runs: []domain.PipelineRun{run}}
	m := createPipelineTestModel(provider)
	m.state = ViewPipelineList
	m.pipelineListView.SetSortMode(domain.PipelineSortPipeline)
	m.pipelineListView.SetPipelines([]domain.Pipeline{pipeline})
	m, cmd := handleTogglePreviewKey(m)
	m, _ = m.handlePipelineRunsLoaded(cmd().(PipelineRunsLoadedMsg))

	m, cmd = handleRefreshKey(m)
	m, _ = m.handlePipelinesLoaded(cmd().(PipelinesLoadedMsg))

	// The reload must already have asked for the runs again, so none are left to request.
	if requests := m.pipelineListView.PendingRunRequests(); len(requests) != 0 {
		t.Fatalf("expected the refreshed list to fetch the selected pipeline's runs again, still pending %+v", requests)
	}
}

func TestEnterOnRunRow_OpensTheRun(t *testing.T) {
	pipeline := testPipeline()
	run := testRun(domain.PipelineStatusSucceeded)
	m := createPipelineTestModel(&mockPipelineProvider{run: &run})
	m.state = ViewPipelineList
	m.pipelineListView.SetSortMode(domain.PipelineSortPipeline)
	m.pipelineListView.SetPipelines([]domain.Pipeline{pipeline})
	m, _ = handleFoldKey(m)
	m.pipelineListView.SetPreviewRuns(pipeline.FavoriteKey(), []domain.PipelineRun{run})
	m, _ = handleDownKey(m)

	m, _ = handleEnterKey(m)

	if m.state != ViewPipelineRun || m.pipelineRunView.GetRun().ID != run.ID {
		t.Errorf("expected run %d to open, got state %v", run.ID, m.state)
	}
}

func TestFavoritesOnlyKey_TogglesTheListFilter(t *testing.T) {
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.state = ViewPipelineList

	newModel, _, handled := m.commandRegistry.HandleKey(m, keyFavoritesOnly)

	if !handled || !newModel.pipelineListView.IsFavoritesOnly() {
		t.Error("expected F to show favorites only")
	}
}

func TestRunsFailure_StopsLoadingAndReportsTheError(t *testing.T) {
	pipeline := testPipeline()
	m := createPipelineTestModel(&mockPipelineProvider{})
	m.state = ViewPipelineList
	m.pipelineListView.SetSortMode(domain.PipelineSortPipeline)
	m.pipelineListView.SetPipelines([]domain.Pipeline{pipeline})
	m.pipelineListView.TogglePreview()
	m.pipelineListView.PendingRunRequests()

	updated, _ := m.Update(PipelineRunsFailedMsg{pipelineKey: pipeline.FavoriteKey(), err: errors.New("boom")})
	m = updated.(Model)

	if m.statusBar.Message() != "boom" {
		t.Errorf("expected the error in the status bar, got %q", m.statusBar.Message())
	}
	if len(m.pipelineListView.PendingRunRequests()) != 1 {
		t.Error("expected the runs to be requested again")
	}
}
