package azuredevops

import (
	"context"
	"fmt"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

const (
	// pullRequestMergeRefFormat is the ref build validation policies build a pull request on.
	pullRequestMergeRefFormat = "refs/pull/%d/merge"
	pullRequestBuildsPageSize = 50
	tfsGitRepositoryType      = "TfsGit"
)

var pipelineCheckStates = map[domain.PipelineStatus]domain.CheckState{
	domain.PipelineStatusQueued:             domain.CheckStatePending,
	domain.PipelineStatusRunning:            domain.CheckStatePending,
	domain.PipelineStatusWaitingForApproval: domain.CheckStatePending,
	domain.PipelineStatusSucceeded:          domain.CheckStatePassing,
	domain.PipelineStatusPartiallySucceeded: domain.CheckStatePassing,
	domain.PipelineStatusFailed:             domain.CheckStateFailing,
	domain.PipelineStatusCanceled:           domain.CheckStateFailing,
}

// ListPullRequestBuilds returns the builds of a pull request's merge ref, newest first.
func (c *Client) ListPullRequestBuilds(ctx context.Context, projectID, repoID string, pullRequestID int) ([]build.Build, error) {
	branch := fmt.Sprintf(pullRequestMergeRefFormat, pullRequestID)
	repositoryType := tfsGitRepositoryType
	queryOrder := build.BuildQueryOrderValues.QueueTimeDescending
	response, err := c.buildClient.GetBuilds(ctx, build.GetBuildsArgs{
		Project:        &projectID,
		BranchName:     &branch,
		RepositoryId:   &repoID,
		RepositoryType: &repositoryType,
		QueryOrder:     &queryOrder,
		Top:            intPtr(pullRequestBuildsPageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list builds of PR %d: %w", pullRequestID, err)
	}
	if response == nil {
		return nil, nil
	}
	return response.Value, nil
}

// GetChecks reports the latest build of each pipeline that built the pull request.
func (p *Provider) GetChecks(ctx context.Context, identifier domain.PRIdentifier) (domain.Checks, error) {
	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, identifier.Repository)
	if err != nil {
		return domain.Checks{}, err
	}
	builds, err := p.client.ListPullRequestBuilds(ctx, projectID, repoID, identifier.Number)
	if err != nil {
		return domain.Checks{}, err
	}
	return domain.SummarizeChecks(latestBuildChecks(builds)), nil
}

// latestBuildChecks keeps the newest build per definition, since builds come newest first.
func latestBuildChecks(builds []build.Build) []domain.Check {
	seen := make(map[int]bool)
	var checks []domain.Check
	for _, b := range builds {
		if b.Definition == nil || b.Definition.Id == nil || seen[*b.Definition.Id] {
			continue
		}
		seen[*b.Definition.Id] = true
		state, known := pipelineCheckStates[mapBuildStatus(b.Status, b.Result)]
		if !known {
			continue
		}
		checks = append(checks, domain.Check{Name: common.GetString(b.Definition.Name), State: state})
	}
	return checks
}
