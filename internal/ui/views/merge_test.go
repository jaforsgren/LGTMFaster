package views

import (
	"strings"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestRenderMergeConflictNotice_OnlyWarnsOnRealConflicts(t *testing.T) {
	const conflictWarning = "has merge conflicts"
	cases := map[domain.MergeConflictStatus]string{
		domain.MergeConflictFound:    conflictWarning,
		domain.MergeConflictNone:     "is mergeable",
		domain.MergeConflictChecking: "Still checking",
		domain.MergeConflictUnknown:  "status unknown",
	}
	for status, want := range cases {
		notice := renderMergeConflictNotice(status)
		if !strings.Contains(notice, want) {
			t.Errorf("%q: expected %q in %q", status, want, notice)
		}
		if status != domain.MergeConflictFound && strings.Contains(notice, conflictWarning) {
			t.Errorf("%q: must not warn about conflicts", status)
		}
	}
}

func TestDashboard_ReadyToMergeRequiresKnownCleanMerge(t *testing.T) {
	const readyToMergeTitle = "Ready to merge"
	section := -1
	for i, definition := range dashboardSections {
		if definition.title == readyToMergeTitle {
			section = i
		}
	}
	if section < 0 {
		t.Fatal("ready to merge section not found")
	}
	passing := dashboardPRState{checksLoaded: true, checks: domain.SummarizeChecks([]domain.Check{{Name: "ci", State: domain.CheckStatePassing}})}
	for status, want := range map[domain.MergeConflictStatus]bool{
		domain.MergeConflictNone:     true,
		domain.MergeConflictFound:    false,
		domain.MergeConflictChecking: false,
		domain.MergeConflictUnknown:  false,
	} {
		pr := dashboardPR(1, "PR", domain.PRCategoryAuthored, func(pr *domain.PullRequest) {
			pr.ApprovalStatus = domain.ApprovalStatusApproved
			pr.MergeConflicts = status
		})
		if got := dashboardSections[section].includes(pr, passing); got != want {
			t.Errorf("%q: got %v, want %v", status, got, want)
		}
	}
}
