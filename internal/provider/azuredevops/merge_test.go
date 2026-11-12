package azuredevops

import (
	"context"
	"testing"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func TestCompletePullRequest_SendsTheLastMergeSourceCommit(t *testing.T) {
	gitClient := &mockGitClient{pullRequest: &git.GitPullRequest{
		LastMergeSourceCommit: &git.GitCommitRef{CommitId: strPtr("abc123")},
	}}
	client := &Client{gitClient: gitClient}

	if err := client.CompletePullRequest(context.Background(), "proj", "repo", 42, "squash", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	update := gitClient.lastUpdateArgs.GitPullRequestToUpdate
	if update.LastMergeSourceCommit == nil || *update.LastMergeSourceCommit.CommitId != "abc123" {
		t.Errorf("expected the source commit abc123, got %+v", update.LastMergeSourceCommit)
	}
	if *update.Status != git.PullRequestStatusValues.Completed || *update.CompletionOptions.MergeStrategy != git.GitPullRequestMergeStrategyValues.Squash {
		t.Errorf("expected a squash completion, got %+v", update)
	}
}

func TestCompletePullRequest_FailsWithoutASourceCommit(t *testing.T) {
	gitClient := &mockGitClient{pullRequest: &git.GitPullRequest{}}
	client := &Client{gitClient: gitClient}

	if err := client.CompletePullRequest(context.Background(), "proj", "repo", 42, "squash", false); err == nil {
		t.Fatal("expected an error")
	}
	if gitClient.lastUpdateArgs != nil {
		t.Error("expected no completion request")
	}
}
