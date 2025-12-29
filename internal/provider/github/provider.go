package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v57/github"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
)

type Provider struct {
	client   *Client
	username string
}

func NewProvider(token string, username string) *Provider {
	return &Provider{
		client:   NewClient(token, username),
		username: username,
	}
}

func (p *Provider) GetType() domain.ProviderType {
	return domain.ProviderGitHub
}

func (p *Provider) ListPullRequests(ctx context.Context, username string) ([]domain.PullRequest, error) {
	ghPRs, err := p.client.ListPullRequests(ctx)
	if err != nil {
		return nil, err
	}

	prs := make([]domain.PullRequest, 0, len(ghPRs))
	for _, ghPR := range ghPRs {
		pr := p.convertPullRequest(ghPR, username)
		prs = append(prs, pr)
	}

	return prs, nil
}

func (p *Provider) GetPullRequest(ctx context.Context, identifier domain.PRIdentifier) (*domain.PullRequest, error) {
	parts := strings.Split(identifier.Repository, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s", identifier.Repository)
	}

	owner, repo := parts[0], parts[1]
	ghPR, err := p.client.GetPullRequest(ctx, owner, repo, identifier.Number)
	if err != nil {
		return nil, err
	}

	pr := p.convertPullRequest(ghPR, p.username)
	return &pr, nil
}

func (p *Provider) GetDiff(ctx context.Context, identifier domain.PRIdentifier) (*domain.Diff, error) {
	parts := strings.Split(identifier.Repository, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s", identifier.Repository)
	}

	owner, repo := parts[0], parts[1]
	diffText, err := p.client.GetDiff(ctx, owner, repo, identifier.Number)
	if err != nil {
		return nil, err
	}

	return common.ParseUnifiedDiff(diffText), nil
}

func (p *Provider) GetComments(ctx context.Context, identifier domain.PRIdentifier) ([]domain.Comment, error) {
	parts := strings.Split(identifier.Repository, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s", identifier.Repository)
	}

	owner, repo := parts[0], parts[1]
	ghComments, err := p.client.ListComments(ctx, owner, repo, identifier.Number)
	if err != nil {
		return nil, err
	}

	comments := make([]domain.Comment, 0, len(ghComments))
	for _, ghComment := range ghComments {
		comment := convertComment(ghComment)
		comments = append(comments, comment)
	}

	return comments, nil
}

func (p *Provider) AddComment(ctx context.Context, identifier domain.PRIdentifier, body string, filePath string, line int) error {
	parts := strings.Split(identifier.Repository, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format: %s", identifier.Repository)
	}

	owner, repo := parts[0], parts[1]

	comment := &github.PullRequestComment{
		Body: github.String(body),
	}

	if filePath != "" && line > 0 {
		comment.Path = github.String(filePath)
		comment.Line = github.Int(line)
	}

	return p.client.CreateComment(ctx, owner, repo, identifier.Number, comment)
}

func (p *Provider) SubmitReview(ctx context.Context, review domain.Review) error {
	owner, repo, prNumber, err := common.ParseGitHubIdentifier(review.PRIdentifier)
	if err != nil {
		return fmt.Errorf("failed to parse PR identifier: %w", err)
	}

	event := convertReviewAction(review.Action)
	ghReview := &github.PullRequestReviewRequest{
		Event: github.String(event),
		Body:  github.String(review.Body),
	}

	if len(review.Comments) > 0 {
		comments := make([]*github.DraftReviewComment, 0, len(review.Comments))
		for _, c := range review.Comments {
			comments = append(comments, &github.DraftReviewComment{
				Path: github.String(c.FilePath),
				Line: github.Int(c.Line),
				Body: github.String(c.Body),
			})
		}
		ghReview.Comments = comments
	}

	return p.client.CreateReview(ctx, owner, repo, prNumber, ghReview)
}

func (p *Provider) ValidateCredentials(ctx context.Context) error {
	_, err := p.client.GetUsername(ctx)
	return err
}

func (p *Provider) convertPullRequest(ghPR *github.PullRequest, currentUser string) domain.PullRequest {
	category := domain.PRCategoryOther
	if ghPR.User != nil && ghPR.User.Login != nil && *ghPR.User.Login == currentUser {
		category = domain.PRCategoryAuthored
	} else if ghPR.Assignee != nil && ghPR.Assignee.Login != nil && *ghPR.Assignee.Login == currentUser {
		category = domain.PRCategoryAssigned
	}

	status := domain.PRStatusOpen
	if ghPR.State != nil {
		if *ghPR.State == "closed" && ghPR.MergedAt != nil {
			status = domain.PRStatusMerged
		} else if *ghPR.State == "closed" {
			status = domain.PRStatusClosed
		}
	}

	pr := domain.PullRequest{
		ID:          fmt.Sprintf("%d", ghPR.GetID()),
		Number:      ghPR.GetNumber(),
		Title:       ghPR.GetTitle(),
		Description: ghPR.GetBody(),
		Status:      status,
		Category:    category,
		CreatedAt:   ghPR.GetCreatedAt().Time,
		UpdatedAt:   ghPR.GetUpdatedAt().Time,
		URL:         ghPR.GetHTMLURL(),
		IsDraft:     ghPR.GetDraft(),
		Mergeable:   ghPR.GetMergeable(),
	}

	if ghPR.User != nil {
		pr.Author = domain.User{
			ID:       fmt.Sprintf("%d", ghPR.User.GetID()),
			Username: ghPR.User.GetLogin(),
			Avatar:   ghPR.User.GetAvatarURL(),
		}
	}

	if ghPR.Base != nil && ghPR.Base.Repo != nil {
		pr.Repository = domain.Repo{
			ID:       fmt.Sprintf("%d", ghPR.Base.Repo.GetID()),
			Name:     ghPR.Base.Repo.GetName(),
			FullName: ghPR.Base.Repo.GetFullName(),
			Owner:    ghPR.Base.Repo.GetOwner().GetLogin(),
			URL:      ghPR.Base.Repo.GetHTMLURL(),
		}
		pr.TargetBranch = ghPR.Base.GetRef()
	}

	if ghPR.Head != nil {
		pr.SourceBranch = ghPR.Head.GetRef()
	}

	return pr
}

func convertComment(ghComment *github.PullRequestComment) domain.Comment {
	comment := domain.Comment{
		ID:        fmt.Sprintf("%d", ghComment.GetID()),
		Body:      ghComment.GetBody(),
		CreatedAt: ghComment.GetCreatedAt().Time,
		UpdatedAt: ghComment.GetUpdatedAt().Time,
		FilePath:  ghComment.GetPath(),
		Line:      ghComment.GetLine(),
		Side:      ghComment.GetSide(),
	}

	if ghComment.User != nil {
		comment.Author = domain.User{
			ID:       fmt.Sprintf("%d", ghComment.User.GetID()),
			Username: ghComment.User.GetLogin(),
			Avatar:   ghComment.User.GetAvatarURL(),
		}
	}

	return comment
}

func convertReviewAction(action domain.ReviewAction) string {
	switch action {
	case domain.ReviewActionApprove:
		return "APPROVE"
	case domain.ReviewActionRequestChanges:
		return "REQUEST_CHANGES"
	default:
		return "COMMENT"
	}
}
