package ui

import (
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

const notMergedMessage = "Only a merged pull request has runs of its merge commit"

// MergeRunsLoadedMsg carries the pipeline runs that built a merged PR's merge commit.
type MergeRunsLoadedMsg struct {
	prKey string
	runs  []domain.PipelineRun
	err   error
}

// loadPRListDetails fetches what the PR list's Info pane shows for the selected PR beyond the PR
// itself, and the linked work items the list sorts or filters by.
func (m Model) loadPRListDetails() tea.Cmd {
	return tea.Batch(m.loadPRListComments(), m.loadPRListMergeRuns(), m.loadPRListWorkItemIDs())
}

func (m Model) loadPRListMergeRuns() tea.Cmd {
	pr := m.prListView.PendingMergeRunsRequest()
	if pr == nil {
		return nil
	}
	return m.loadMergeRuns(*pr)
}

// loadMergeRuns tags the runs with the PR's PAT, so opening one finds the provider that listed it.
func (m Model) loadMergeRuns(pr domain.PullRequest) tea.Cmd {
	provider := m.getProviderForPR(pr)
	merge, _ := pr.MergeCommit()
	ctx := m.ctx
	return func() tea.Msg {
		if provider == nil {
			return MergeRunsLoadedMsg{prKey: pr.Key(), err: errors.New(noProviderMessage)}
		}
		runs, err := provider.ListMergeRuns(ctx, merge)
		for i := range runs {
			runs[i].PATID = pr.PATID
			runs[i].ProviderType = provider.GetType()
		}
		return MergeRunsLoadedMsg{prKey: pr.Key(), runs: runs, err: err}
	}
}

func (m Model) handleMergeRunsLoaded(msg MergeRunsLoadedMsg) (Model, tea.Cmd) {
	listed := m.mergeRunsView.IsActive() && m.mergeRunsView.PRKey() == msg.prKey
	if msg.err != nil {
		m.prListView.MergeRunsFailed(msg.prKey)
		if listed {
			m.mergeRunsView.Deactivate()
		}
		m.statusBar.SetMessage(fmt.Sprintf("Failed to find the runs of the merge: %v", msg.err), true)
		return m, nil
	}
	m.prListView.SetMergeRuns(msg.prKey, msg.runs)
	if listed {
		m.mergeRunsView.SetRuns(msg.runs)
	}
	return m, nil
}

// handleMergeRunsKey lists the runs of the selected merged PR, fetching them unless they are known.
func handleMergeRunsKey(m Model) (Model, tea.Cmd) {
	pr := m.prListView.GetSelectedPR()
	if pr == nil {
		return m, nil
	}
	if _, merged := pr.MergeCommit(); !merged {
		m.statusBar.SetMessage(notMergedMessage, true)
		return m, nil
	}
	m.mergeRunsView.Activate(*pr)
	if runs, loaded := m.prListView.MergeRuns(pr.Key()); loaded {
		m.mergeRunsView.SetRuns(runs)
		return m, nil
	}
	if !m.prListView.StartMergeRunsRequest(*pr) {
		return m, nil
	}
	return m, m.loadMergeRuns(*pr)
}

func (m Model) handleMergeRunsViewKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.mergeRunsView.HandleKey(msg) {
	case views.MergeRunsActionCancel:
		m.mergeRunsView.Deactivate()
	case views.MergeRunsActionOpen:
		return m.openSelectedMergeRun()
	case views.MergeRunsActionOpenBrowser:
		return m.openMergeRunInBrowser(*m.mergeRunsView.Selected())
	}
	return m, nil
}

// openSelectedMergeRun opens the run in the run view, or in the browser for providers without one.
func (m Model) openSelectedMergeRun() (Model, tea.Cmd) {
	run := *m.mergeRunsView.Selected()
	if _, ok := m.getProviderForPAT(run.PATID).(domain.PipelineProvider); !ok {
		return m.openMergeRunInBrowser(run)
	}
	m.mergeRunsView.Deactivate()
	return m.enterPipelineRun(run)
}

func (m Model) openMergeRunInBrowser(run domain.PipelineRun) (Model, tea.Cmd) {
	if err := openExternal(run.URL); err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to open browser: %v", err), true)
		return m, nil
	}
	m.statusBar.SetMessage(fmt.Sprintf("Opened run %s in browser", run.Number), false)
	return m, nil
}
