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

// PipelineRunsFailedMsg names the pipeline whose runs could not be fetched, so its in-flight
// request is forgotten and the list stops showing it as loading.
type PipelineRunsFailedMsg struct {
	pipelineKey string
	err         error
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
	m.applyPipelineFavorites()
	m.applyPipelinePreferences()
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
			return PipelineRunsFailedMsg{pipelineKey: pipeline.FavoriteKey(), err: fmt.Errorf("failed to load runs for %s: %w", pipeline.Name, err)}
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
	m.markPipelineViewed(pipeline)
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
	m, resumeCmd := m.resumePipelines()
	return m, tea.Batch(clearStatusAfterDelay(4*time.Second), resumeCmd)
}

func (m Model) handlePipelineRunsLoaded(msg PipelineRunsLoadedMsg) (Model, tea.Cmd) {
	m.pipelineListView.SetPreviewRuns(domain.PipelineFavoriteKey(msg.patID, msg.pipeline.Project, msg.pipeline.PipelineID), msg.runs)

	current := m.pipelineRunsView.GetPipeline()
	if current == nil || current.Identifier() != msg.pipeline || current.PATID != msg.patID {
		return m, nil
	}
	m.pipelineRunsView.SetRuns(msg.runs)
	return m.resumePipelineRuns()
}

func (m Model) handlePipelineRunsFailed(msg PipelineRunsFailedMsg) (Model, tea.Cmd) {
	m.pipelineListView.PreviewRunsFailed(msg.pipelineKey)
	m.statusBar.SetMessage(msg.err.Error(), true)
	return m, nil
}

func (m Model) handlePipelineRunLoaded(msg PipelineRunLoadedMsg) (Model, tea.Cmd) {
	m.pipelineRunsView.SetPreviewStages(msg.run.FavoriteKey(), msg.stages)

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
		cmds := []tea.Cmd{m.loadPipelineRuns(*pipeline), schedulePipelineWatchTick(msg.watchID)}
		if run := m.pipelineRunsView.GetSelectedRun(); run != nil && m.pipelineRunsView.IsPreviewOpen() {
			cmds = append(cmds, m.loadPipelineRun(*run))
		}
		return m, tea.Batch(cmds...)
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
		m.pipelineListView.ResetPreview()
		return m, m.loadPipelines()
	case ViewPipelineRuns:
		m.pipelineRunsView.ResetPreview()
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
		if run := m.pipelineListView.GetSelectedRun(); run != nil {
			return run.URL, "run"
		}
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

func (m Model) applyPipelineFavorites() {
	keys, err := m.repository.ListPipelineFavorites()
	if err != nil {
		logger.LogError("LOAD_PIPELINE_FAVORITES", "storage", err)
		return
	}

	favorites := make(map[string]bool, len(keys))
	for _, key := range keys {
		favorites[key] = true
	}
	m.pipelineListView.SetFavorites(favorites)
	m.pipelineRunsView.SetFavorites(favorites)
	if run := m.pipelineRunView.GetRun(); run != nil {
		m.pipelineRunView.SetFavorite(favorites[run.FavoriteKey()])
	}
}

func (m Model) applyPipelinePreferences() {
	if settings, err := m.repository.GetSettings(); err == nil {
		m.pipelineListView.SetSortMode(settings.PipelineSortMode)
	} else {
		logger.LogError("LOAD_PIPELINE_SORT_MODE", "storage", err)
	}

	if lastViewed, err := m.repository.ListPipelineLastViewed(); err == nil {
		m.pipelineListView.SetLastViewed(lastViewed)
	} else {
		logger.LogError("LOAD_PIPELINE_LAST_VIEWED", "storage", err)
	}
}

func (m Model) markPipelineViewed(pipeline domain.Pipeline) {
	if err := m.repository.MarkPipelineViewed(pipeline.FavoriteKey(), time.Now()); err != nil {
		logger.LogError("MARK_PIPELINE_VIEWED", pipeline.FavoriteKey(), err)
		return
	}
	if lastViewed, err := m.repository.ListPipelineLastViewed(); err == nil {
		m.pipelineListView.SetLastViewed(lastViewed)
	}
}

func (m Model) setPipelineSortMode(mode domain.PipelineSortMode) (Model, tea.Cmd) {
	if m.state != ViewPipelineList {
		return m, nil
	}

	m.pipelineListView.SetSortMode(mode)
	label := views.PipelineSortModeLabel(mode)

	settings, err := m.repository.GetSettings()
	if err == nil {
		settings.PipelineSortMode = mode
		err = m.repository.SaveSettings(settings)
	}
	if err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Sorted by %s (not saved: %v)", label, err), true)
		return m, clearStatusAfterDelay(4 * time.Second)
	}

	m.statusBar.SetMessage("Sorted by "+label, false)
	return m, tea.Batch(m.loadPipelinePreview(), clearStatusAfterDelay(4*time.Second))
}

// selectedFavoriteTarget returns the favorite key and display name of whatever is selected in the current pipeline view.
func (m Model) selectedFavoriteTarget() (string, string) {
	switch m.state {
	case ViewPipelineList:
		if node := m.pipelineListView.GetSelectedNode(); node != nil {
			return node.Key, node.Name
		}
	case ViewPipelineRuns:
		if run := m.pipelineRunsView.GetSelectedRun(); run != nil {
			return run.FavoriteKey(), "run " + run.Number
		}
	case ViewPipelineRun:
		if run := m.pipelineRunView.GetRun(); run != nil {
			return run.FavoriteKey(), "run " + run.Number
		}
	}
	return "", ""
}

func (m Model) togglePipelineFavorite() (Model, tea.Cmd) {
	key, name := m.selectedFavoriteTarget()
	if key == "" {
		m.statusBar.SetMessage("Nothing selected to favorite", true)
		return m, nil
	}

	isFavorite, err := m.repository.TogglePipelineFavorite(key)
	if err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to update favorites: %v", err), true)
		return m, nil
	}

	m.applyPipelineFavorites()
	if isFavorite {
		m.statusBar.SetMessage(fmt.Sprintf("Added %s to favorites", name), false)
	} else {
		m.statusBar.SetMessage(fmt.Sprintf("Removed %s from favorites", name), false)
	}
	return m, clearStatusAfterDelay(4 * time.Second)
}

// loadPipelinePreview fetches whatever the open details pane is missing for the current selection.
func (m Model) loadPipelinePreview() tea.Cmd {
	switch m.state {
	case ViewPipelineList:
		var cmds []tea.Cmd
		for _, pipeline := range m.pipelineListView.PendingRunRequests() {
			cmds = append(cmds, m.loadPipelineRuns(pipeline))
		}
		return tea.Batch(cmds...)
	case ViewPipelineRuns:
		if run := m.pipelineRunsView.NextPreviewRequest(); run != nil {
			return m.loadPipelineRun(*run)
		}
	}
	return nil
}

func (m Model) toggleFavoritesOnly() (Model, tea.Cmd) {
	m.pipelineListView.ToggleFavoritesOnly()
	if m.pipelineListView.IsFavoritesOnly() {
		m.statusBar.SetMessage("Showing favorites only", false)
	} else {
		m.statusBar.SetMessage("Showing all pipelines", false)
	}
	return m, tea.Batch(m.loadPipelinePreview(), clearStatusAfterDelay(4*time.Second))
}

func isPipelineView(state ViewState) bool {
	return state == ViewPipelineList || state == ViewPipelineRuns || state == ViewPipelineRun
}
