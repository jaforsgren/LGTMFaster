package views

import (
	"strings"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	loadingMergeRunsText = "loading..."
	noMergeRunsText      = "none built the merge commit"
	mergeRunsSeparator   = "  "
)

// PendingMergeRunsRequest returns the selected PR when the open Info pane shows a merged PR whose runs
// are not known yet, and marks the request in flight.
func (m *PRListViewModel) PendingMergeRunsRequest() *domain.PullRequest {
	pr := m.GetSelectedPR()
	if !m.showClosed || !m.split.IsOpen() || pr == nil || !m.StartMergeRunsRequest(*pr) {
		return nil
	}
	return pr
}

// StartMergeRunsRequest reports whether the merged PR's runs need fetching, marking the fetch in flight.
func (m *PRListViewModel) StartMergeRunsRequest(pr domain.PullRequest) bool {
	if _, merged := pr.MergeCommit(); !merged {
		return false
	}
	key := pr.Key()
	if _, loaded := m.mergeRuns[key]; loaded || m.mergeRunsPending[key] {
		return false
	}
	m.mergeRunsPending[key] = true
	return true
}

func (m *PRListViewModel) SetMergeRuns(prKey string, runs []domain.PipelineRun) {
	delete(m.mergeRunsPending, prKey)
	m.mergeRuns[prKey] = runs
}

// MergeRunsFailed forgets the request so the runs are fetched again next time.
func (m *PRListViewModel) MergeRunsFailed(prKey string) {
	delete(m.mergeRunsPending, prKey)
}

func (m *PRListViewModel) MergeRuns(prKey string) ([]domain.PipelineRun, bool) {
	runs, loaded := m.mergeRuns[prKey]
	return runs, loaded
}

// mergeRunsText summarises the runs as status icons and pipeline names.
func (m *PRListViewModel) mergeRunsText(pr domain.PullRequest) string {
	runs, loaded := m.mergeRuns[pr.Key()]
	switch {
	case !loaded:
		return loadingMergeRunsText
	case len(runs) == 0:
		return noMergeRunsText
	}
	summaries := make([]string, len(runs))
	for i, run := range runs {
		summaries[i] = appearanceFor(run.Status).icon + " " + run.PipelineName
	}
	return strings.Join(summaries, mergeRunsSeparator)
}
