package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const testMergeCommitID = "squash-sha"

func mergedTestPR() domain.PullRequest {
	return domain.PullRequest{
		Number: 7, Title: "Squashed feature", Status: domain.PRStatusMerged, PATID: testPATID,
		ClosedAt: time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC), MergeCommitID: testMergeCommitID,
		TargetBranch: "main", Repository: domain.Repo{FullName: "Proj/api"},
	}
}

func testMergeRuns() []domain.PipelineRun {
	return []domain.PipelineRun{
		{ID: 11, Number: "20261110.1", PipelineName: "CI", Project: "Proj", Status: domain.PipelineStatusSucceeded, URL: "https://ci/11"},
		{ID: 12, Number: "20261110.2", PipelineName: "Deploy", Project: "Proj", Status: domain.PipelineStatusWaitingForApproval},
	}
}

// mergeRunsTestModel lists the PR among the closed ones, served by the given provider.
func mergeRunsTestModel(t *testing.T, provider domain.Provider, pr domain.PullRequest) Model {
	t.Helper()
	m := sizedPipelineTestModel(t, provider)
	m.state = ViewPRList
	m.prListView.SetClosedPRs([]domain.PullRequest{pr})
	m.prListView.ShowClosed(true)
	return m
}

func openMergeRuns(t *testing.T, m Model) Model {
	t.Helper()
	m, cmd := handleMergeRunsKey(m)
	m, _ = m.handleMergeRunsLoaded(findMsg[MergeRunsLoadedMsg](t, cmd))
	return m
}

func TestMergeRunsKey_ListsTheRunsOfTheMergeCommit(t *testing.T) {
	provider := &mockProvider{mergeRuns: testMergeRuns()}
	m := mergeRunsTestModel(t, provider, mergedTestPR())

	m = openMergeRuns(t, m)

	if !m.mergeRunsView.IsActive() || !strings.Contains(m.View(), "Deploy") {
		t.Fatal("expected the merge runs listed")
	}
	want := domain.MergeCommit{Repository: "Proj/api", TargetBranch: "main", CommitID: testMergeCommitID, MergedAt: mergedTestPR().ClosedAt}
	if provider.lastMerge == nil || *provider.lastMerge != want {
		t.Errorf("expected the runs of %+v, got %+v", want, provider.lastMerge)
	}
	if selected := m.mergeRunsView.Selected(); selected.PATID != testPATID || selected.ProviderType != domain.ProviderGitHub {
		t.Errorf("expected the run tagged with the PR's PAT, got %+v", selected)
	}
}

func TestMergeRunsKey_ReusesRunsAlreadyFetched(t *testing.T) {
	m := mergeRunsTestModel(t, &mockProvider{mergeRuns: testMergeRuns()}, mergedTestPR())
	m = openMergeRuns(t, m)
	m.mergeRunsView.Deactivate()

	m, cmd := handleMergeRunsKey(m)

	if cmd != nil || m.mergeRunsView.Selected() == nil {
		t.Error("expected the known runs to be listed without fetching them again")
	}
}

func TestMergeRunsKey_RefusesPullRequestsThatWereNotMerged(t *testing.T) {
	abandoned := mergedTestPR()
	abandoned.Status = domain.PRStatusClosed
	m := mergeRunsTestModel(t, &mockProvider{}, abandoned)

	m, cmd := handleMergeRunsKey(m)

	if cmd != nil || m.mergeRunsView.IsActive() || !strings.Contains(m.View(), notMergedMessage) {
		t.Error("expected a message instead of the runs list")
	}
}

func TestMergeRunsLoaded_ErrorClosesTheListAndRetriesNextTime(t *testing.T) {
	m := mergeRunsTestModel(t, &mockProvider{mergeRunsErr: errors.New("forbidden")}, mergedTestPR())

	m = openMergeRuns(t, m)

	if m.mergeRunsView.IsActive() || !strings.Contains(m.View(), "forbidden") {
		t.Error("expected the list closed and the error shown")
	}
	if _, cmd := handleMergeRunsKey(m); cmd == nil {
		t.Error("expected the runs to be fetched again")
	}
}

func TestMergeRuns_EnterOpensTheRunAndBackReturnsToThePRList(t *testing.T) {
	provider := &mockPipelineProvider{mockProvider: mockProvider{mergeRuns: testMergeRuns()}}
	m := mergeRunsTestModel(t, provider, mergedTestPR())
	m = openMergeRuns(t, m)

	m, _ = m.handleMergeRunsViewKey(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != ViewPipelineRun || m.mergeRunsView.IsActive() || m.pipelineRunView.GetRun().ID != 11 {
		t.Fatalf("expected the first run opened, got state %v", m.state)
	}
	m, _ = m.leavePipelineRun()
	if m.state != ViewPRList || !m.prListView.IsShowingClosed() {
		t.Errorf("expected back to return to the closed PRs, got %v", m.state)
	}
}

func TestMergeRuns_EnterOpensTheBrowserWithoutPipelineViews(t *testing.T) {
	var opened string
	restore := openExternal
	openExternal = func(url string) error { opened = url; return nil }
	t.Cleanup(func() { openExternal = restore })
	m := mergeRunsTestModel(t, &mockProvider{mergeRuns: testMergeRuns()}, mergedTestPR())
	m = openMergeRuns(t, m)

	m, _ = m.handleMergeRunsViewKey(tea.KeyMsg{Type: tea.KeyEnter})

	if opened != "https://ci/11" || m.state != ViewPRList {
		t.Errorf("expected the run opened in the browser, got %q in %v", opened, m.state)
	}
}

func TestPRListInfoPane_SummarisesTheRunsOfASelectedMergedPR(t *testing.T) {
	m := mergeRunsTestModel(t, &mockProvider{mergeRuns: testMergeRuns()}, mergedTestPR())

	m, cmd := handleTogglePreviewKey(m)
	if !strings.Contains(m.View(), "Runs:") {
		t.Fatal("expected a Runs row for a merged PR")
	}
	m, _ = m.handleMergeRunsLoaded(findMsg[MergeRunsLoadedMsg](t, cmd))

	if view := m.View(); !strings.Contains(view, "✓ CI") || !strings.Contains(view, "Deploy") {
		t.Error("expected the runs summarised in the Info pane")
	}
}
