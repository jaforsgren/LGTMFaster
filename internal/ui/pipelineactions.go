package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

type PipelineRetriedMsg struct {
	run   domain.PipelineRun
	scope domain.PipelineRetryScope
	stage domain.PipelineStage
}

type PipelineTriggerOptionsLoadedMsg struct {
	pipelineKey string
	options     domain.PipelineTriggerOptions
}

type PipelineTriggeredMsg struct {
	run    domain.PipelineRun
	branch string
}

// retryTarget is the run a retry applies to in the current view, with the selected stage in the
// run view.
func (m Model) retryTarget() (*domain.PipelineRun, *domain.PipelineStage) {
	switch m.state {
	case ViewPipelineRun:
		return m.pipelineRunView.GetRun(), m.pipelineRunView.GetSelectedStage()
	case ViewPipelineLog:
		return m.pipelineLogView.GetRun(), nil
	case ViewPipelineRuns:
		return m.pipelineRunsView.GetSelectedRun(), nil
	}
	return nil, nil
}

func (m Model) beginPipelineRetry() (Model, tea.Cmd) {
	run, stage := m.retryTarget()
	if run == nil {
		m.statusBar.SetMessage("No run selected to retry", true)
		return m, nil
	}
	m.pipelineRetryView.Activate(*run, stage)
	return m, nil
}

func (m Model) handlePipelineRetryKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case keyUp, keyUpVim:
		m.pipelineRetryView.MoveUp()
	case keyDown, keyDownVim:
		m.pipelineRetryView.MoveDown()
	case keyEnter:
		return m, m.submitPipelineRetry()
	case keyEsc:
		m.pipelineRetryView.Deactivate()
	}
	return m, nil
}

func (m Model) submitPipelineRetry() tea.Cmd {
	retry := m.pipelineRetryView.GetRetry()
	run := m.pipelineRetryView.GetRun()
	m.pipelineRetryView.Deactivate()

	provider, err := m.getPipelineProvider(run.PATID)
	if err != nil {
		return func() tea.Msg {
			return ErrorMsg{err: err}
		}
	}

	ctx := m.ctx
	m.statusBar.SetMessage(views.RetryScopeLabel(retry.Scope, retry.Stage)+"...", false)
	logger.Log("UI: Retrying %s of run %d", retry.Scope, run.ID)
	return func() tea.Msg {
		retried, err := provider.RetryPipelineRun(ctx, retry)
		if err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to retry run %s: %w", run.Number, err)}
		}
		tagPipelineRun(retried, run.PATID, run.ProviderType)
		return PipelineRetriedMsg{run: *retried, scope: retry.Scope, stage: retry.Stage}
	}
}

// handlePipelineRetried opens a rerun's new run; a retry within the run refreshes it where it is
// shown, and its logs are followed again.
func (m Model) handlePipelineRetried(msg PipelineRetriedMsg) (Model, tea.Cmd) {
	status := clearStatusAfterDelay(4 * time.Second)
	if msg.scope == domain.PipelineRetryRerun {
		m.statusBar.SetMessage(fmt.Sprintf("Started run %s", msg.run.Number), false)
		opened, cmd := m.enterPipelineRun(msg.run)
		return opened, tea.Batch(cmd, status)
	}

	m.statusBar.SetMessage(fmt.Sprintf("%s of run %s", views.RetryScopeLabel(msg.scope, msg.stage), msg.run.Number), false)
	switch m.state {
	case ViewPipelineLog:
		m.pipelineLogView.Open(msg.run, "")
		return m, tea.Batch(m.loadPipelineLogs(msg.run), m.startPipelineLogFollow(), status)
	case ViewPipelineRun:
		return m, tea.Batch(m.loadPipelineRun(msg.run), m.startPipelineWatch(), status)
	case ViewPipelineRuns:
		if pipeline := m.pipelineRunsView.GetPipeline(); pipeline != nil {
			return m, tea.Batch(m.loadPipelineRuns(*pipeline), status)
		}
	}
	return m, status
}

// triggerTarget is the pipeline the current view would start: the selected one in the list, the
// shown one in its runs.
func (m Model) triggerTarget() *domain.Pipeline {
	switch m.state {
	case ViewPipelineList:
		return m.pipelineListView.GetSelectedPipeline()
	case ViewPipelineRuns:
		return m.pipelineRunsView.GetPipeline()
	}
	return nil
}

func (m Model) beginPipelineTrigger() (Model, tea.Cmd) {
	pipeline := m.triggerTarget()
	if pipeline == nil {
		m.statusBar.SetMessage("No pipeline selected to run", true)
		return m, nil
	}
	provider, err := m.getPipelineProvider(pipeline.PATID)
	if err != nil {
		m.statusBar.SetMessage(err.Error(), true)
		return m, nil
	}

	m.pipelineTriggerView.Activate(*pipeline)
	ctx, selected := m.ctx, *pipeline
	return m, func() tea.Msg {
		options, err := provider.GetPipelineTriggerOptions(ctx, selected.Identifier())
		if err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to load the branches and parameters of %s: %w", selected.Name, err)}
		}
		return PipelineTriggerOptionsLoadedMsg{pipelineKey: selected.FavoriteKey(), options: *options}
	}
}

func (m Model) handlePipelineTriggerOptionsLoaded(msg PipelineTriggerOptionsLoadedMsg) (Model, tea.Cmd) {
	if m.pipelineTriggerView.IsActive() && m.pipelineTriggerView.GetPipeline().FavoriteKey() == msg.pipelineKey {
		m.pipelineTriggerView.SetOptions(msg.options)
	}
	return m, nil
}

func (m Model) handlePipelineTriggerKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	action, cmd := m.pipelineTriggerView.HandleKey(msg)
	switch action {
	case views.PipelineTriggerActionSubmit:
		return m.submitPipelineTrigger()
	case views.PipelineTriggerActionCancel:
		m.pipelineTriggerView.Deactivate()
		return m, nil
	}
	return m, cmd
}

func (m Model) submitPipelineTrigger() (Model, tea.Cmd) {
	trigger, err := m.pipelineTriggerView.Trigger()
	if err != nil {
		m.statusBar.SetMessage(err.Error(), true)
		return m, nil
	}
	pipeline := m.pipelineTriggerView.GetPipeline()
	provider, err := m.getPipelineProvider(pipeline.PATID)
	if err != nil {
		m.statusBar.SetMessage(err.Error(), true)
		return m, nil
	}

	ctx := m.ctx
	m.statusBar.SetMessage(fmt.Sprintf("Starting %s on %s...", pipeline.Name, trigger.Branch), false)
	logger.Log("UI: Triggering pipeline %d on %s", pipeline.ID, trigger.Branch)
	return m, func() tea.Msg {
		run, err := provider.TriggerPipeline(ctx, trigger)
		if err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to start %s: %w", pipeline.Name, err)}
		}
		tagPipelineRun(run, pipeline.PATID, pipeline.ProviderType)
		return PipelineTriggeredMsg{run: *run, branch: trigger.Branch}
	}
}

// handlePipelineTriggered opens the new run, which is followed on screen while it runs.
func (m Model) handlePipelineTriggered(msg PipelineTriggeredMsg) (Model, tea.Cmd) {
	m.pipelineTriggerView.Deactivate()
	m.statusBar.SetMessage(fmt.Sprintf("Started run %s on %s", msg.run.Number, msg.branch), false)
	opened, cmd := m.enterPipelineRun(msg.run)
	return opened, tea.Batch(cmd, clearStatusAfterDelay(4*time.Second))
}
