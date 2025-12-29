package azuredevops

import (
	"context"
	"fmt"

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
	commits, err := c.gitClient.GetCommits(ctx, git.GetCommitsArgs{
		RepositoryId: &repoID,
		Project:      &projectID,
		SearchCriteria: &git.GitQueryCommitsCriteria{
			ItemVersion: &git.GitVersionDescriptor{
				Version:     &targetCommit,
				VersionType: &git.GitVersionTypeValues.Commit,
			},
			CompareVersion: &git.GitVersionDescriptor{
				Version:     &baseCommit,
				VersionType: &git.GitVersionTypeValues.Commit,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get commits: %w", err)
	}

	if commits == nil || len(*commits) == 0 {
		return "", nil
	}

	diffText := ""
	for _, commit := range *commits {
		if commit.CommitId == nil {
			continue
		}

		changes, err := c.gitClient.GetChanges(ctx, git.GetChangesArgs{
			CommitId:     commit.CommitId,
			RepositoryId: &repoID,
			Project:      &projectID,
		})
		if err != nil {
			continue
		}

		if changes == nil || changes.Changes == nil {
			continue
		}

		for _, change := range *changes.Changes {
			changeMap, ok := change.(map[string]interface{})
			if !ok {
				continue
			}

			item, hasItem := changeMap["item"].(map[string]interface{})
			if !hasItem {
				continue
			}

			path, hasPath := item["path"].(string)
			if !hasPath {
				continue
			}

			diffText += fmt.Sprintf("diff --git a%s b%s\n", path, path)

			if changeTypeStr, hasChangeType := changeMap["changeType"].(string); hasChangeType {
				switch changeTypeStr {
				case "add":
					diffText += fmt.Sprintf("--- /dev/null\n+++ b%s\n", path)
					diffText += "@@ -0,0 +1,1 @@\n"
					diffText += "+ (file added)\n"
				case "delete":
					diffText += fmt.Sprintf("--- a%s\n+++ /dev/null\n", path)
					diffText += "@@ -1,1 +0,0 @@\n"
					diffText += "- (file deleted)\n"
				case "edit":
					diffText += fmt.Sprintf("--- a%s\n+++ b%s\n", path, path)
					diffText += "@@ -1,1 +1,1 @@\n"
					diffText += "  (file modified)\n"
				default:
					diffText += fmt.Sprintf("--- a%s\n+++ b%s\n", path, path)
					diffText += "@@ -1,1 +1,1 @@\n"
					diffText += "  (file changed)\n"
				}
			}
		}
	}

	return diffText, nil
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
