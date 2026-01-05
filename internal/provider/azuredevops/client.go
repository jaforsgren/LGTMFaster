package azuredevops

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aymanbagabas/go-udiff"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

type Client struct {
	connection   *azuredevops.Connection
	coreClient   core.Client
	gitClient    git.Client
	organization string
	username     string
}

func NewClient(token string, organization string, username string) (*Client, error) {
	organizationURL := fmt.Sprintf("https://dev.azure.com/%s", organization)
	connection := azuredevops.NewPatConnection(organizationURL, token)

	coreClient, err := core.NewClient(context.Background(), connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create core client: %w", err)
	}

	gitClient, err := git.NewClient(context.Background(), connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create git client: %w", err)
	}

	return &Client{
		connection:   connection,
		coreClient:   coreClient,
		gitClient:    gitClient,
		organization: organization,
		username:     username,
	}, nil
}

func (c *Client) ValidateCredentials(ctx context.Context) error {
	projects, err := c.coreClient.GetProjects(ctx, core.GetProjectsArgs{
		Top: intPtr(1),
	})
	if err != nil {
		return fmt.Errorf("failed to validate credentials: %w", err)
	}
	if projects == nil {
		return fmt.Errorf("failed to validate credentials: no response")
	}
	return nil
}

func (c *Client) GetAuthenticatedUserID(ctx context.Context) (string, error) {
	connectionData := c.connection.AuthorizationString
	return connectionData, nil
}

func (c *Client) ListProjects(ctx context.Context) (*[]core.TeamProjectReference, error) {
	response, err := c.coreClient.GetProjects(ctx, core.GetProjectsArgs{})
	if err != nil {
		return nil, fmt.Errorf("failed to list projects for organization '%s': %w", c.organization, err)
	}
	if response == nil {
		return &[]core.TeamProjectReference{}, nil
	}
	return &response.Value, nil
}

func (c *Client) ListRepositories(ctx context.Context, projectID string) (*[]git.GitRepository, error) {
	repos, err := c.gitClient.GetRepositories(ctx, git.GetRepositoriesArgs{
		Project: &projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories for project '%s': %w", projectID, err)
	}
	return repos, nil
}

func (c *Client) ListPullRequests(ctx context.Context, projectID string, repoID string) (*[]git.GitPullRequest, error) {
	status := git.PullRequestStatusValues.Active

	prs, err := c.gitClient.GetPullRequests(ctx, git.GetPullRequestsArgs{
		RepositoryId: &repoID,
		Project:      &projectID,
		SearchCriteria: &git.GitPullRequestSearchCriteria{
			Status: &status,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests for repo '%s' in project '%s': %w", repoID, projectID, err)
	}
	return prs, nil
}

func (c *Client) GetPullRequest(ctx context.Context, projectID string, repoID string, pullRequestID int) (*git.GitPullRequest, error) {
	pr, err := c.gitClient.GetPullRequest(ctx, git.GetPullRequestArgs{
		RepositoryId:  &repoID,
		PullRequestId: &pullRequestID,
		Project:       &projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request %d in repo '%s' project '%s': %w", pullRequestID, repoID, projectID, err)
	}
	return pr, nil
}

func (c *Client) GetPullRequestCommits(ctx context.Context, projectID string, repoID string, pullRequestID int) (*[]git.GitCommitRef, error) {
	response, err := c.gitClient.GetPullRequestCommits(ctx, git.GetPullRequestCommitsArgs{
		RepositoryId:  &repoID,
		PullRequestId: &pullRequestID,
		Project:       &projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get commits for PR %d: %w", pullRequestID, err)
	}
	if response == nil {
		return &[]git.GitCommitRef{}, nil
	}
	return &response.Value, nil
}

func (c *Client) GetCommitDiffs(ctx context.Context, projectID string, repoID string, baseCommit string, targetCommit string) (string, error) {
	logger.Log("AzureDevOps: GetCommitDiffs called with project=%s, repo=%s, base=%s, target=%s", projectID, repoID, baseCommit, targetCommit)

	baseVersionType := git.GitVersionTypeValues.Commit
	targetVersionType := git.GitVersionTypeValues.Commit

	diffs, err := c.gitClient.GetCommitDiffs(ctx, git.GetCommitDiffsArgs{
		RepositoryId: &repoID,
		Project:      &projectID,
		BaseVersionDescriptor: &git.GitBaseVersionDescriptor{
			BaseVersion:     &baseCommit,
			BaseVersionType: &baseVersionType,
		},
		TargetVersionDescriptor: &git.GitTargetVersionDescriptor{
			TargetVersion:     &targetCommit,
			TargetVersionType: &targetVersionType,
		},
	})
	if err != nil {
		logger.LogError("AZDO_GET_COMMIT_DIFFS", "API_CALL", err)
		return "", fmt.Errorf("failed to get commit diffs: %w", err)
	}

	logger.Log("AzureDevOps: GetCommitDiffs response - diffs=%v, changes=%v", diffs != nil, diffs != nil && diffs.Changes != nil)

	if diffs == nil {
		logger.Log("AzureDevOps: diffs is nil, returning empty")
		return "", nil
	}

	if diffs.Changes == nil {
		logger.Log("AzureDevOps: diffs.Changes is nil, returning empty")
		return "", nil
	}

	changeCount := len(*diffs.Changes)
	logger.Log("AzureDevOps: Found %d changes in diff", changeCount)

	if changeCount == 0 {
		logger.Log("AzureDevOps: No changes found, returning empty")
		return "", nil
	}

	var diffText strings.Builder
	processedCount := 0

	for i, change := range *diffs.Changes {
		logger.Log("AzureDevOps: Processing change %d/%d: %+v", i+1, changeCount, change)

		changeMap, ok := change.(map[string]interface{})
		if !ok {
			logger.Log("AzureDevOps: Change %d is not a map[string]interface{}, skipping", i+1)
			continue
		}

		item, hasItem := changeMap["item"].(map[string]interface{})
		if !hasItem {
			logger.Log("AzureDevOps: Change %d has no item field, skipping", i+1)
			continue
		}

		path, hasPath := item["path"].(string)
		if !hasPath {
			logger.Log("AzureDevOps: Change %d item has no path, skipping", i+1)
			continue
		}

		gitObjectType, _ := item["gitObjectType"].(string)
		if gitObjectType != "blob" {
			logger.Log("AzureDevOps: Change %d - path=%s is not a blob (type=%s), skipping", i+1, path, gitObjectType)
			continue
		}

		objectID, hasObjectID := item["objectId"].(string)
		changeTypeStr, _ := changeMap["changeType"].(string)

		logger.Log("AzureDevOps: Change %d - path=%s, changeType=%s, objectID=%s, hasObjectID=%v",
			i+1, path, changeTypeStr, objectID, hasObjectID)

		var oldContent, newContent string

		switch changeTypeStr {
		case "add":
			logger.Log("AzureDevOps: Processing ADD for %s", path)
			oldContent = ""
			if hasObjectID && objectID != "" {
				content, err := c.getFileContent(ctx, projectID, repoID, path, targetCommit)
				if err == nil {
					newContent = content
					logger.Log("AzureDevOps: Fetched new content for %s (%d bytes)", path, len(newContent))
				} else {
					logger.LogError("AZDO_GET_FILE_CONTENT", path, err)
				}
			}
		case "delete":
			logger.Log("AzureDevOps: Processing DELETE for %s", path)
			if hasObjectID && objectID != "" {
				content, err := c.getFileContent(ctx, projectID, repoID, path, baseCommit)
				if err == nil {
					oldContent = content
					logger.Log("AzureDevOps: Fetched old content for %s (%d bytes)", path, len(oldContent))
				} else {
					logger.LogError("AZDO_GET_FILE_CONTENT", path, err)
				}
			}
			newContent = ""
		case "edit":
			logger.Log("AzureDevOps: Processing EDIT for %s", path)
			oldC, err1 := c.getFileContent(ctx, projectID, repoID, path, baseCommit)
			newC, err2 := c.getFileContent(ctx, projectID, repoID, path, targetCommit)
			if err1 == nil && err2 == nil {
				oldContent = oldC
				newContent = newC
				logger.Log("AzureDevOps: Fetched old (%d bytes) and new (%d bytes) content for %s",
					len(oldContent), len(newContent), path)
			} else {
				if err1 != nil {
					logger.LogError("AZDO_GET_FILE_CONTENT_OLD", path, err1)
				}
				if err2 != nil {
					logger.LogError("AZDO_GET_FILE_CONTENT_NEW", path, err2)
				}
			}
		default:
			logger.Log("AzureDevOps: Unknown changeType '%s' for %s, skipping", changeTypeStr, path)
			continue
		}

		diff := udiff.Unified(
			fmt.Sprintf("a%s", path),
			fmt.Sprintf("b%s", path),
			oldContent,
			newContent,
		)

		logger.Log("AzureDevOps: Generated diff for %s (%d bytes)", path, len(diff))

		diffText.WriteString(fmt.Sprintf("diff --git a%s b%s\n", path, path))
		diffText.WriteString(diff)
		processedCount++
	}

	result := diffText.String()
	logger.Log("AzureDevOps: GetCommitDiffs complete - processed %d/%d changes, total diff size: %d bytes",
		processedCount, changeCount, len(result))

	return result, nil
}

func (c *Client) getFileContent(ctx context.Context, projectID string, repoID string, path string, commitID string) (string, error) {
	logger.Log("AzureDevOps: getFileContent called for path=%s, commit=%s", path, commitID)

	versionType := git.GitVersionTypeValues.Commit
	reader, err := c.gitClient.GetItemText(ctx, git.GetItemTextArgs{
		RepositoryId: &repoID,
		Project:      &projectID,
		Path:         &path,
		VersionDescriptor: &git.GitVersionDescriptor{
			Version:     &commitID,
			VersionType: &versionType,
		},
	})
	if err != nil {
		logger.LogError("AZDO_GET_ITEM_TEXT", fmt.Sprintf("%s@%s", path, commitID), err)
		return "", fmt.Errorf("failed to get file content for %s at %s: %w", path, commitID, err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		logger.LogError("AZDO_READ_CONTENT", path, err)
		return "", fmt.Errorf("failed to read file content: %w", err)
	}

	logger.Log("AzureDevOps: getFileContent success for %s (%d bytes)", path, len(content))
	return string(content), nil
}

func (c *Client) GetPullRequestThreads(ctx context.Context, projectID string, repoID string, pullRequestID int) (*[]git.GitPullRequestCommentThread, error) {
	threads, err := c.gitClient.GetThreads(ctx, git.GetThreadsArgs{
		RepositoryId:  &repoID,
		PullRequestId: &pullRequestID,
		Project:       &projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get threads for PR %d: %w", pullRequestID, err)
	}
	return threads, nil
}

func (c *Client) CreateCommentThread(ctx context.Context, projectID string, repoID string, pullRequestID int, body string, filePath string, line int) error {
	thread := git.GitPullRequestCommentThread{
		Comments: &[]git.Comment{
			{
				Content:     &body,
				CommentType: &git.CommentTypeValues.Text,
			},
		},
	}

	if filePath != "" && line > 0 {
		thread.ThreadContext = &git.CommentThreadContext{
			FilePath: &filePath,
			RightFileStart: &git.CommentPosition{
				Line:   &line,
				Offset: intPtr(1),
			},
			RightFileEnd: &git.CommentPosition{
				Line:   &line,
				Offset: intPtr(1),
			},
		}
	}

	_, err := c.gitClient.CreateThread(ctx, git.CreateThreadArgs{
		CommentThread: &thread,
		RepositoryId:  &repoID,
		PullRequestId: &pullRequestID,
		Project:       &projectID,
	})
	if err != nil {
		return fmt.Errorf("failed to create comment thread: %w", err)
	}
	return nil
}

func (c *Client) CreatePullRequestReview(ctx context.Context, projectID string, repoID string, pullRequestID int, reviewerID string, vote int) error {
	reviewer := git.IdentityRefWithVote{
		Vote: &vote,
	}

	_, err := c.gitClient.CreatePullRequestReviewer(ctx, git.CreatePullRequestReviewerArgs{
		Reviewer:      &reviewer,
		RepositoryId:  &repoID,
		PullRequestId: &pullRequestID,
		ReviewerId:    &reviewerID,
		Project:       &projectID,
	})
	if err != nil {
		return fmt.Errorf("failed to create review: %w", err)
	}
	return nil
}

func intPtr(i int) *int {
	return &i
}
