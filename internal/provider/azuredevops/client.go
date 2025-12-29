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

func intPtr(i int) *int {
	return &i
}
