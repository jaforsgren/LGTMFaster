package azuredevops

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

const branchRefFilter = "heads/"

// ListRecentPushes returns the pushes the user made to a repository since the given time, with the
// refs each one updated.
func (c *Client) ListRecentPushes(ctx context.Context, projectID, repoID string, since time.Time) ([]git.GitPush, error) {
	userID, err := c.GetAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	pusherID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID '%s': %w", userID, err)
	}

	pushes, err := c.gitClient.GetPushes(ctx, git.GetPushesArgs{
		RepositoryId: &repoID,
		Project:      &projectID,
		SearchCriteria: &git.GitPushSearchCriteria{
			FromDate:          &azuredevops.Time{Time: since},
			PusherId:          &pusherID,
			IncludeRefUpdates: boolPtr(true),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pushes for repo '%s' in project '%s': %w", repoID, projectID, err)
	}
	if pushes == nil {
		return nil, nil
	}
	return *pushes, nil
}

// ListBranchRefs returns every branch ref of a repository, following the continuation token.
func (c *Client) ListBranchRefs(ctx context.Context, projectID, repoID string) ([]git.GitRef, error) {
	filter := branchRefFilter
	var refs []git.GitRef
	continuationToken := ""
	for {
		args := git.GetRefsArgs{RepositoryId: &repoID, Project: &projectID, Filter: &filter}
		if continuationToken != "" {
			args.ContinuationToken = &continuationToken
		}
		page, err := c.gitClient.GetRefs(ctx, args)
		if err != nil {
			return nil, fmt.Errorf("failed to list branches for repo '%s' in project '%s': %w", repoID, projectID, err)
		}
		if page == nil {
			return refs, nil
		}
		refs = append(refs, page.Value...)
		if page.ContinuationToken == "" {
			return refs, nil
		}
		continuationToken = page.ContinuationToken
	}
}

func (c *Client) GetRepository(ctx context.Context, projectID, repoID string) (*git.GitRepository, error) {
	repo, err := c.gitClient.GetRepository(ctx, git.GetRepositoryArgs{RepositoryId: &repoID, Project: &projectID})
	if err != nil {
		return nil, fmt.Errorf("failed to get repo '%s' in project '%s': %w", repoID, projectID, err)
	}
	return repo, nil
}

func (c *Client) CreatePullRequest(ctx context.Context, projectID, repoID string, pr *git.GitPullRequest) (*git.GitPullRequest, error) {
	created, err := c.gitClient.CreatePullRequest(ctx, git.CreatePullRequestArgs{
		GitPullRequestToCreate: pr,
		RepositoryId:           &repoID,
		Project:                &projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request in repo '%s': %w", repoID, err)
	}
	return created, nil
}
