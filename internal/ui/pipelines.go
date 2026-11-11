package ui

import (
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/ui/components"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

const pipelineWatchInterval = 10 * time.Second

var errNoPipelineProviders = errors.New("no selected PAT supports pipelines (Azure DevOps only)")

type PipelinesLoadedMsg struct {
	pipelines      []domain.Pipeline
	failedPATCount int
}

type PipelineRunsLoadedMsg struct {
	pipeline domain.PipelineIdentifier
	patID    string
	runs     []domain.PipelineRun
}

type PipelineRunLoadedMsg struct {
	run    domain.PipelineRun
	stages []domain.PipelineStage
}

// PipelineWatchTickMsg carries the watch generation it was scheduled for, so ticks from a
// view the user already left are dropped instead of spawning parallel refresh loops.
type PipelineWatchTickMsg struct {
	watchID int
}

type StageApprovalSubmittedMsg struct {
	run       domain.PipelineRun
	stageName string
	decision  domain.ApprovalDecision
}

func (m Model) pipelineProviders() map[string]domain.Provider {
	providers := make(map[string]domain.Provider)
	for patID, provider := range m.providers {
		if _, ok := provider.(domain.PipelineProvider); ok {
			providers[patID] = provider
		}
	}
	if len(m.providers) == 0 && m.provider != nil {
		if _, ok := m.provider.(domain.PipelineProvider); ok {
			// Single active PAT mode has no PAT ID; getProviderForPAT falls back to m.provider for "".
			providers[""] = m.provider
		}
	}
	return providers
}

func (m Model) getPipelineProvider(patID string) (domain.PipelineProvider, error) {
	provider := m.getProviderForPAT(patID)
	if provider == nil {
		return nil, fmt.Errorf("no provider available for PAT %q", patID)
	}
	pipelineProvider, ok := provider.(domain.PipelineProvider)
	if !ok {
		return nil, fmt.Errorf("provider %s does not support pipelines", provider.GetType())
	}
	return pipelineProvider, nil
}

func (m Model) openPipelineList() (Model, tea.Cmd) {
	if len(m.pipelineProviders()) == 0 {
		m.statusBar.SetMessage(errNoPipelineProviders.Error(), true)
		return m, nil
	}

	m.state = ViewPipelineList
	m.pipelineWatchID++
	m.topBar.SetView(components.ViewNamePipelines)
	m.topBar.SetPipelineContext("", "", "")
	m.topBar.SetPipelineCount(m.pipelineListView.PipelineCount())
	m.updateShortcuts()

	if m.pipelineListView.HasPipelines() {
		return m, nil
	}

	m.statusBar.SetMessage("Loading pipelines...", false)
	return m, m.loadPipelines()
}

func (m Model) loadPipelines() tea.Cmd {
	providers := m.pipelineProviders()
	if len(providers) == 0 {
		return func() tea.Msg {
			return ErrorMsg{err: errNoPipelineProviders}
		}
	}

	ctx := m.ctx
	return func() tea.Msg {
		type pipelineResult struct {
			pipelines []domain.Pipeline
			err       error
		}

		results := make(chan pipelineResult, len(providers))
		for patID, provider := range providers {
			go func(patID string, provider domain.Provider) {
				pipelines, err := provider.(domain.PipelineProvider).ListPipelines(ctx)
				for i := range pipelines {
					tagPipeline(&pipelines[i], patID, provider.GetType())
				}
				if err != nil {
					logger.LogError("LOAD_PIPELINES", patID, err)
				}
				results <- pipelineResult{pipelines: pipelines, err: err}
			}(patID, provider)
		}

		msg := PipelinesLoadedMsg{}
		for range providers {
			result := <-results
			if result.err != nil {
				msg.failedPATCount++
			}
			msg.pipelines = append(msg.pipelines, result.pipelines...)
		}
		return msg
	}
}

func (m Model) loadPipelineRuns(pipeline domain.Pipeline) tea.Cmd {
	ctx := m.ctx
	provider, providerErr := m.getPipelineProvider(pipeline.PATID)
	return func() tea.Msg {
		if providerErr != nil {
			return ErrorMsg{err: providerErr}
		}

		runs, err := provider.ListPipelineRuns(ctx, pipeline.Identifier())
		if err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to load runs for %s: %w", pipeline.Name, err)}
		}
		for i := range runs {
			tagPipelineRun(&runs[i], pipeline.PATID, pipeline.ProviderType)
		}
		return PipelineRunsLoadedMsg{pipeline: pipeline.Identifier(), patID: pipeline.PATID, runs: runs}
	}
}

func (m Model) loadPipelineRun(run domain.PipelineRun) tea.Cmd {
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
		stages, err := provider.GetPipelineRunStages(ctx, run.Identifier())
		if err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to load stages for run %s: %w", run.Number, err)}
		}

		tagPipelineRun(refreshed, run.PATID, run.ProviderType)
		return PipelineRunLoadedMsg{run: *refreshed, stages: stages}
	}
}

func tagPipeline(pipeline *domain.Pipeline, patID string, providerType domain.ProviderType) {
	pipeline.PATID = patID
	pipeline.ProviderType = providerType
	if pipeline.LatestRun != nil {
		tagPipelineRun(pipeline.LatestRun, patID, providerType)
	}
}

func tagPipelineRun(run *domain.PipelineRun, patID string, providerType domain.ProviderType) {
	run.PATID = patID
	run.ProviderType = providerType
}

func (m *Model) startPipelineWatch() tea.Cmd {
	m.pipelineWatchID++
	return schedulePipelineWatchTick(m.pipelineWatchID)
}

func schedulePipelineWatchTick(watchID int) tea.Cmd {
	return tea.Tick(pipelineWatchInterval, func(time.Time) tea.Msg {
		return PipelineWatchTickMsg{watchID: watchID}
	})
}

func (m Model) enterPipelineRuns(pipeline domain.Pipeline) (Model, tea.Cmd) {
	m.pipelineRunsView.SetPipeline(pipeline)
	return m.showPipelineRuns()
}

func (m Model) showPipelineRuns() (Model, tea.Cmd) {
	pipeline := m.pipelineRunsView.GetPipeline()
	if pipeline == nil {
		return m, nil
	}

	m.state = ViewPipelineRuns
	m.topBar.SetView(components.ViewNamePipelineRuns)
	m.topBar.SetPipelineContext(pipelineRepoLabel(*pipeline), views.PipelineDisplayName(*pipeline), "")
	m.updateShortcuts()

	watch := m.startPipelineWatch()
	return m, tea.Batch(m.loadPipelineRuns(*pipeline), watch)
}

func (m Model) enterPipelineRun(run domain.PipelineRun) (Model, tea.Cmd) {
	m.state = ViewPipelineRun
	m.pipelineRunView.SetRun(run)
	m.topBar.SetView(components.ViewNamePipelineRun)
	m.setPipelineRunContext(run)
	m.updateShortcuts()

	watch := m.startPipelineWatch()
	return m, tea.Batch(m.loadPipelineRun(run), watch)
}

func (m Model) setPipelineRunContext(run domain.PipelineRun) {
	repo, pipelineName := run.Project, run.PipelineName
	if pipeline := m.pipelineRunsView.GetPipeline(); pipeline != nil {
		repo = pipelineRepoLabel(*pipeline)
		pipelineName = views.PipelineDisplayName(*pipeline)
	}
	runLabel := fmt.Sprintf("%s [%s]", run.Number, views.PipelineStatusLabel(run.Status))
	m.topBar.SetPipelineContext(repo, pipelineName, runLabel)
}

func pipelineRepoLabel(pipeline domain.Pipeline) string {
	return pipeline.Project + "/" + pipeline.Repository
}

func (m Model) handlePipelinesLoaded(msg PipelinesLoadedMsg) (Model, tea.Cmd) {
	m.pipelineListView.SetPipelines(msg.pipelines)
	m.topBar.SetPipelineCount(m.pipelineListView.PipelineCount())

	if msg.failedPATCount > 0 {
		m.statusBar.SetMessage(fmt.Sprintf("Loaded %d pipelines (%d PAT(s) failed)", len(msg.pipelines), msg.failedPATCount), true)
	} else {
		m.statusBar.SetMessage(fmt.Sprintf("Loaded %d pipelines", len(msg.pipelines)), false)
	}
	return m, clearStatusAfterDelay(4 * time.Second)
}

func (m Model) handlePipelineRunsLoaded(msg PipelineRunsLoadedMsg) (Model, tea.Cmd) {
	current := m.pipelineRunsView.GetPipeline()
	if current == nil || current.Identifier() != msg.pipeline || current.PATID != msg.patID {
		return m, nil
	}
	m.pipelineRunsView.SetRuns(msg.runs)
	return m, nil
}

func (m Model) handlePipelineRunLoaded(msg PipelineRunLoadedMsg) (Model, tea.Cmd) {
	current := m.pipelineRunView.GetRun()
	if current == nil || current.Identifier() != msg.run.Identifier() || current.PATID != msg.run.PATID {
		return m, nil
	}
	m.pipelineRunView.SetRun(msg.run)
	m.pipelineRunView.SetStages(msg.stages)
	if m.state == ViewPipelineRun {
		m.setPipelineRunContext(msg.run)
	}
	return m, nil
}

func (m Model) handlePipelineWatchTick(msg PipelineWatchTickMsg) (Model, tea.Cmd) {
	if msg.watchID != m.pipelineWatchID {
		return m, nil
	}

	switch m.state {
	case ViewPipelineRuns:
		pipeline := m.pipelineRunsView.GetPipeline()
		if pipeline == nil {
			return m, nil
		}
		return m, tea.Batch(m.loadPipelineRuns(*pipeline), schedulePipelineWatchTick(msg.watchID))
	case ViewPipelineRun:
		run := m.pipelineRunView.GetRun()
		if run == nil || run.Status.IsFinished() {
			logger.Log("UI: Stopping pipeline watch, run is finished")
			return m, nil
		}
		return m, tea.Batch(m.loadPipelineRun(*run), schedulePipelineWatchTick(msg.watchID))
	default:
		return m, nil
	}
}

func (m Model) refreshPipelineView() (Model, tea.Cmd) {
	switch m.state {
	case ViewPipelineList:
		m.statusBar.SetMessage("Refreshing pipelines...", false)
		return m, m.loadPipelines()
	case ViewPipelineRuns:
		if pipeline := m.pipelineRunsView.GetPipeline(); pipeline != nil {
			return m, m.loadPipelineRuns(*pipeline)
		}
	case ViewPipelineRun:
		if run := m.pipelineRunView.GetRun(); run != nil {
			return m, m.loadPipelineRun(*run)
		}
	}
	return m, nil
}

func (m Model) beginStageApproval(decision domain.ApprovalDecision) (Model, tea.Cmd) {
	if m.state != ViewPipelineRun {
		return m, nil
	}

	run := m.pipelineRunView.GetRun()
	stage := m.pipelineRunView.GetSelectedStage()
	if run == nil || stage == nil {
		m.statusBar.SetMessage("No stage selected", true)
		return m, nil
	}
	if !stage.AwaitsApproval() {
		m.statusBar.SetMessage(fmt.Sprintf("Stage '%s' has no pending approval", stage.Name), true)
		return m, nil
	}

	m.stageApprovalView.Activate(run.Project, *stage, decision)
	return m, nil
}

func (m Model) submitStageApproval() tea.Cmd {
	approval := m.stageApprovalView.GetApproval()
	stage := m.stageApprovalView.GetStage()
	m.stageApprovalView.Deactivate()

	run := m.pipelineRunView.GetRun()
	if run == nil {
		return func() tea.Msg {
			return ErrorMsg{err: fmt.Errorf("no pipeline run selected")}
		}
	}

	provider, err := m.getPipelineProvider(run.PATID)
	if err != nil {
		return func() tea.Msg {
			return ErrorMsg{err: err}
		}
	}

	ctx := m.ctx
	runCopy := *run
	logger.Log("UI: Submitting %s for stage %s of run %d", approval.Decision, stage.Name, run.ID)
	return func() tea.Msg {
		if err := provider.SubmitStageApproval(ctx, approval); err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to %s stage '%s': %w", approval.Decision, stage.Name, err)}
		}
		return StageApprovalSubmittedMsg{run: runCopy, stageName: stage.Name, decision: approval.Decision}
	}
}

func (m Model) handleStageApprovalSubmitted(msg StageApprovalSubmittedMsg) (Model, tea.Cmd) {
	verb := "Approved"
	if msg.decision == domain.ApprovalDecisionReject {
		verb = "Rejected"
	}
	m.statusBar.SetMessage(fmt.Sprintf("%s stage '%s'", verb, msg.stageName), false)
	return m, tea.Batch(m.loadPipelineRun(msg.run), clearStatusAfterDelay(4*time.Second))
}

// pipelineBrowserTarget returns the URL ctrl+o should open: the pipeline in the list,
// the selected run in the run list (falling back to the pipeline), or the inspected run.
func (m Model) pipelineBrowserTarget() (string, string) {
	switch m.state {
	case ViewPipelineList:
		if pipeline := m.pipelineListView.GetSelectedPipeline(); pipeline != nil {
			return pipeline.URL, "pipeline"
		}
		return "", "pipeline"
	case ViewPipelineRuns:
		if run := m.pipelineRunsView.GetSelectedRun(); run != nil {
			return run.URL, "run"
		}
		if pipeline := m.pipelineRunsView.GetPipeline(); pipeline != nil {
			return pipeline.URL, "pipeline"
		}
		return "", "run"
	case ViewPipelineRun:
		if run := m.pipelineRunView.GetRun(); run != nil {
			return run.URL, "run"
		}
		return "", "run"
	default:
		return "", "pipeline"
	}
}

func isPipelineView(state ViewState) bool {
	return state == ViewPipelineList || state == ViewPipelineRuns || state == ViewPipelineRun
}
