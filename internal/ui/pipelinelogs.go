package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/ui/components"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

// pipelineLogFollowInterval is how often a followed log asks for new lines, independent of the
// auto-refresh setting, since following is asked for explicitly.
const pipelineLogFollowInterval = 3 * time.Second

// PipelineLogsLoadedMsg carries the refreshed run with its step logs, so following knows when the
// run has finished.
type PipelineLogsLoadedMsg struct {
	run  domain.PipelineRun
	logs []domain.PipelineLog
}

type PipelineLogLinesLoadedMsg struct {
	run     domain.PipelineRunIdentifier
	patID   string
	request views.LogLinesRequest
	lines   []string
}

type PipelineLogLinesFailedMsg struct {
	run   domain.PipelineRunIdentifier
	logID int
	err   error
}

// PipelineLogTickMsg carries the follow generation it was scheduled for; ticks of an older one
// are dropped.
type PipelineLogTickMsg struct {
	watchID int
}

// openPipelineLogs shows the logs of the inspected run, opening on the selected stage.
func (m Model) openPipelineLogs() (Model, tea.Cmd) {
	run := m.pipelineRunView.GetRun()
	if run == nil {
		m.statusBar.SetMessage("No run selected", true)
		return m, nil
	}
	stage := ""
	if selected := m.pipelineRunView.GetSelectedStage(); selected != nil {
		stage = selected.Name
	}

	logger.Log("UI: Opening logs of run %d", run.ID)
	m.state = ViewPipelineLog
	m.pipelineLogView.Open(*run, stage)
	m.topBar.SetView(components.ViewNamePipelineLog)
	m.setPipelineRunContext(*run)
	m.updateShortcuts()
	return m, tea.Batch(m.loadPipelineLogs(*run), m.startPipelineLogFollow())
}

// leavePipelineLogs goes back to the run, stopping the follow loop.
func (m Model) leavePipelineLogs() (Model, tea.Cmd) {
	logger.Log("UI: Navigating back from Pipeline Logs to Pipeline Run")
	m.pipelineLogWatchID++
	m.state = ViewPipelineRun
	m.topBar.SetView(components.ViewNamePipelineRun)
	m.updateShortcuts()
	run := m.pipelineRunView.GetRun()
	if run == nil {
		return m, nil
	}
	return m, m.loadPipelineRun(*run)
}

func (m Model) loadPipelineLogs(run domain.PipelineRun) tea.Cmd {
	ctx := m.ctx
	provider, providerErr := m.getPipelineProvider(run.PATID)
	return func() tea.Msg {
		if providerErr != nil {
			return ErrorMsg{err: providerErr}
		}

		refreshed, err := provider.GetPipelineRun(ctx, run.Identifier())
		if err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to load run %s: %w", run.Number, err)}
		}
		logs, err := provider.ListPipelineRunLogs(ctx, run.Identifier())
		if err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to load logs of run %s: %w", run.Number, err)}
		}
		tagPipelineRun(refreshed, run.PATID, run.ProviderType)
		return PipelineLogsLoadedMsg{run: *refreshed, logs: logs}
	}
}

// loadPendingLogLines reads what the selected step's log is missing.
func (m Model) loadPendingLogLines() tea.Cmd {
	run := m.pipelineLogView.GetRun()
	if run == nil {
		return nil
	}
	request, needed := m.pipelineLogView.PendingLinesRequest()
	if !needed {
		return nil
	}

	ctx := m.ctx
	identifier, patID := run.Identifier(), run.PATID
	provider, providerErr := m.getPipelineProvider(patID)
	return func() tea.Msg {
		if providerErr != nil {
			return PipelineLogLinesFailedMsg{run: identifier, logID: request.Log.ID, err: providerErr}
		}
		lines, err := provider.GetPipelineLogLines(ctx, identifier, request.Log.ID, request.Offset)
		if err != nil {
			return PipelineLogLinesFailedMsg{run: identifier, logID: request.Log.ID,
				err: fmt.Errorf("failed to read the log of %s: %w", request.Log.Path(), err)}
		}
		return PipelineLogLinesLoadedMsg{run: identifier, patID: patID, request: request, lines: lines}
	}
}

func (m Model) isShownPipelineLog(identifier domain.PipelineRunIdentifier, patID string) bool {
	run := m.pipelineLogView.GetRun()
	return run != nil && run.Identifier() == identifier && run.PATID == patID
}

func (m Model) handlePipelineLogsLoaded(msg PipelineLogsLoadedMsg) (Model, tea.Cmd) {
	if !m.isShownPipelineLog(msg.run.Identifier(), msg.run.PATID) {
		return m, nil
	}
	m.pipelineLogView.SetRun(msg.run)
	m.pipelineLogView.SetLogs(msg.logs)
	m.pipelineRunView.SetRun(msg.run)
	if m.state == ViewPipelineLog {
		m.setPipelineRunContext(msg.run)
	}
	m.topBar.SetLastRefresh(time.Now())
	return m, m.loadPendingLogLines()
}

func (m Model) handlePipelineLogLinesLoaded(msg PipelineLogLinesLoadedMsg) (Model, tea.Cmd) {
	if !m.isShownPipelineLog(msg.run, msg.patID) {
		return m, nil
	}
	m.pipelineLogView.AppendLines(msg.request, msg.lines)
	return m, nil
}

func (m Model) handlePipelineLogLinesFailed(msg PipelineLogLinesFailedMsg) (Model, tea.Cmd) {
	if run := m.pipelineLogView.GetRun(); run != nil && run.Identifier() == msg.run {
		m.pipelineLogView.LinesFailed(msg.logID)
	}
	m.statusBar.SetMessage(msg.err.Error(), true)
	return m, nil
}

// startPipelineLogFollow restarts the follow loop, which runs while the log view follows an
// unfinished run. Bumping the generation stops the previous loop.
func (m *Model) startPipelineLogFollow() tea.Cmd {
	m.pipelineLogWatchID++
	run := m.pipelineLogView.GetRun()
	if !m.pipelineLogView.IsFollowing() || run == nil || run.Status.IsFinished() {
		return nil
	}
	return schedulePipelineLogTick(m.pipelineLogWatchID)
}

func schedulePipelineLogTick(watchID int) tea.Cmd {
	return tea.Tick(pipelineLogFollowInterval, func(time.Time) tea.Msg {
		return PipelineLogTickMsg{watchID: watchID}
	})
}

// handlePipelineLogTick reloads the followed run; the loop ends once the run has finished, after
// this last load has read its final lines.
func (m Model) handlePipelineLogTick(msg PipelineLogTickMsg) (Model, tea.Cmd) {
	run := m.pipelineLogView.GetRun()
	if msg.watchID != m.pipelineLogWatchID || m.state != ViewPipelineLog || run == nil || !m.pipelineLogView.IsFollowing() {
		return m, nil
	}
	load := m.loadPipelineLogs(*run)
	if run.Status.IsFinished() {
		return m, load
	}
	return m, tea.Batch(load, schedulePipelineLogTick(msg.watchID))
}

func (m Model) toggleLogFollow() (Model, tea.Cmd) {
	m.pipelineLogView.ToggleFollow()
	if !m.pipelineLogView.IsFollowing() {
		m.pipelineLogWatchID++
		m.statusBar.SetMessage("Stopped following the log", false)
		return m, clearStatusAfterDelay(4 * time.Second)
	}
	m.statusBar.SetMessage("Following the log", false)
	return m, tea.Batch(m.startPipelineLogFollow(), m.loadPendingLogLines(), clearStatusAfterDelay(4*time.Second))
}

func (m Model) refreshPipelineLogs() (Model, tea.Cmd) {
	run := m.pipelineLogView.GetRun()
	if run == nil {
		return m, nil
	}
	m.pipelineLogView.ReloadLines()
	return m, m.loadPipelineLogs(*run)
}
