package ui

import (
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func checkingPR() domain.PullRequest {
	return domain.PullRequest{
		Number:         7,
		Status:         domain.PRStatusOpen,
		MergeConflicts: domain.MergeConflictChecking,
		Repository:     domain.Repo{FullName: "Proj/api"},
	}
}

func TestNextMergeConflictStep_RechecksWhileComputing(t *testing.T) {
	status, recheck := Model{}.nextMergeConflictStep(checkingPR())
	if status != domain.MergeConflictChecking || recheck == nil {
		t.Errorf("expected a recheck while computing, got %q", status)
	}
}

func TestNextMergeConflictStep_GivesUpAfterTheLimit(t *testing.T) {
	m := Model{mergeConflicts: mergeConflictCheck{rechecks: maxMergeConflictRechecks}}
	status, recheck := m.nextMergeConflictStep(checkingPR())
	if status != domain.MergeConflictUnknown || recheck != nil {
		t.Errorf("expected unknown and no recheck, got %q", status)
	}
}

func TestNextMergeConflictStep_NoRecheckOnceKnownOrClosed(t *testing.T) {
	known := checkingPR()
	known.MergeConflicts = domain.MergeConflictFound
	closed := checkingPR()
	closed.Status = domain.PRStatusMerged
	for _, pr := range []domain.PullRequest{known, closed} {
		if _, recheck := (Model{}).nextMergeConflictStep(pr); recheck != nil {
			t.Errorf("did not expect a recheck for %+v", pr)
		}
	}
}

func TestHandleMergeConflictRecheck_DropsStaleTicks(t *testing.T) {
	m := Model{state: ViewPRInspect, mergeConflicts: mergeConflictCheck{prKey: "another"}}
	if _, cmd := m.handleMergeConflictRecheck(MergeConflictRecheckMsg{pr: checkingPR()}); cmd != nil {
		t.Error("expected a tick for a PR no longer open to be dropped")
	}
}

func TestHandlePRDetailLoaded_DropsResponseForAnotherPR(t *testing.T) {
	m := Model{mergeConflicts: mergeConflictCheck{prKey: "another"}}
	pr := checkingPR()
	if _, cmd := m.handlePRDetailLoaded(PRDetailLoadedMsg{pr: &pr}); cmd != nil {
		t.Error("expected a response for a PR no longer open to be dropped")
	}
}
