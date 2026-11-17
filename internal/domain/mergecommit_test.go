package domain

import (
	"testing"
	"time"
)

func TestMergeCommit_OfAMergedPullRequest(t *testing.T) {
	closedAt := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	pr := PullRequest{Status: PRStatusMerged, MergeCommitID: "abc", TargetBranch: "main", ClosedAt: closedAt,
		Repository: Repo{FullName: "Proj/api"}}

	merge, ok := pr.MergeCommit()

	want := MergeCommit{Repository: "Proj/api", TargetBranch: "main", CommitID: "abc", MergedAt: closedAt}
	if !ok || merge != want {
		t.Errorf("expected %+v, got %+v (%v)", want, merge, ok)
	}
}

func TestMergeCommit_NoneUnlessMergedWithACommit(t *testing.T) {
	for _, pr := range []PullRequest{
		{Status: PRStatusClosed, MergeCommitID: "abc"},
		{Status: PRStatusOpen, MergeCommitID: "abc"},
		{Status: PRStatusMerged},
	} {
		if _, ok := pr.MergeCommit(); ok {
			t.Errorf("expected no merge commit for %+v", pr)
		}
	}
}
