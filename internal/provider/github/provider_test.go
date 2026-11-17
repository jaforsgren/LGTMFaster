package github

import (
	"testing"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func closedGitHubPR(merged bool, closedAt time.Time) *github.PullRequest {
	pr := &github.PullRequest{
		Number:         github.Int(5),
		State:          github.String("closed"),
		ClosedAt:       &github.Timestamp{Time: closedAt},
		MergeCommitSHA: github.String("merge-sha"),
	}
	if merged {
		pr.MergedAt = &github.Timestamp{Time: closedAt}
	}
	return pr
}

func TestConvertPullRequest_MergedPRCarriesClosingTimeAndMergeCommit(t *testing.T) {
	closedAt := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	p := &Provider{}

	pr := p.convertPullRequest(closedGitHubPR(true, closedAt), "me")

	if pr.Status != domain.PRStatusMerged || !pr.ClosedAt.Equal(closedAt) || pr.MergeCommitID != "merge-sha" {
		t.Errorf("expected a merged PR with its closing time and merge commit, got %+v", pr)
	}
}

func TestConvertPullRequest_ClosedUnmergedPRHasNoMergeCommit(t *testing.T) {
	p := &Provider{}

	pr := p.convertPullRequest(closedGitHubPR(false, time.Now()), "me")

	if pr.Status != domain.PRStatusClosed || pr.MergeCommitID != "" {
		t.Errorf("expected a closed PR without a merge commit, got %+v", pr)
	}
}
