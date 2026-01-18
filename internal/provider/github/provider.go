package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v57/github"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
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
	logger.Log("GitHub: Listing pull requests for user %s", username)
	ghPRs, err := p.client.ListPullRequests(ctx)
	if err != nil {
		logger.LogError("GITHUB_LIST_PRS", username, err)
		return nil, err
	}

	prs := make([]domain.PullRequest, 0, len(ghPRs))
	for _, ghPR := range ghPRs {
		pr := p.convertPullRequest(ghPR, username)

		if ghPR.Base != nil && ghPR.Base.Repo != nil {
			owner := ghPR.Base.Repo.GetOwner().GetLogin()
			repo := ghPR.Base.Repo.GetName()
			reviews, err := p.client.ListReviews(ctx, owner, repo, ghPR.GetNumber())
			if err == nil {
				pr.ApprovalStatus = p.calculateApprovalStatus(reviews)
			}
		}

		prs = append(prs, pr)
	}

	logger.Log("GitHub: Found %d pull requests", len(prs))
	return prs, nil
}

func (p *Provider) GetPullRequest(ctx context.Context, identifier domain.PRIdentifier) (*domain.PullRequest, error) {
	logger.Log("GitHub: Getting PR #%d from %s", identifier.Number, identifier.Repository)
	owner, repo, err := common.ParseGitHubRepository(identifier.Repository)
	if err != nil {
		logger.LogError("GITHUB_GET_PR", identifier.Repository, err)
		return nil, err
	}

	ghPR, err := p.client.GetPullRequest(ctx, owner, repo, identifier.Number)
	if err != nil {
		logger.LogError("GITHUB_GET_PR", fmt.Sprintf("%s/%s#%d", owner, repo, identifier.Number), err)
		return nil, err
	}

	pr := p.convertPullRequest(ghPR, p.username)

	reviews, err := p.client.ListReviews(ctx, owner, repo, identifier.Number)
	if err == nil {
		pr.ApprovalStatus = p.calculateApprovalStatus(reviews)
	}

	logger.Log("GitHub: Retrieved PR #%d: %s", identifier.Number, *ghPR.Title)
	return &pr, nil
}

func (p *Provider) GetDiff(ctx context.Context, identifier domain.PRIdentifier) (*domain.Diff, error) {
	logger.Log("GitHub: Getting diff for PR #%d from %s", identifier.Number, identifier.Repository)
	owner, repo, err := common.ParseGitHubRepository(identifier.Repository)
	if err != nil {
		logger.LogError("GITHUB_GET_DIFF", identifier.Repository, err)
		return nil, err
	}

	diffText, err := p.client.GetDiff(ctx, owner, repo, identifier.Number)
	if err != nil {
		logger.LogError("GITHUB_GET_DIFF", fmt.Sprintf("%s/%s#%d", owner, repo, identifier.Number), err)
		return nil, err
	}

	logger.Log("GitHub: Received diff text length: %d bytes", len(diffText))
	if len(diffText) > 0 {
		logger.Log("GitHub: First 200 chars of diff: %s", diffText[:min(200, len(diffText))])
	}

	diff := common.ParseUnifiedDiff(diffText)
	logger.Log("GitHub: Parsed diff with %d files", len(diff.Files))
	for i, file := range diff.Files {
		logger.Log("GitHub: File %d: %s -> %s (%d hunks)", i+1, file.OldPath, file.NewPath, len(file.Hunks))
	}
	return diff, nil
}

func (p *Provider) GetComments(ctx context.Context, identifier domain.PRIdentifier) ([]domain.Comment, error) {
	owner, repo, err := common.ParseGitHubRepository(identifier.Repository)
	if err != nil {
		return nil, err
	}

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
	owner, repo, err := common.ParseGitHubRepository(identifier.Repository)
	if err != nil {
		return err
	}

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
	logger.Log("GitHub: Submitting review for %s (Action: %s)", review.PRIdentifier, review.Action)
	owner, repo, prNumber, err := common.ParseGitHubIdentifier(review.PRIdentifier)
	if err != nil {
		logger.LogError("GITHUB_SUBMIT_REVIEW", review.PRIdentifier, err)
		return fmt.Errorf("failed to parse PR identifier: %w", err)
	}

	event := convertReviewAction(review.Action)
	ghReview := &github.PullRequestReviewRequest{
		Event: github.String(event),
		Body:  github.String(review.Body),
	}

	if len(review.Comments) > 0 {
		logger.Log("GitHub: Review includes %d inline comments", len(review.Comments))
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

	if err := p.client.CreateReview(ctx, owner, repo, prNumber, ghReview); err != nil {
		logger.LogError("GITHUB_SUBMIT_REVIEW", fmt.Sprintf("%s/%s#%d", owner, repo, prNumber), err)
		return fmt.Errorf("%s", common.ExtractErrorMessage(err))
	}

	logger.Log("GitHub: Review submitted successfully for %s/%s#%d", owner, repo, prNumber)
	return nil
}

func (p *Provider) ValidateCredentials(ctx context.Context) error {
	_, err := p.client.GetUsername(ctx)
	return err
}

func (p *Provider) MergePullRequest(ctx context.Context, identifier domain.PRIdentifier, mergeMethod string, deleteBranch bool) error {
	logger.Log("GitHub: Merging PR #%d from %s (method: %s, deleteBranch: %v)",
		identifier.Number, identifier.Repository, mergeMethod, deleteBranch)

	owner, repo, err := common.ParseGitHubRepository(identifier.Repository)
	if err != nil {
		logger.LogError("GITHUB_MERGE_PR", identifier.Repository, err)
		return err
	}

	if err := p.client.MergePullRequest(ctx, owner, repo, identifier.Number, mergeMethod, deleteBranch); err != nil {
		logger.LogError("GITHUB_MERGE_PR", fmt.Sprintf("%s/%s#%d", owner, repo, identifier.Number), err)
		return fmt.Errorf("%s", common.ExtractErrorMessage(err))
	}

	logger.Log("GitHub: Successfully merged PR #%d", identifier.Number)
	return nil
}

func (p *Provider) UpdatePullRequestDescription(ctx context.Context, identifier domain.PRIdentifier, description string) error {
	logger.Log("GitHub: Updating PR #%d description from %s", identifier.Number, identifier.Repository)

	owner, repo, err := common.ParseGitHubRepository(identifier.Repository)
	if err != nil {
		logger.LogError("GITHUB_UPDATE_PR_DESC", identifier.Repository, err)
		return err
	}

	update := &github.PullRequest{
		Body: github.String(description),
	}

	_, err = p.client.UpdatePullRequest(ctx, owner, repo, identifier.Number, update)
	if err != nil {
		logger.LogError("GITHUB_UPDATE_PR_DESC", fmt.Sprintf("%s/%s#%d", owner, repo, identifier.Number), err)
		return fmt.Errorf("%s", common.ExtractErrorMessage(err))
	}

	logger.Log("GitHub: Successfully updated PR #%d description", identifier.Number)
	return nil
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

func (p *Provider) calculateApprovalStatus(reviews []*github.PullRequestReview) domain.ApprovalStatus {
	if len(reviews) == 0 {
		return domain.ApprovalStatusNone
	}

	latestByUser := make(map[string]*github.PullRequestReview)
	for _, review := range reviews {
		if review.User == nil || review.User.Login == nil {
			continue
		}
		state := review.GetState()
		if state == "COMMENTED" || state == "DISMISSED" || state == "PENDING" {
			continue
		}
		username := *review.User.Login
		existing, exists := latestByUser[username]
		if !exists || review.GetSubmittedAt().After(existing.GetSubmittedAt().Time) {
			latestByUser[username] = review
		}
	}

	hasApproval := false
	hasChangesRequested := false

	for _, review := range latestByUser {
		state := review.GetState()
		switch state {
		case "APPROVED":
			hasApproval = true
		case "CHANGES_REQUESTED":
			hasChangesRequested = true
		}
	}

	if hasChangesRequested {
		return domain.ApprovalStatusChangesRequested
	}
	if hasApproval {
		return domain.ApprovalStatusApproved
	}
	return domain.ApprovalStatusPending
}
