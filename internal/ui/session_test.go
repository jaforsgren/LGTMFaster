package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func sessionTestPR() domain.PullRequest {
	return domain.PullRequest{Number: 42, PATID: testPATID, Repository: domain.Repo{FullName: "owner/repo"}}
}

func quitsWith(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestCtrlC_QuitsFromEveryViewAndSavesTheSession(t *testing.T) {
	for _, state := range allViewStates {
		repository := &mockRepository{pats: map[string]*domain.PAT{}}
		m := sizedTestModel(t, layoutTestWidth, layoutTestHeight)
		m.repository = repository
		m.state = state

		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

		if !quitsWith(cmd) {
			t.Errorf("view %v: expected ctrl+c to quit", state)
		}
		if repository.session.View != sessionViews[updated.(Model).state] {
			t.Errorf("view %v: expected the session to be saved as %q, got %q", state, sessionViews[state], repository.session.View)
		}
	}
}

func TestCaptureSession_RecordsInspectedPRAndOpenPanes(t *testing.T) {
	m := sizedTestModel(t, layoutTestWidth, layoutTestHeight)
	pr := sessionTestPR()
	m.prListView.SetPRs([]domain.PullRequest{pr})
	m.prListView.TogglePreview()
	m.state = ViewPRInspect
	m.prInspect.SetPR(&pr)

	session := m.captureSession()

	if session.View != domain.SessionViewPRInspect || session.PRKey != pr.Key() {
		t.Errorf("expected the inspected PR in the session, got %+v", session)
	}
	if !session.PRListPaneOpen || session.PipelineListPaneOpen {
		t.Errorf("expected only the PR list pane to be recorded open, got %+v", session)
	}
}

func TestLoadSession_ReopensPanesAndWaitsToResume(t *testing.T) {
	repository := &mockRepository{
		pats:    map[string]*domain.PAT{},
		session: domain.Session{View: domain.SessionViewPRList, PRKey: "key", PipelineRunsPaneOpen: true},
	}

	m := NewModel(repository)

	if !m.pipelineRunsView.IsPreviewOpen() || m.prListView.IsPreviewOpen() {
		t.Error("expected only the saved runs pane to be reopened")
	}
	if m.resume == nil || m.resume.PRKey != "key" {
		t.Errorf("expected the session to wait for its PR, got %+v", m.resume)
	}
}

func TestLoadSession_PATsViewHasNothingToResume(t *testing.T) {
	m := NewModel(&mockRepository{pats: map[string]*domain.PAT{}, session: domain.Session{View: domain.SessionViewPATs}})

	if m.resume != nil {
		t.Errorf("expected nothing to resume, got %+v", m.resume)
	}
}

func TestResumePRs_ReopensTheInspectedPR(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRList
	pr := sessionTestPR()
	m.resume = &domain.Session{View: domain.SessionViewPRInspect, PRKey: pr.Key()}

	m, _ = m.resumePRs(false)
	if m.resume == nil {
		t.Fatal("expected to keep waiting while the PR is not listed yet")
	}

	other := pr
	other.Number = 1
	m.prListView.SetPRs([]domain.PullRequest{other, pr})
	m, _ = m.resumePRs(false)

	if m.resume != nil {
		t.Error("expected the resume to be done")
	}
	if m.state != ViewPRInspect || m.prListView.GetSelectedPR().Number != pr.Number {
		t.Errorf("expected #%d to be inspected, got state %v", pr.Number, m.state)
	}
}

func TestResumePRs_GivesUpWhenLoadingEndsWithoutThePR(t *testing.T) {
	m := createTestModel()
	m.state = ViewPRList
	m.resume = &domain.Session{View: domain.SessionViewPRInspect, PRKey: "missing"}

	m, _ = m.resumePRs(true)

	if m.resume != nil || m.state != ViewPRList {
		t.Errorf("expected to give up and stay in the list, got resume %+v in %v", m.resume, m.state)
	}
}

func TestResumePipelines_ReopensTheRunThroughItsPipeline(t *testing.T) {
	pipeline := testPipeline()
	run := testRun(domain.PipelineStatusSucceeded)
	provider := &mockPipelineProvider{pipelines: []domain.Pipeline{pipeline}, runs: []domain.PipelineRun{run}, run: &run}
	m := createPipelineTestModel(provider)
	m.resume = &domain.Session{
		View:        domain.SessionViewPipelineRun,
		PipelineKey: pipeline.FavoriteKey(),
		RunKey:      run.FavoriteKey(),
	}

	m, _, ok := m.startResumedView()
	if !ok {
		t.Fatal("expected the pipeline session to open pipeline mode")
	}
	m, _ = m.handlePipelinesLoaded(PipelinesLoadedMsg{pipelines: []domain.Pipeline{pipeline}})
	if m.state != ViewPipelineRuns {
		t.Fatalf("expected the runs of the saved pipeline, got %v", m.state)
	}

	m, _ = m.handlePipelineRunsLoaded(PipelineRunsLoadedMsg{pipeline: pipeline.Identifier(), patID: pipeline.PATID, runs: []domain.PipelineRun{run}})

	if m.state != ViewPipelineRun || m.pipelineRunView.GetRun().ID != run.ID {
		t.Errorf("expected the saved run to be reopened, got %v", m.state)
	}
	if m.resume != nil {
		t.Error("expected the resume to be done")
	}
}
