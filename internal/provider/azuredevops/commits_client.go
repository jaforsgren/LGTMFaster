package azuredevops

import (
	"context"
	"fmt"

	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

// GetCommitDiff renders a commit's changes against its first parent as a unified diff.
func (c *Client) GetCommitDiff(ctx context.Context, projectID, repoID, commitID string) (string, error) {
	changes, err := c.gitClient.GetChanges(ctx, git.GetChangesArgs{CommitId: &commitID, RepositoryId: &repoID, Project: &projectID})
	if err != nil {
		return "", fmt.Errorf("failed to get changes of commit %s: %w", commitID, err)
	}
	if changes == nil || changes.Changes == nil {
		return "", nil
	}

	diffText := ""
	parentID := ""
	for _, raw := range *changes.Changes {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		changeType, _ := entry["changeType"].(string)
		file, ok := parseFileChange(changeType, entry["item"])
		if !ok {
			continue
		}
		if file.changeType != changeTypeAdd && file.originalObjectID == "" {
			if parentID == "" {
				if parentID, err = c.firstParent(ctx, projectID, repoID, commitID); err != nil {
					return "", err
				}
			}
			file.originalObjectID = c.objectIDAt(ctx, projectID, repoID, file.path, parentID)
		}
		fileDiff, err := c.fileDiffText(ctx, projectID, repoID, file)
		if err != nil {
			logger.LogError("AZURE_COMMIT_FILE_DIFF", file.path, err)
			continue
		}
		diffText += fileDiff
	}
	return diffText, nil
}

func (c *Client) GetCommitComment(ctx context.Context, projectID, repoID, commitID string) (string, error) {
	commit, err := c.gitClient.GetCommit(ctx, git.GetCommitArgs{CommitId: &commitID, RepositoryId: &repoID, Project: &projectID})
	if err != nil {
		return "", fmt.Errorf("failed to get commit %s: %w", commitID, err)
	}
	if commit == nil {
		return "", fmt.Errorf("commit %s not found in repository '%s'", commitID, repoID)
	}
	return common.GetString(commit.Comment), nil
}

func (c *Client) firstParent(ctx context.Context, projectID, repoID, commitID string) (string, error) {
	commit, err := c.gitClient.GetCommit(ctx, git.GetCommitArgs{CommitId: &commitID, RepositoryId: &repoID, Project: &projectID})
	if err != nil {
		return "", fmt.Errorf("failed to get commit %s: %w", commitID, err)
	}
	if commit == nil || commit.Parents == nil || len(*commit.Parents) == 0 {
		return "", fmt.Errorf("commit %s has no parent to compare with", commitID)
	}
	return (*commit.Parents)[0], nil
}

// objectIDAt is the blob of a file at a commit, or "" when it cannot be read.
func (c *Client) objectIDAt(ctx context.Context, projectID, repoID, path, commitID string) string {
	versionType := git.GitVersionTypeValues.Commit
	item, err := c.gitClient.GetItem(ctx, git.GetItemArgs{
		RepositoryId:      &repoID,
		Project:           &projectID,
		Path:              &path,
		VersionDescriptor: &git.GitVersionDescriptor{Version: &commitID, VersionType: &versionType},
	})
	if err != nil || item == nil || item.ObjectId == nil {
		logger.LogError("AZURE_GET_ITEM", fmt.Sprintf("%s@%s", path, commitID), err)
		return ""
	}
	return *item.ObjectId
}
