package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v57/github"
)

const openIssueState = "open"

func (c *Client) GetIssue(ctx context.Context, owner, repo string, number int) (*github.Issue, error) {
	issue, _, err := c.client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue #%d: %w", number, err)
	}
	return issue, nil
}

// ListAssignedOpenIssues returns the repository's open issues assigned to the user. The issues API
// lists pull requests too; they are left in for the caller to drop.
func (c *Client) ListAssignedOpenIssues(ctx context.Context, owner, repo string) ([]*github.Issue, error) {
	username, err := c.GetUsername(ctx)
	if err != nil {
		return nil, err
	}
	var issues []*github.Issue
	opts := &github.IssueListByRepoOptions{
		Assignee:    username,
		State:       openIssueState,
		ListOptions: github.ListOptions{PerPage: listPageSize},
	}
	for {
		page, resp, err := c.client.Issues.ListByRepo(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list assigned issues: %w", err)
		}
		issues = append(issues, page...)
		if resp.NextPage == 0 {
			return issues, nil
		}
		opts.Page = resp.NextPage
	}
}
