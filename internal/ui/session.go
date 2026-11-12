package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
)

const noSessionMessage = "No previous session to resume"

var sessionViews = map[ViewState]domain.SessionView{
	ViewPATs:         domain.SessionViewPATs,
	ViewPRList:       domain.SessionViewPRList,
	ViewPRInspect:    domain.SessionViewPRInspect,
	ViewPipelineList: domain.SessionViewPipelines,
	ViewPipelineRuns: domain.SessionViewPipelineRuns,
	ViewPipelineRun:  domain.SessionViewPipelineRun,
	ViewDashboard:    domain.SessionViewDashboard,
}

// captureSession records the view, its selection and which details panes are open.
func (m Model) captureSession() domain.Session {
	session := domain.Session{
		View:                 sessionViews[m.state],
		PRListPaneOpen:       m.prListView.IsPreviewOpen(),
		PipelineListPaneOpen: m.pipelineListView.IsPreviewOpen(),
		PipelineRunsPaneOpen: m.pipelineRunsView.IsPreviewOpen(),
	}

	if pr := m.prInspect.GetPR(); m.state == ViewPRInspect && pr != nil {
		session.PRKey = pr.Key()
	} else if pr := m.prListView.GetSelectedPR(); pr != nil {
		session.PRKey = pr.Key()
	}

	if pipeline := m.pipelineRunsView.GetPipeline(); pipeline != nil && (m.state == ViewPipelineRuns || m.state == ViewPipelineRun) {
		session.PipelineKey = pipeline.FavoriteKey()
	} else if node := m.pipelineListView.GetSelectedNode(); node != nil {
		session.PipelineKey = node.Key
	}

	if run := m.pipelineRunView.GetRun(); run != nil && m.state == ViewPipelineRun {
		session.RunKey = run.FavoriteKey()
	} else if run := m.pipelineRunsView.GetSelectedRun(); run != nil {
		session.RunKey = run.FavoriteKey()
	}
	return session
}

func (m Model) saveSession() {
	if err := m.repository.SaveSession(m.captureSession()); err != nil {
		logger.LogError("SAVE_SESSION", "session", err)
	}
}

// quit saves where the user is, like postOffice, so the next start resumes there.
func (m Model) quit() (Model, tea.Cmd) {
	m.saveSession()
	return m, tea.Quit
}

// loadSession reopens the saved details panes straight away and keeps the rest of the session to
// resume once the PRs or pipelines it points at have loaded.
func (m Model) loadSession() Model {
	session, err := m.repository.GetSession()
	if err != nil {
		logger.LogError("LOAD_SESSION", "session", err)
		return m
	}

	panes := []struct {
		open bool
		list splitList
	}{
		{session.PRListPaneOpen, m.prListView},
		{session.PipelineListPaneOpen, m.pipelineListView},
		{session.PipelineRunsPaneOpen, m.pipelineRunsView},
	}
	for _, pane := range panes {
		if pane.open != pane.list.IsPreviewOpen() {
			pane.list.TogglePreview()
		}
	}

	if session.View != "" && session.View != domain.SessionViewPATs {
		m.lastSession = &session
	}
	return m
}

// handleSessionCommand resumes the session saved when the app last quit, like postOffice's
// :session; the app itself always starts on the dashboard.
func handleSessionCommand(m Model, args []string) (Model, tea.Cmd) {
	if m.lastSession == nil {
		m.statusBar.SetMessage(noSessionMessage, true)
		return m, nil
	}
	session := *m.lastSession
	m.resume = &session

	if resumed, cmd, ok := m.startResumedView(); ok {
		if !resumed.pipelineListView.HasPipelines() {
			return resumed, cmd
		}
		resumed, resumeCmd := resumed.resumePipelines()
		return resumed, tea.Batch(cmd, resumeCmd)
	}
	if len(m.prListView.AllPRs()) == 0 {
		return m, m.loadPRsWithCache()
	}
	return m.resumePRs(true)
}

// startResumedView opens the view family of the session being resumed, once the PATs are loaded.
func (m Model) startResumedView() (Model, tea.Cmd, bool) {
	if m.resume == nil || !m.resume.IsPipelineView() {
		return m, nil, false
	}
	model, cmd := m.openPipelineList()
	return model, cmd, true
}

// resumePRs selects the session's PR as soon as it is listed, and reopens it if it was inspected.
// Once every PAT has loaded without finding it, the resume is given up.
func (m Model) resumePRs(loadingDone bool) (Model, tea.Cmd) {
	if m.resume == nil || m.resume.IsPipelineView() {
		return m, nil
	}
	if m.resume.View == domain.SessionViewDashboard {
		m.resume = nil
		return m.openDashboard()
	}
	if !m.prListView.SelectPR(m.resume.PRKey) {
		if loadingDone {
			m.resume = nil
		}
		return m, nil
	}

	inspect := m.resume.View == domain.SessionViewPRInspect
	m.resume = nil
	m = m.showPRList()
	if inspect {
		return handleEnterKey(m)
	}
	return m, nil
}

// resumePipelines selects the session's pipeline and, for the runs and run views, opens it.
func (m Model) resumePipelines() (Model, tea.Cmd) {
	if m.resume == nil || !m.resume.IsPipelineView() {
		return m, m.loadPipelinePreview()
	}
	if !m.pipelineListView.SelectNode(m.resume.PipelineKey) {
		m.resume = nil
		return m, nil
	}

	pipeline := m.pipelineListView.GetSelectedPipeline()
	if m.resume.View == domain.SessionViewPipelines || pipeline == nil {
		m.resume = nil
		return m, m.loadPipelinePreview()
	}
	return m.enterPipelineRuns(*pipeline)
}

// resumePipelineRuns selects the session's run once the runs are listed, and reopens it if it was
// open.
func (m Model) resumePipelineRuns() (Model, tea.Cmd) {
	if m.resume == nil || m.state != ViewPipelineRuns {
		return m, m.loadPipelinePreview()
	}

	found := m.pipelineRunsView.SelectRun(m.resume.RunKey)
	openRun := m.resume.View == domain.SessionViewPipelineRun
	m.resume = nil
	if run := m.pipelineRunsView.GetSelectedRun(); found && openRun && run != nil {
		return m.enterPipelineRun(*run)
	}
	return m, m.loadPipelinePreview()
}
