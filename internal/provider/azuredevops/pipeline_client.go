package azuredevops

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/pipelinesapproval"
)

const (
	azureReposGitRepositoryType = "TfsGit"
	pipelineRunsPageSize        = 50
)

// activeBuildStatusFilter relies on statusFilter being a flags enum, which the REST API accepts as
// a comma-separated list. Runs waiting for an approval are inProgress.
var activeBuildStatusFilter = build.BuildStatus(strings.Join([]string{
	string(build.BuildStatusValues.NotStarted),
	string(build.BuildStatusValues.Postponed),
	string(build.BuildStatusValues.InProgress),
	string(build.BuildStatusValues.Cancelling),
}, ","))

func (c *Client) ListPipelineDefinitions(ctx context.Context, projectID string, repoID string) ([]build.BuildDefinitionReference, error) {
	repositoryType := azureReposGitRepositoryType
	var definitions []build.BuildDefinitionReference
	continuationToken := ""

	for {
		args := build.GetDefinitionsArgs{
			Project:             &projectID,
			RepositoryId:        &repoID,
			RepositoryType:      &repositoryType,
			IncludeLatestBuilds: boolPtr(true),
		}
		if continuationToken != "" {
			args.ContinuationToken = &continuationToken
		}

		response, err := c.buildClient.GetDefinitions(ctx, args)
		if err != nil {
			return nil, fmt.Errorf("failed to list pipelines for repo '%s' in project '%s': %w", repoID, projectID, err)
		}
		if response == nil {
			return definitions, nil
		}

		definitions = append(definitions, response.Value...)
		if response.ContinuationToken == "" {
			return definitions, nil
		}
		continuationToken = response.ContinuationToken
	}
}

func (c *Client) ListBuilds(ctx context.Context, project string, definitionID int) ([]build.Build, error) {
	queryOrder := build.BuildQueryOrderValues.QueueTimeDescending
	response, err := c.buildClient.GetBuilds(ctx, build.GetBuildsArgs{
		Project:     &project,
		Definitions: &[]int{definitionID},
		Top:         intPtr(pipelineRunsPageSize),
		QueryOrder:  &queryOrder,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list runs for pipeline %d in project '%s': %w", definitionID, project, err)
	}
	if response == nil {
		return []build.Build{}, nil
	}
	return response.Value, nil
}

// ListActiveBuildsRequestedFor lists the project's unfinished builds requested for the user, which
// is a user ID or a unique name.
func (c *Client) ListActiveBuildsRequestedFor(ctx context.Context, project string, requestedFor string) ([]build.Build, error) {
	queryOrder := build.BuildQueryOrderValues.QueueTimeDescending
	statusFilter := activeBuildStatusFilter
	response, err := c.buildClient.GetBuilds(ctx, build.GetBuildsArgs{
		Project:      &project,
		RequestedFor: &requestedFor,
		StatusFilter: &statusFilter,
		Top:          intPtr(pipelineRunsPageSize),
		QueryOrder:   &queryOrder,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list active runs in project '%s': %w", project, err)
	}
	if response == nil {
		return []build.Build{}, nil
	}
	return response.Value, nil
}

func (c *Client) GetBuild(ctx context.Context, project string, buildID int) (*build.Build, error) {
	b, err := c.buildClient.GetBuild(ctx, build.GetBuildArgs{
		Project: &project,
		BuildId: &buildID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get run %d in project '%s': %w", buildID, project, err)
	}
	return b, nil
}

func (c *Client) GetBuildTimeline(ctx context.Context, project string, buildID int) (*build.Timeline, error) {
	timeline, err := c.buildClient.GetBuildTimeline(ctx, build.GetBuildTimelineArgs{
		Project: &project,
		BuildId: &buildID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline for run %d in project '%s': %w", buildID, project, err)
	}
	return timeline, nil
}

func (c *Client) UpdateApproval(ctx context.Context, project string, approvalID string, status pipelinesapproval.ApprovalStatus, comment string) error {
	parsedID, err := uuid.Parse(approvalID)
	if err != nil {
		return fmt.Errorf("invalid approval ID '%s': %w", approvalID, err)
	}

	updates := []pipelinesapproval.ApprovalUpdateParameters{
		{
			ApprovalId: &parsedID,
			Status:     &status,
			Comment:    &comment,
		},
	}

	if _, err := c.approvalClient.UpdateApprovals(ctx, pipelinesapproval.UpdateApprovalsArgs{
		Project:          &project,
		UpdateParameters: &updates,
	}); err != nil {
		return fmt.Errorf("failed to update approval %s in project '%s': %w", approvalID, project, err)
	}
	return nil
}

// requestedForUser prefers the user's ID; when it cannot be resolved the PAT's username is used,
// which the builds API also matches against the requester's unique name.
func (c *Client) requestedForUser(ctx context.Context) string {
	userID, err := c.GetAuthenticatedUserID(ctx)
	if err != nil {
		logger.Log("AzureDevOps: Filtering runs by username %s, user ID unavailable: %v", c.username, err)
		return c.username
	}
	return userID
}
