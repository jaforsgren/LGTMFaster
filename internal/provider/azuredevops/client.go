package azuredevops

import (
	"context"
	"fmt"
	"strings"

	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

type Client struct {
	connection   *azuredevops.Connection
	coreClient   core.Client
	gitClient    GitClientInterface
	organization string
	username     string
	userID       string
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

	client := &Client{
		connection:   connection,
		coreClient:   coreClient,
		gitClient:    gitClient,
		organization: organization,
		username:     username,
	}

	userID, err := client.getAuthenticatedUserID(context.Background())
	if err != nil {
		logger.Log("AzureDevOps: Warning - Could not determine user ID during initialization: %v", err)
		logger.Log("AzureDevOps: User ID will be resolved when needed for review submission")
	} else {
		client.userID = userID
		logger.Log("AzureDevOps: Authenticated user ID: %s (username: %s)", userID, username)
	}

	return client, nil
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
	if c.userID != "" {
		return c.userID, nil
	}

	logger.Log("AzureDevOps: User ID not cached, attempting lazy resolution...")
	userID, err := c.getAuthenticatedUserID(ctx)
	if err != nil {
		return "", err
	}

	c.userID = userID
	logger.Log("AzureDevOps: Successfully resolved user ID: %s", userID)
	return c.userID, nil
}

func (c *Client) getAuthenticatedUserID(ctx context.Context) (string, error) {
	projects, err := c.coreClient.GetProjects(ctx, core.GetProjectsArgs{
		Top: intPtr(10),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get projects: %w", err)
	}

	if projects == nil || len(projects.Value) == 0 {
		return "", fmt.Errorf("no projects found to determine user identity")
	}

	statuses := []git.PullRequestStatus{
		git.PullRequestStatusValues.Active,
		git.PullRequestStatusValues.Completed,
		git.PullRequestStatusValues.Abandoned,
	}

	for _, project := range projects.Value {
		projectIDStr := project.Id.String()

		repos, err := c.gitClient.GetRepositories(ctx, git.GetRepositoriesArgs{
			Project: &projectIDStr,
		})
		if err != nil || repos == nil || len(*repos) == 0 {
			continue
		}

		for _, repo := range *repos {
			repoIDStr := repo.Id.String()

			for _, status := range statuses {
				prs, err := c.gitClient.GetPullRequests(ctx, git.GetPullRequestsArgs{
					RepositoryId: &repoIDStr,
					Project:      &projectIDStr,
					SearchCriteria: &git.GitPullRequestSearchCriteria{
						Status: &status,
					},
					Top: intPtr(100),
				})
				if err != nil || prs == nil {
					continue
				}

				for _, pr := range *prs {
					if pr.CreatedBy != nil && pr.CreatedBy.Id != nil && pr.CreatedBy.UniqueName != nil {
						if *pr.CreatedBy.UniqueName == c.username {
							logger.Log("AzureDevOps: Found user ID %s from PR creator in %s/%s", *pr.CreatedBy.Id, *project.Name, *repo.Name)
							return *pr.CreatedBy.Id, nil
						}
					}

					if pr.Reviewers != nil {
						for _, reviewer := range *pr.Reviewers {
							if reviewer.UniqueName != nil && reviewer.Id != nil {
								if *reviewer.UniqueName == c.username {
									logger.Log("AzureDevOps: Found user ID %s from PR reviewer in %s/%s", *reviewer.Id, *project.Name, *repo.Name)
									return *reviewer.Id, nil
								}
							}
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("unable to determine user ID from Azure DevOps - searched all projects/repos but found no PRs created by or reviewed by user %s", c.username)
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

func (c *Client) GetPullRequestIterationChanges(ctx context.Context, projectID string, repoID string, pullRequestID int) (string, error) {
	iterations, err := c.gitClient.GetPullRequestIterations(ctx, git.GetPullRequestIterationsArgs{
		RepositoryId:  &repoID,
		PullRequestId: &pullRequestID,
		Project:       &projectID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get PR iterations: %w", err)
	}

	if iterations == nil || len(*iterations) == 0 {
		return "", fmt.Errorf("no iterations found for PR")
	}

	latestIteration := (*iterations)[len(*iterations)-1]
	if latestIteration.Id == nil {
		return "", fmt.Errorf("latest iteration has no ID")
	}

	changes, err := c.gitClient.GetPullRequestIterationChanges(ctx, git.GetPullRequestIterationChangesArgs{
		RepositoryId:  &repoID,
		PullRequestId: &pullRequestID,
		IterationId:   latestIteration.Id,
		Project:       &projectID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get PR iteration changes: %w", err)
	}

	if changes == nil || changes.ChangeEntries == nil || len(*changes.ChangeEntries) == 0 {
		return "", fmt.Errorf("no changes found in latest iteration")
	}

	diffText := ""
	for _, change := range *changes.ChangeEntries {
		itemMap, ok := change.Item.(map[string]interface{})
		if !ok || itemMap == nil {
			continue
		}

		path, _ := itemMap["path"].(string)
		if path == "" {
			continue
		}

		isFolder, _ := itemMap["isFolder"].(bool)
		if isFolder {
			continue
		}

		objectId, _ := itemMap["objectId"].(string)

		originalObjectId := ""
		if originalPath, ok := itemMap["originalPath"]; ok && originalPath != nil {
			originalObjectId, _ = itemMap["originalObjectId"].(string)
		}

		changeType := 0
		if change.ChangeType != nil {
			changeTypeStr := string(*change.ChangeType)
			switch changeTypeStr {
			case "add", "1":
				changeType = 1
			case "edit", "2":
				changeType = 2
			case "delete", "16":
				changeType = 16
			}
		}

		const (
			changeTypeAdd    = 1
			changeTypeEdit   = 2
			changeTypeDelete = 16
		)

		switch changeType {
		case changeTypeAdd:
			if objectId == "" {
				continue
			}
			content, err := c.getFileContent(ctx, projectID, repoID, objectId)
			if err != nil {
				continue
			}
			diffText += fmt.Sprintf("diff --git a%s b%s\n", path, path)
			diffText += "--- /dev/null\n"
			diffText += fmt.Sprintf("+++ b%s\n", path)
			diffText += fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(content))
			for _, line := range content {
				diffText += "+" + line + "\n"
			}

		case changeTypeDelete:
			if originalObjectId == "" {
				continue
			}
			content, err := c.getFileContent(ctx, projectID, repoID, originalObjectId)
			if err != nil {
				continue
			}
			diffText += fmt.Sprintf("diff --git a%s b%s\n", path, path)
			diffText += fmt.Sprintf("--- a%s\n", path)
			diffText += "+++ /dev/null\n"
			diffText += fmt.Sprintf("@@ -1,%d +0,0 @@\n", len(content))
			for _, line := range content {
				diffText += "-" + line + "\n"
			}

		case changeTypeEdit:
			if objectId == "" || originalObjectId == "" {
				continue
			}

			newContent, err := c.getFileContent(ctx, projectID, repoID, objectId)
			if err != nil {
				continue
			}
			oldContent, err := c.getFileContent(ctx, projectID, repoID, originalObjectId)
			if err != nil {
				continue
			}

			diffText += fmt.Sprintf("diff --git a%s b%s\n", path, path)
			diffText += fmt.Sprintf("--- a%s\n", path)
			diffText += fmt.Sprintf("+++ b%s\n", path)

			unifiedDiff := generateUnifiedDiff(oldContent, newContent)
			diffText += unifiedDiff
		}
	}

	return diffText, nil
}

func (c *Client) getFileContent(ctx context.Context, projectID string, repoID string, objectId string) ([]string, error) {
	stream, err := c.gitClient.GetBlobContent(ctx, git.GetBlobContentArgs{
		RepositoryId: &repoID,
		Sha1:         &objectId,
		Project:      &projectID,
	})
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	content := make([]byte, 0, 4096)
	buf := make([]byte, 1024)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			content = append(content, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	text := string(content)
	lines := strings.Split(text, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}

func generateUnifiedDiff(oldLines, newLines []string) string {
	return generateSimpleDiff(oldLines, newLines)
}

func generateSimpleDiff(oldLines, newLines []string) string {
	result := fmt.Sprintf("@@ -1,%d +1,%d @@\n", len(oldLines), len(newLines))

	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	for i := 0; i < maxLen; i++ {
		if i < len(oldLines) && i < len(newLines) {
			if oldLines[i] == newLines[i] {
				result += " " + oldLines[i] + "\n"
			} else {
				result += "-" + oldLines[i] + "\n"
				result += "+" + newLines[i] + "\n"
			}
		} else if i < len(oldLines) {
			result += "-" + oldLines[i] + "\n"
		} else {
			result += "+" + newLines[i] + "\n"
		}
	}

	return result
}

func boolPtr(b bool) *bool {
	return &b
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
