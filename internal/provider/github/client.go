package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

type Client struct {
	client   *github.Client
	username string
}

func NewClient(token string, username string) *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	return &Client{
		client:   client,
		username: username,
	}
}

func (c *Client) GetUsername(ctx context.Context) (string, error) {
	if c.username != "" {
		return c.username, nil
	}

	user, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if user.Login != nil {
		c.username = *user.Login
	}

	return c.username, nil
}

func (c *Client) ListPullRequests(ctx context.Context) ([]*github.PullRequest, error) {
	username, err := c.GetUsername(ctx)
	if err != nil {
		return nil, err
	}

	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	query := fmt.Sprintf("is:pr is:open involves:%s", username)
	result, _, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search pull requests: %w", err)
	}

	prs := make([]*github.PullRequest, 0, len(result.Issues))
	for _, issue := range result.Issues {
		if issue.PullRequestLinks == nil {
			continue
		}

		parts := strings.Split(*issue.RepositoryURL, "/")
		if len(parts) < 2 {
			continue
		}
		owner := parts[len(parts)-2]
		repo := parts[len(parts)-1]

		pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, *issue.Number)
		if err != nil {
			continue
		}

		prs = append(prs, pr)
	}

	return prs, nil
}

func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}
	return pr, nil
}

func (c *Client) GetDiff(ctx context.Context, owner, repo string, number int) (string, error) {
	diff, _, err := c.client.PullRequests.GetRaw(ctx, owner, repo, number, github.RawOptions{Type: github.Diff})
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}
	return diff, nil
}

func (c *Client) ListComments(ctx context.Context, owner, repo string, number int) ([]*github.PullRequestComment, error) {
	opts := &github.PullRequestListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	comments, _, err := c.client.PullRequests.ListComments(ctx, owner, repo, number, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}

	return comments, nil
}

func (c *Client) CreateComment(ctx context.Context, owner, repo string, number int, comment *github.PullRequestComment) error {
	_, _, err := c.client.PullRequests.CreateComment(ctx, owner, repo, number, comment)
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}
	return nil
}

func (c *Client) CreateReview(ctx context.Context, owner, repo string, number int, review *github.PullRequestReviewRequest) error {
	_, _, err := c.client.PullRequests.CreateReview(ctx, owner, repo, number, review)
	if err != nil {
		return fmt.Errorf("failed to create review: %w", err)
	}
	return nil
}
