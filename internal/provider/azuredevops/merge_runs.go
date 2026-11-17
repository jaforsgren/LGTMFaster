package azuredevops

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

const (
	mergeRunsPageSize = 100
	// mergeRunsClockSlack reaches back before the merge, since the time the pull request was closed
	// and the time its first build was queued come from different services.
	mergeRunsClockSlack = time.Minute
)

// ListBranchBuildsSince lists the repository's builds of a branch queued since the given time, oldest
// first; with a zero time, the latest builds. The builds API cannot filter by commit, so callers match
// the commit themselves.
func (c *Client) ListBranchBuildsSince(ctx context.Context, projectID, repoID, branch string, since time.Time) ([]build.Build, error) {
	repositoryType := azureReposGitRepositoryType
	queryOrder := build.BuildQueryOrderValues.QueueTimeAscending
	if since.IsZero() {
		queryOrder = build.BuildQueryOrderValues.QueueTimeDescending
	}
	args := build.GetBuildsArgs{
		Project:        &projectID,
		BranchName:     &branch,
		RepositoryId:   &repoID,
		RepositoryType: &repositoryType,
		QueryOrder:     &queryOrder,
		Top:            intPtr(mergeRunsPageSize),
	}
	if !since.IsZero() {
		args.MinTime = &azuredevops.Time{Time: since}
	}
	response, err := c.buildClient.GetBuilds(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to list builds of %s since %s: %w", branch, since.Format(time.RFC3339), err)
	}
	if response == nil {
		return nil, nil
	}
	builds := response.Value
	if since.IsZero() {
		slices.Reverse(builds)
	}
	return builds, nil
}

// ListMergeRuns finds the target branch's builds of the merge commit. This includes pipelines that a
// pipeline-completion trigger started, since they build the same commit; runs that batched the merge
// into a later commit are not found.
func (p *Provider) ListMergeRuns(ctx context.Context, merge domain.MergeCommit) ([]domain.PipelineRun, error) {
	projectName, _, err := parseRepositoryIdentifier(merge.Repository)
	if err != nil {
		return nil, err
	}
	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, merge.Repository)
	if err != nil {
		return nil, err
	}

	var since time.Time
	if !merge.MergedAt.IsZero() {
		since = merge.MergedAt.Add(-mergeRunsClockSlack)
	}
	builds, err := p.client.ListBranchBuildsSince(ctx, projectID, repoID, branchRefPrefix+merge.TargetBranch, since)
	if err != nil {
		logger.LogError("AZDO_LIST_MERGE_RUNS", merge.Repository, err)
		return nil, err
	}

	matching := buildsOfCommit(builds, merge.CommitID)
	logger.Log("AzureDevOps: Found %d runs of merge commit %s in %s", len(matching), merge.CommitID, merge.Repository)
	return p.convertPipelineRuns(ctx, matching, projectName), nil
}

func buildsOfCommit(builds []build.Build, commitID string) []build.Build {
	var matching []build.Build
	for _, b := range builds {
		if common.GetString(b.SourceVersion) == commitID {
			matching = append(matching, b)
		}
	}
	return matching
}
