package github

import (
	"testing"

	"github.com/google/go-github/v57/github"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestChecks_CombineStatusesAndCheckRuns(t *testing.T) {
	status := &github.CombinedStatus{Statuses: []*github.RepoStatus{
		{Context: github.String("ci/legacy"), State: github.String("success")},
	}}
	runs := []*github.CheckRun{
		{Name: github.String("build"), Status: github.String("in_progress")},
		{Name: github.String("lint"), Status: github.String(checkRunCompleted), Conclusion: github.String("failure")},
	}

	checks := domain.SummarizeChecks(append(statusChecks(status), checkRunChecks(runs)...))

	if checks.State != domain.CheckStateFailing || len(checks.Checks) != 3 || checks.Failing()[0] != "lint" {
		t.Errorf("expected lint failing among three checks, got %+v", checks)
	}
}

func TestReviewedBy_IgnoresPendingReviews(t *testing.T) {
	reviews := []*github.PullRequestReview{
		{User: &github.User{Login: github.String("johan")}, State: github.String(reviewStatePending)},
		{User: &github.User{Login: github.String("anna")}, State: github.String("APPROVED")},
	}

	if reviewedBy(reviews, "johan") || !reviewedBy(reviews, "anna") {
		t.Error("expected only submitted reviews to count")
	}
}
