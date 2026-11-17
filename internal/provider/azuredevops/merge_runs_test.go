package azuredevops

import (
	"context"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

const testMergeCommit = "squash-sha"

func newMergeRunsTestProvider(buildClient *mockBuildClient) *Provider {
	provider := newPullRequestTestProvider(&mockGitClient{})
	provider.client.buildClient = buildClient
	return provider
}

func mergeTestBuild(id int, commitID string) build.Build {
	succeeded := build.BuildResultValues.Succeeded
	b := newTestBuild(id, "20261110.1", build.BuildStatusValues.Completed, &succeeded)
	b.SourceVersion = strPtr(commitID)
	b.TriggerInfo = &map[string]string{triggerInfoCIMessage: "feature: squashed work"}
	return b
}

func TestListMergeRuns_KeepsTheBuildsOfTheMergeCommit(t *testing.T) {
	mock := &mockBuildClient{builds: []build.Build{
		mergeTestBuild(1, testMergeCommit),
		mergeTestBuild(2, "later-commit"),
		mergeTestBuild(3, testMergeCommit),
	}}
	provider := newMergeRunsTestProvider(mock)

	runs, err := provider.ListMergeRuns(context.Background(), domain.MergeCommit{
		Repository: testRepository, TargetBranch: "main", CommitID: testMergeCommit, MergedAt: time.Now(),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 2 || runs[0].ID != 1 || runs[1].ID != 3 {
		t.Fatalf("expected the two builds of the merge commit, got %+v", runs)
	}
	if runs[0].Project != "Proj" || runs[0].CommitMessage != "feature: squashed work" {
		t.Errorf("expected the run to carry its project and commit, got %+v", runs[0])
	}
}

func TestListMergeRuns_QueriesTheTargetBranchFromJustBeforeTheMerge(t *testing.T) {
	mergedAt := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	mock := &mockBuildClient{}
	provider := newMergeRunsTestProvider(mock)

	if _, err := provider.ListMergeRuns(context.Background(), domain.MergeCommit{
		Repository: testRepository, TargetBranch: "main", CommitID: testMergeCommit, MergedAt: mergedAt,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	args := mock.lastGetBuildsArgs
	if *args.BranchName != "refs/heads/main" || *args.RepositoryId != "repo-id" || *args.Project != "proj-id" {
		t.Errorf("expected the repository's main branch, got %+v", args)
	}
	if !args.MinTime.Time.Equal(mergedAt.Add(-mergeRunsClockSlack)) || *args.QueryOrder != build.BuildQueryOrderValues.QueueTimeAscending {
		t.Errorf("expected builds queued from just before the merge, oldest first, got %+v", args)
	}
}

func TestListMergeRuns_WithoutAMergeTimeReadsTheLatestBuildsOldestFirst(t *testing.T) {
	mock := &mockBuildClient{builds: []build.Build{mergeTestBuild(3, testMergeCommit), mergeTestBuild(1, testMergeCommit)}}
	provider := newMergeRunsTestProvider(mock)

	runs, err := provider.ListMergeRuns(context.Background(), domain.MergeCommit{
		Repository: testRepository, TargetBranch: "main", CommitID: testMergeCommit,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 2 || runs[0].ID != 1 {
		t.Errorf("expected the runs oldest first, got %+v", runs)
	}

	args := mock.lastGetBuildsArgs
	if args.MinTime != nil || *args.QueryOrder != build.BuildQueryOrderValues.QueueTimeDescending {
		t.Errorf("expected the latest builds without a time filter, got %+v", args)
	}
}
