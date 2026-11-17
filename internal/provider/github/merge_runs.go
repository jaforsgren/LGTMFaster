package github

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/go-github/v57/github"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
)

const workflowRunCompleted = "completed"

// workflowRunStatuses map an unfinished run's status; "waiting" is a deployment waiting for an
// environment's approval.
var workflowRunStatuses = map[string]domain.PipelineStatus{
	"requested":   domain.PipelineStatusQueued,
	"queued":      domain.PipelineStatusQueued,
	"pending":     domain.PipelineStatusQueued,
	"in_progress": domain.PipelineStatusRunning,
	"waiting":     domain.PipelineStatusWaitingForApproval,
}

var workflowRunConclusions = map[string]domain.PipelineStatus{
	"success":         domain.PipelineStatusSucceeded,
	"failure":         domain.PipelineStatusFailed,
	"timed_out":       domain.PipelineStatusFailed,
	"startup_failure": domain.PipelineStatusFailed,
	"cancelled":       domain.PipelineStatusCanceled,
	"skipped":         domain.PipelineStatusSkipped,
	"neutral":         domain.PipelineStatusSkipped,
	"action_required": domain.PipelineStatusWaitingForApproval,
}

// ListWorkflowRunsForCommit returns the repository's workflow runs of a commit, newest first.
func (c *Client) ListWorkflowRunsForCommit(ctx context.Context, owner, repo, sha string) ([]*github.WorkflowRun, error) {
	result, _, err := c.client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, &github.ListWorkflowRunsOptions{
		HeadSHA:     sha,
		ListOptions: github.ListOptions{PerPage: listPageSize},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow runs of %s: %w", sha, err)
	}
	return result.WorkflowRuns, nil
}

// ListMergeRuns returns the workflow runs of the merge commit. GitHub has no pipeline views here yet,
// so the runs are only listed and opened in the browser.
func (p *Provider) ListMergeRuns(ctx context.Context, merge domain.MergeCommit) ([]domain.PipelineRun, error) {
	owner, repo, err := common.ParseGitHubRepository(merge.Repository)
	if err != nil {
		return nil, err
	}
	workflowRuns, err := p.client.ListWorkflowRunsForCommit(ctx, owner, repo, merge.CommitID)
	if err != nil {
		logger.LogError("GITHUB_LIST_MERGE_RUNS", merge.Repository, err)
		return nil, err
	}

	runs := make([]domain.PipelineRun, 0, len(workflowRuns))
	for _, workflowRun := range workflowRuns {
		runs = append(runs, convertWorkflowRun(workflowRun, merge.Repository))
	}
	slices.Reverse(runs)
	return runs, nil
}

func convertWorkflowRun(run *github.WorkflowRun, repository string) domain.PipelineRun {
	converted := domain.PipelineRun{
		ID:            int(run.GetID()),
		Number:        fmt.Sprintf("#%d", run.GetRunNumber()),
		PipelineID:    int(run.GetWorkflowID()),
		PipelineName:  run.GetName(),
		Project:       repository,
		Status:        workflowRunStatus(run),
		SourceBranch:  run.GetHeadBranch(),
		CommitMessage: common.CommitSummary(run.GetHeadCommit().GetMessage()),
		RequestedBy:   domain.User{Username: run.GetActor().GetLogin()},
		QueuedAt:      run.GetCreatedAt().Time,
		StartedAt:     run.GetRunStartedAt().Time,
		URL:           run.GetHTMLURL(),
		ProviderType:  domain.ProviderGitHub,
	}
	if run.GetStatus() == workflowRunCompleted {
		converted.FinishedAt = run.GetUpdatedAt().Time
	}
	return converted
}

func workflowRunStatus(run *github.WorkflowRun) domain.PipelineStatus {
	statuses, value := workflowRunStatuses, run.GetStatus()
	if value == workflowRunCompleted {
		statuses, value = workflowRunConclusions, run.GetConclusion()
	}
	if status, known := statuses[value]; known {
		return status
	}
	return domain.PipelineStatusUnknown
}
