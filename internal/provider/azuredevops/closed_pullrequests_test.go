package azuredevops

import (
	"context"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func closedTestPR(prID int, status git.PullRequestStatus, closedAt time.Time) git.GitPullRequest {
	pr := *createMockPR(prID, "PR", nil)
	pr.Status = &status
	pr.ClosedDate = &azuredevops.Time{Time: closedAt}
	pr.LastMergeCommit = &git.GitCommitRef{CommitId: strPtr("squash-sha")}
	return pr
}

func newClosedPRsTestProvider(gitClient *mockGitClient) *Provider {
	repoID := mustParseUUID("12345678-1234-1234-1234-123456789012")
	gitClient.repositories = &[]git.GitRepository{{Id: &repoID, Name: strPtr("api")}}
	provider := newPullRequestTestProvider(gitClient)
	provider.client.coreClient = &mockCoreClient{projects: []core.TeamProjectReference{{Name: strPtr(testProject)}}}
	return provider
}

func TestListClosedPullRequests_KeepsCompletedAndAbandonedOnes(t *testing.T) {
	closedAt := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	gitClient := &mockGitClient{pullRequests: &[]git.GitPullRequest{
		closedTestPR(1, git.PullRequestStatusValues.Active, time.Time{}),
		closedTestPR(2, git.PullRequestStatusValues.Completed, closedAt),
		closedTestPR(3, git.PullRequestStatusValues.Abandoned, closedAt),
	}}
	provider := newClosedPRsTestProvider(gitClient)

	prs, err := provider.ListClosedPullRequests(context.Background(), "user")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 2 || prs[0].Number != 2 || prs[1].Number != 3 {
		t.Fatalf("expected the completed and the abandoned PR, got %+v", prs)
	}
	if prs[0].Status != domain.PRStatusMerged || !prs[0].ClosedAt.Equal(closedAt) || prs[0].MergeCommitID != "squash-sha" {
		t.Errorf("expected a merged PR with its closing time and merge commit, got %+v", prs[0])
	}
	if prs[1].Status != domain.PRStatusClosed || prs[1].MergeCommitID != "" {
		t.Errorf("expected an abandoned PR without a merge commit, got %+v", prs[1])
	}
}

func TestListClosedPullRequests_ReadsTheLatestOfEveryState(t *testing.T) {
	gitClient := &mockGitClient{}
	provider := newClosedPRsTestProvider(gitClient)

	if _, err := provider.ListClosedPullRequests(context.Background(), "user"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	args := gitClient.lastListPRArgs
	if args == nil || *args.SearchCriteria.Status != git.PullRequestStatusValues.All || *args.Top != closedPullRequestsPerRepository {
		t.Errorf("expected the latest %d PRs in any state, got %+v", closedPullRequestsPerRepository, args)
	}
}

func TestListPullRequests_ReadsEveryActivePullRequest(t *testing.T) {
	gitClient := &mockGitClient{}
	provider := newClosedPRsTestProvider(gitClient)

	if _, err := provider.ListPullRequests(context.Background(), "user"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	args := gitClient.lastListPRArgs
	if args == nil || *args.SearchCriteria.Status != git.PullRequestStatusValues.Active || args.Top != nil {
		t.Errorf("expected every active PR, got %+v", args)
	}
}

func TestConvertPullRequest_LeavesMergeCommitOfOpenPRsEmpty(t *testing.T) {
	open := closedTestPR(1, git.PullRequestStatusValues.Active, time.Time{})

	if pr := convertPullRequest(&open, "user"); pr.MergeCommitID != "" {
		t.Errorf("expected no merge commit while the PR is open, got %q", pr.MergeCommitID)
	}
}
