package azuredevops

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
)


type ResolvedRepository struct {
	ProjectID string
	RepoID    string
	CachedAt  time.Time
}

type Provider struct {
	client     *Client
	repoCache  map[string]*ResolvedRepository
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
}

func NewProvider(token string, organization string, username string) (*Provider, error) {
	client, err := NewClient(token, organization, username)
	if err != nil {
		return nil, err
	}
	return &Provider{
		client:    client,
		repoCache: make(map[string]*ResolvedRepository),
		cacheTTL:  5 * time.Minute,
	}, nil
}

func (p *Provider) GetType() domain.ProviderType {
	return domain.ProviderAzureDevOps
}

func (p *Provider) ListPullRequests(ctx context.Context, username string) ([]domain.PullRequest, error) {
	projects, err := p.client.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	if projects == nil || len(*projects) == 0 {
		return []domain.PullRequest{}, nil
	}

	var allPRs []domain.PullRequest
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(*projects))

	for _, project := range *projects {
		wg.Add(1)
		go func(projectID, projectName string) {
			defer wg.Done()

			repos, err := p.client.ListRepositories(ctx, projectID)
			if err != nil {
				errChan <- err
				return
			}

			if repos == nil || len(*repos) == 0 {
				return
			}

			for _, repo := range *repos {
				if repo.Id == nil {
					continue
				}

				repoID := repo.Id.String()
				repoName := getString(repo.Name)
				prs, err := p.client.ListPullRequests(ctx, projectID, repoID)
				if err != nil {
					errChan <- err
					continue
				}

				if prs == nil {
					continue
				}

				for _, pr := range *prs {
					domainPR := convertPullRequest(&pr, username)
					if domainPR.URL == "" {
						domainPR.URL = p.buildPRURL(projectName, repoName, domainPR.Number)
					}
					mu.Lock()
					allPRs = append(allPRs, domainPR)
					mu.Unlock()
				}
			}
		}(getUUIDString(project.Id), getString(project.Name))
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return allPRs, <-errChan
	}

	return allPRs, nil
}

func (p *Provider) GetPullRequest(ctx context.Context, identifier domain.PRIdentifier) (*domain.PullRequest, error) {
	projectName, repoName, err := parseRepositoryIdentifier(identifier.Repository)
	if err != nil {
		return nil, err
	}

	projects, err := p.client.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	var projectID string
	for _, project := range *projects {
		if getString(project.Name) == projectName {
			projectID = getUUIDString(project.Id)
			break
		}
	}

	if projectID == "" {
		return nil, fmt.Errorf("project not found: %s", projectName)
	}

	repos, err := p.client.ListRepositories(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var repoID string
	for _, repo := range *repos {
		if getString(repo.Name) == repoName {
			repoID = repo.Id.String()
			break
		}
	}

	if repoID == "" {
		return nil, fmt.Errorf("repository not found: %s", repoName)
	}

	pr, err := p.client.GetPullRequest(ctx, projectID, repoID, identifier.Number)
	if err != nil {
		return nil, err
	}

	domainPR := convertPullRequest(pr, p.client.username)
	if domainPR.URL == "" {
		domainPR.URL = p.buildPRURL(projectName, repoName, domainPR.Number)
	}
	return &domainPR, nil
}

func (p *Provider) GetDiff(ctx context.Context, identifier domain.PRIdentifier) (*domain.Diff, error) {
	projectName, repoName, err := parseRepositoryIdentifier(identifier.Repository)
	if err != nil {
		return nil, err
	}

	projects, err := p.client.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	var projectID string
	for _, project := range *projects {
		if getString(project.Name) == projectName {
			projectID = getUUIDString(project.Id)
			break
		}
	}

	if projectID == "" {
		return nil, fmt.Errorf("project not found: %s", projectName)
	}

	repos, err := p.client.ListRepositories(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var repoID string
	for _, repo := range *repos {
		if getString(repo.Name) == repoName {
			repoID = repo.Id.String()
			break
		}
	}

	if repoID == "" {
		return nil, fmt.Errorf("repository not found: %s", repoName)
	}

	pr, err := p.client.GetPullRequest(ctx, projectID, repoID, identifier.Number)
	if err != nil {
		return nil, err
	}

	if pr.LastMergeSourceCommit == nil || pr.LastMergeTargetCommit == nil {
		return &domain.Diff{Files: []domain.FileDiff{}}, nil
	}

	logger.Log("AzureDevOps: Requesting PR iteration changes for PR #%d", identifier.Number)
	diffText, err := p.client.GetPullRequestIterationChanges(ctx, projectID, repoID, identifier.Number)
	if err != nil {
		logger.LogError("AZURE_GET_DIFF", fmt.Sprintf("project=%s repo=%s PR=%d", projectID, repoID, identifier.Number), err)
		return nil, err
	}

	logger.Log("AzureDevOps: Received diff text length: %d bytes", len(diffText))
	if len(diffText) > 0 {
		logger.Log("AzureDevOps: First 200 chars of diff: %s", diffText[:min(200, len(diffText))])
	}

	diff := common.ParseUnifiedDiff(diffText)
	logger.Log("AzureDevOps: Parsed diff with %d files", len(diff.Files))
	for i, file := range diff.Files {
		logger.Log("AzureDevOps: File %d: %s -> %s (%d hunks)", i+1, file.OldPath, file.NewPath, len(file.Hunks))
	}
	return diff, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *Provider) GetComments(ctx context.Context, identifier domain.PRIdentifier) ([]domain.Comment, error) {
	projectName, repoName, err := parseRepositoryIdentifier(identifier.Repository)
	if err != nil {
		return nil, err
	}

	projects, err := p.client.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	var projectID string
	for _, project := range *projects {
		if getString(project.Name) == projectName {
			projectID = getUUIDString(project.Id)
			break
		}
	}

	if projectID == "" {
		return nil, fmt.Errorf("project not found: %s", projectName)
	}

	repos, err := p.client.ListRepositories(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var repoID string
	for _, repo := range *repos {
		if getString(repo.Name) == repoName {
			repoID = repo.Id.String()
			break
		}
	}

	if repoID == "" {
		return nil, fmt.Errorf("repository not found: %s", repoName)
	}

	threads, err := p.client.GetPullRequestThreads(ctx, projectID, repoID, identifier.Number)
	if err != nil {
		return nil, err
	}

	if threads == nil {
		return []domain.Comment{}, nil
	}

	var comments []domain.Comment
	for _, thread := range *threads {
		if thread.Comments == nil {
			continue
		}

		for _, comment := range *thread.Comments {
			domainComment := domain.Comment{
				ID:        fmt.Sprintf("%d", *comment.Id),
				Body:      getString(comment.Content),
				CreatedAt: comment.PublishedDate.Time,
				UpdatedAt: comment.LastUpdatedDate.Time,
			}

			if comment.Author != nil {
				domainComment.Author = convertIdentity(comment.Author)
			}

			if thread.ThreadContext != nil && thread.ThreadContext.FilePath != nil {
				domainComment.FilePath = getString(thread.ThreadContext.FilePath)
				if thread.ThreadContext.RightFileStart != nil && thread.ThreadContext.RightFileStart.Line != nil {
					domainComment.Line = *thread.ThreadContext.RightFileStart.Line
				}
			}

			comments = append(comments, domainComment)
		}
	}

	return comments, nil
}

func (p *Provider) AddComment(ctx context.Context, identifier domain.PRIdentifier, body string, filePath string, line int) error {
	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, identifier.Repository)
	if err != nil {
		return err
	}

	return p.client.CreateCommentThread(ctx, projectID, repoID, identifier.Number, body, filePath, line)
}

func (p *Provider) SubmitReview(ctx context.Context, review domain.Review) error {
	logger.Log("AzureDevOps: Submitting review for %s (Action: %s)", review.PRIdentifier, review.Action)
	project, repo, prNumber, err := common.ParseAzureDevOpsIdentifier(review.PRIdentifier)
	if err != nil {
		logger.LogError("AZDO_SUBMIT_REVIEW", review.PRIdentifier, err)
		return fmt.Errorf("failed to parse PR identifier: %w", err)
	}

	repository := fmt.Sprintf("%s/%s", project, repo)

	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, repository)
	if err != nil {
		return err
	}

	var createdComments int

	if len(review.Comments) > 0 {
		logger.Log("AzureDevOps: Creating %d inline comment threads", len(review.Comments))
	}
	for i, comment := range review.Comments {
		logger.Log("AzureDevOps: Creating comment %d/%d on %s:%d", i+1, len(review.Comments), comment.FilePath, comment.Line)
		if err := p.client.CreateCommentThread(ctx, projectID, repoID, prNumber, comment.Body, comment.FilePath, comment.Line); err != nil {
			logger.LogError("AZDO_CREATE_COMMENT", fmt.Sprintf("%s#%d", repository, prNumber), err)
			if createdComments > 0 {
				return fmt.Errorf("%w: failed to create comment %d/%d (created %d comments before failure): %v",
					common.ErrPartialReviewSubmission, i+1, len(review.Comments), createdComments, err)
			}
			return fmt.Errorf("failed to create comment %d/%d: %w", i+1, len(review.Comments), err)
		}
		createdComments++
	}

	if review.Action != domain.ReviewActionComment {
		vote := convertReviewActionToVote(review.Action)
		logger.Log("AzureDevOps: Submitting vote %d for PR #%d", vote, prNumber)
		userID, err := p.client.GetAuthenticatedUserID(ctx)
		if err != nil {
			logger.LogError("AZDO_GET_USER_ID", fmt.Sprintf("%s#%d", repository, prNumber), err)
			return fmt.Errorf("failed to get authenticated user ID: %w", err)
		}

		logger.Log("AzureDevOps: Using user ID %s for vote", userID)
		if err := p.client.CreatePullRequestReview(ctx, projectID, repoID, prNumber, userID, vote); err != nil {
			logger.LogError("AZDO_SUBMIT_VOTE", fmt.Sprintf("%s#%d", repository, prNumber), err)
			if createdComments > 0 {
				return fmt.Errorf("%w: failed to submit vote (created %d comments): %v",
					common.ErrPartialReviewSubmission, createdComments, err)
			}
			return fmt.Errorf("failed to submit vote: %w", err)
		}
	}

	if review.Body != "" && review.Action == domain.ReviewActionComment {
		logger.Log("AzureDevOps: Creating review body comment")
		if err := p.client.CreateCommentThread(ctx, projectID, repoID, prNumber, review.Body, "", 0); err != nil {
			logger.LogError("AZDO_CREATE_REVIEW_BODY", fmt.Sprintf("%s#%d", repository, prNumber), err)
			if createdComments > 0 || review.Action != domain.ReviewActionComment {
				return fmt.Errorf("%w: failed to create review body comment (created %d inline comments): %v",
					common.ErrPartialReviewSubmission, createdComments, err)
			}
			return fmt.Errorf("failed to create review comment: %w", err)
		}
	}

	logger.Log("AzureDevOps: Review submitted successfully for %s#%d", repository, prNumber)
	return nil
}

func (p *Provider) ValidateCredentials(ctx context.Context) error {
	return p.client.ValidateCredentials(ctx)
}

func (p *Provider) buildPRURL(projectName, repoName string, prNumber int) string {
	return fmt.Sprintf("https://dev.azure.com/%s/%s/_git/%s/pullrequest/%d",
		p.client.organization, projectName, repoName, prNumber)
}

func convertPullRequest(adoPR *git.GitPullRequest, currentUser string) domain.PullRequest {
	pr := domain.PullRequest{
		ID:           fmt.Sprintf("%d", *adoPR.PullRequestId),
		Number:       *adoPR.PullRequestId,
		Title:        getString(adoPR.Title),
		Description:  getString(adoPR.Description),
		Status:       mapPRStatus(adoPR.Status, adoPR.MergeStatus),
		Category:     determinePRCategory(adoPR, currentUser),
		CreatedAt:    adoPR.CreationDate.Time,
		UpdatedAt:    getUpdateTime(adoPR),
		URL:          buildPRWebURL(adoPR),
		IsDraft:      getBool(adoPR.IsDraft),
		Mergeable:    isMergeable(adoPR.MergeStatus),
		SourceBranch: extractBranchName(adoPR.SourceRefName),
		TargetBranch: extractBranchName(adoPR.TargetRefName),
	}

	if adoPR.CreatedBy != nil {
		pr.Author = convertIdentity(adoPR.CreatedBy)
	}

	if adoPR.Repository != nil {
		pr.Repository = convertRepository(adoPR.Repository)
	}

	return pr
}

func convertIdentity(identity *webapi.IdentityRef) domain.User {
	return domain.User{
		ID:       getString(identity.Id),
		Username: getString(identity.DisplayName),
		Email:    getString(identity.UniqueName),
		Avatar:   getString(identity.ImageUrl),
	}
}

func convertRepository(repo *git.GitRepository) domain.Repo {
	projectName := ""
	if repo.Project != nil {
		projectName = getString(repo.Project.Name)
	}

	repoName := getString(repo.Name)

	return domain.Repo{
		ID:       repo.Id.String(),
		Name:     repoName,
		FullName: buildRepositoryIdentifier(projectName, repoName),
		Owner:    projectName,
		URL:      getString(repo.WebUrl),
	}
}

func mapPRStatus(status *git.PullRequestStatus, mergeStatus *git.PullRequestAsyncStatus) domain.PRStatus {
	if status == nil {
		return domain.PRStatusOpen
	}

	switch *status {
	case git.PullRequestStatusValues.Active:
		return domain.PRStatusOpen
	case git.PullRequestStatusValues.Completed:
		if mergeStatus != nil && *mergeStatus == git.PullRequestAsyncStatusValues.Succeeded {
			return domain.PRStatusMerged
		}
		return domain.PRStatusClosed
	case git.PullRequestStatusValues.Abandoned:
		return domain.PRStatusClosed
	default:
		return domain.PRStatusOpen
	}
}

func determinePRCategory(pr *git.GitPullRequest, currentUser string) domain.PRCategory {
	if pr.CreatedBy != nil && matchesUser(pr.CreatedBy, currentUser) {
		return domain.PRCategoryAuthored
	}

	if pr.Reviewers != nil {
		for _, reviewer := range *pr.Reviewers {
			identity := webapi.IdentityRef{
				Id:          reviewer.Id,
				DisplayName: reviewer.DisplayName,
				UniqueName:  reviewer.UniqueName,
				ImageUrl:    reviewer.ImageUrl,
			}
			if matchesUser(&identity, currentUser) {
				return domain.PRCategoryAssigned
			}
		}
	}

	return domain.PRCategoryOther
}

func matchesUser(identity *webapi.IdentityRef, username string) bool {
	if identity == nil {
		return false
	}

	displayName := getString(identity.DisplayName)
	uniqueName := getString(identity.UniqueName)

	return displayName == username ||
		strings.EqualFold(displayName, username) ||
		strings.HasPrefix(strings.ToLower(uniqueName), strings.ToLower(username)+"@")
}

func getUpdateTime(pr *git.GitPullRequest) time.Time {
	if pr.ClosedDate != nil && !pr.ClosedDate.Time.IsZero() {
		return pr.ClosedDate.Time
	}
	if pr.CreationDate != nil {
		return pr.CreationDate.Time
	}
	return time.Now()
}

func (p *Provider) resolveProjectAndRepoWithCache(ctx context.Context, repository string) (projectID, repoID string, err error) {
	p.cacheMutex.RLock()
	if cached, ok := p.repoCache[repository]; ok {
		if time.Since(cached.CachedAt) < p.cacheTTL {
			p.cacheMutex.RUnlock()
			logger.Log("AzureDevOps: Cache HIT for repository %s", repository)
			return cached.ProjectID, cached.RepoID, nil
		}
		logger.Log("AzureDevOps: Cache EXPIRED for repository %s", repository)
	}
	p.cacheMutex.RUnlock()

	logger.Log("AzureDevOps: Cache MISS for repository %s - resolving", repository)
	projectID, repoID, err = p.resolveProjectAndRepo(ctx, repository)
	if err != nil {
		logger.LogError("AZDO_RESOLVE_REPO", repository, err)
		p.cacheMutex.Lock()
		delete(p.repoCache, repository)
		p.cacheMutex.Unlock()
		return "", "", err
	}

	p.cacheMutex.Lock()
	p.repoCache[repository] = &ResolvedRepository{
		ProjectID: projectID,
		RepoID:    repoID,
		CachedAt:  time.Now(),
	}
	p.cacheMutex.Unlock()

	logger.Log("AzureDevOps: Cached repository %s (Project: %s, Repo: %s)", repository, projectID, repoID)
	return projectID, repoID, nil
}

func (p *Provider) resolveProjectAndRepo(ctx context.Context, repository string) (projectID, repoID string, err error) {
	projectName, repoName, err := parseRepositoryIdentifier(repository)
	if err != nil {
		return "", "", err
	}

	projects, err := p.client.ListProjects(ctx)
	if err != nil {
		return "", "", err
	}

	for _, project := range *projects {
		if getString(project.Name) == projectName {
			projectID = getUUIDString(project.Id)
			break
		}
	}

	if projectID == "" {
		return "", "", fmt.Errorf("project not found: %s", projectName)
	}

	repos, err := p.client.ListRepositories(ctx, projectID)
	if err != nil {
		return "", "", err
	}

	for _, repo := range *repos {
		if getString(repo.Name) == repoName {
			repoID = repo.Id.String()
			break
		}
	}

	if repoID == "" {
		return "", "", fmt.Errorf("repository not found: %s", repoName)
	}

	return projectID, repoID, nil
}

func convertReviewActionToVote(action domain.ReviewAction) int {
	switch action {
	case domain.ReviewActionApprove:
		return 10
	case domain.ReviewActionRequestChanges:
		return -10
	default:
		return 0
	}
}
