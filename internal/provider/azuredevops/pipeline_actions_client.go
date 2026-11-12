package azuredevops

import (
	"context"
	"fmt"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

// GetBuildLogLines reads a log after its first offset lines. Log lines are numbered from 1.
func (c *Client) GetBuildLogLines(ctx context.Context, project string, buildID, logID, offset int) ([]string, error) {
	args := build.GetBuildLogLinesArgs{
		Project: &project,
		BuildId: &buildID,
		LogId:   &logID,
	}
	if offset > 0 {
		startLine := uint64(offset + 1)
		args.StartLine = &startLine
	}

	lines, err := c.buildClient.GetBuildLogLines(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to read log %d of run %d in project '%s': %w", logID, buildID, project, err)
	}
	if lines == nil {
		return []string{}, nil
	}
	return *lines, nil
}

func (c *Client) GetBuildDefinition(ctx context.Context, project string, definitionID int) (*build.BuildDefinition, error) {
	definition, err := c.buildClient.GetDefinition(ctx, build.GetDefinitionArgs{
		Project:      &project,
		DefinitionId: &definitionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline %d in project '%s': %w", definitionID, project, err)
	}
	if definition == nil {
		return nil, fmt.Errorf("pipeline %d not found in project '%s'", definitionID, project)
	}
	return definition, nil
}

// QueueBuild starts a new build of the definition, with the branch, commit and parameters set on b.
func (c *Client) QueueBuild(ctx context.Context, project string, definitionID int, b build.Build) (*build.Build, error) {
	b.Definition = &build.DefinitionReference{Id: &definitionID}
	queued, err := c.buildClient.QueueBuild(ctx, build.QueueBuildArgs{
		Project: &project,
		Build:   &b,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to queue pipeline %d in project '%s': %w", definitionID, project, err)
	}
	if queued == nil {
		return nil, fmt.Errorf("queueing pipeline %d in project '%s' returned no run", definitionID, project)
	}
	return queued, nil
}

// RetryBuild reruns the failed jobs of a finished build in the same build.
func (c *Client) RetryBuild(ctx context.Context, project string, buildID int) (*build.Build, error) {
	retried, err := c.buildClient.UpdateBuild(ctx, build.UpdateBuildArgs{
		Project: &project,
		BuildId: &buildID,
		Build:   &build.Build{},
		Retry:   boolPtr(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retry run %d in project '%s': %w", buildID, project, err)
	}
	return retried, nil
}

// RetryStage reruns the failed jobs of one stage of a build.
func (c *Client) RetryStage(ctx context.Context, project string, buildID int, stageRefName string) error {
	state := build.StageUpdateTypeValues.Retry
	if err := c.buildClient.UpdateStage(ctx, build.UpdateStageArgs{
		Project:      &project,
		BuildId:      &buildID,
		StageRefName: &stageRefName,
		UpdateParameters: &build.UpdateStageParameters{
			State:             &state,
			ForceRetryAllJobs: boolPtr(false),
		},
	}); err != nil {
		return fmt.Errorf("failed to retry stage '%s' of run %d in project '%s': %w", stageRefName, buildID, project, err)
	}
	return nil
}

// GetFileContentAtBranch reads a file of a repository as it is on a branch.
func (c *Client) GetFileContentAtBranch(ctx context.Context, project, repoID, path, branch string) (string, error) {
	versionType := git.GitVersionTypeValues.Branch
	item, err := c.gitClient.GetItem(ctx, git.GetItemArgs{
		RepositoryId:      &repoID,
		Project:           &project,
		Path:              &path,
		IncludeContent:    boolPtr(true),
		VersionDescriptor: &git.GitVersionDescriptor{Version: &branch, VersionType: &versionType},
	})
	if err != nil {
		return "", fmt.Errorf("failed to read '%s' on branch '%s': %w", path, branch, err)
	}
	if item == nil || item.Content == nil {
		return "", fmt.Errorf("'%s' on branch '%s' has no content", path, branch)
	}
	return *item.Content, nil
}
