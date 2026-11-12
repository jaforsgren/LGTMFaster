package azuredevops

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func newRunDetailsTestProvider(buildClient *mockBuildClient, gitClient *mockGitClient) *Provider {
	provider := newPipelineTestProvider(buildClient, nil)
	provider.client.gitClient = gitClient
	return provider
}

func azureReposBuild(b build.Build, commitID string) build.Build {
	b.Repository = &build.BuildRepository{Id: strPtr("repo-id"), Type: strPtr(azureReposGitRepositoryType)}
	b.SourceVersion = &commitID
	return b
}

func succeededBuild(id int) build.Build {
	succeeded := build.BuildResultValues.Succeeded
	return newTestBuild(id, "20260101.1", build.BuildStatusValues.Completed, &succeeded)
}

func TestConvertPipelineRuns_TakesTheCommitMessageFromTheTriggerInfo(t *testing.T) {
	b := succeededBuild(1)
	b.TriggerInfo = &map[string]string{triggerInfoCIMessage: "feature: add login\n\nlong body"}
	gitClient := &mockGitClient{}

	runs := newRunDetailsTestProvider(&mockBuildClient{}, gitClient).convertPipelineRuns(context.Background(), []build.Build{b}, testProject)

	if runs[0].CommitMessage != "feature: add login" {
		t.Errorf("expected the first line of the trigger message, got %q", runs[0].CommitMessage)
	}
	if gitClient.commitCalls != 0 {
		t.Error("expected no commit lookup when the trigger info has the message")
	}
}

func TestConvertPipelineRuns_LooksUpAndCachesTheCommitMessage(t *testing.T) {
	b := azureReposBuild(succeededBuild(1), "abc123")
	gitClient := &mockGitClient{commit: &git.GitCommit{Comment: strPtr("fix: retry deploys\nbody")}}
	provider := newRunDetailsTestProvider(&mockBuildClient{}, gitClient)

	provider.convertPipelineRuns(context.Background(), []build.Build{b}, testProject)
	runs := provider.convertPipelineRuns(context.Background(), []build.Build{b}, testProject)

	if runs[0].CommitMessage != "fix: retry deploys" {
		t.Errorf("expected the looked up commit summary, got %q", runs[0].CommitMessage)
	}
	if gitClient.commitCalls != 1 {
		t.Errorf("expected the commit to be read once, got %d", gitClient.commitCalls)
	}
}

func TestConvertPipelineRuns_SkipsCommitsOutsideAzureRepos(t *testing.T) {
	b := azureReposBuild(succeededBuild(1), "abc123")
	b.Repository.Type = strPtr("GitHub")
	gitClient := &mockGitClient{commit: &git.GitCommit{Comment: strPtr("unused")}}

	runs := newRunDetailsTestProvider(&mockBuildClient{}, gitClient).convertPipelineRuns(context.Background(), []build.Build{b}, testProject)

	if runs[0].CommitMessage != "" || gitClient.commitCalls != 0 {
		t.Errorf("expected no lookup for a GitHub repository, got %q", runs[0].CommitMessage)
	}
}

func TestConvertPipelineRuns_NamesTheStageAnUnfinishedRunWaitsAt(t *testing.T) {
	stageID, checkpointID, approvalID := uuid.New(), uuid.New(), uuid.New()
	inProgress, pending := build.TimelineRecordStateValues.InProgress, build.TimelineRecordStateValues.Pending
	records := []build.TimelineRecord{
		newTimelineRecord(stageID, nil, timelineRecordTypeStage, "Prod", 1, inProgress, nil),
		newTimelineRecord(checkpointID, &stageID, "Checkpoint", "Checkpoint", 1, inProgress, nil),
		newTimelineRecord(approvalID, &checkpointID, timelineRecordTypeApproval, "Approval", 1, pending, nil),
	}
	b := newTestBuild(1, "20260101.1", build.BuildStatusValues.InProgress, nil)
	buildClient := &mockBuildClient{timeline: &build.Timeline{Records: &records}}

	runs := newRunDetailsTestProvider(buildClient, &mockGitClient{}).convertPipelineRuns(context.Background(), []build.Build{b}, testProject)

	if runs[0].CurrentStage != "Prod" || runs[0].Status != domain.PipelineStatusWaitingForApproval {
		t.Errorf("expected the run waiting for approval at Prod, got %q %s", runs[0].CurrentStage, runs[0].Status)
	}
}

func TestConvertPipelineRuns_LeavesFinishedRunsWithoutAStage(t *testing.T) {
	stageID := uuid.New()
	records := []build.TimelineRecord{
		newTimelineRecord(stageID, nil, timelineRecordTypeStage, "Build", 1, build.TimelineRecordStateValues.InProgress, nil),
	}
	buildClient := &mockBuildClient{timeline: &build.Timeline{Records: &records}}

	runs := newRunDetailsTestProvider(buildClient, &mockGitClient{}).convertPipelineRuns(context.Background(), []build.Build{succeededBuild(1)}, testProject)

	if runs[0].CurrentStage != "" || runs[0].Status != domain.PipelineStatusSucceeded {
		t.Errorf("expected a finished run to keep its result and no stage, got %q %s", runs[0].CurrentStage, runs[0].Status)
	}
}
