package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v57/github"
)

const listPageSize = 100

// ListEventsSince pages through the user's recent events, newest first, until they are older than
// since or GitHub has no more (it keeps at most 300).
func (c *Client) ListEventsSince(ctx context.Context, since time.Time) ([]*github.Event, error) {
	username, err := c.GetUsername(ctx)
	if err != nil {
		return nil, err
	}

	var events []*github.Event
	opts := &github.ListOptions{PerPage: listPageSize}
	for {
		page, resp, err := c.client.Activity.ListEventsPerformedByUser(ctx, username, false, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list events: %w", err)
		}
		events = append(events, page...)
		if len(page) == 0 || page[len(page)-1].GetCreatedAt().Before(since) || resp.NextPage == 0 {
			return events, nil
		}
		opts.Page = resp.NextPage
	}
}

func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	repository, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	return repository, nil
}

func (c *Client) ListBranches(ctx context.Context, owner, repo string) ([]*github.Branch, error) {
	var branches []*github.Branch
	opts := &github.BranchListOptions{ListOptions: github.ListOptions{PerPage: listPageSize}}
	for {
		page, resp, err := c.client.Repositories.ListBranches(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list branches: %w", err)
		}
		branches = append(branches, page...)
		if resp.NextPage == 0 {
			return branches, nil
		}
		opts.Page = resp.NextPage
	}
}

func (c *Client) CreatePullRequest(ctx context.Context, owner, repo string, pr *github.NewPullRequest) (*github.PullRequest, error) {
	created, _, err := c.client.PullRequests.Create(ctx, owner, repo, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}
	return created, nil
}
