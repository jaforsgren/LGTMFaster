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
	// commitMessages maps "repoID@commitID" to the commit's summary line.
	commitMessages sync.Map
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
	allPRs := []domain.PullRequest{}
	var mu sync.Mutex

	err := p.visitRepositories(ctx, func(projectID, projectName, repoID, repoName string) {
		prs, err := p.client.ListPullRequests(ctx, projectID, repoID)
		if err != nil {
			logger.Log("AzureDevOps: Skipping inaccessible repo %s/%s: %v", projectName, repoName, err)
			return
		}

		if prs == nil {
			return
		}

		for _, pr := range *prs {
			domainPR := p.toDomainPullRequest(&pr, projectName, repoName, username)
			mu.Lock()
			allPRs = append(allPRs, domainPR)
			mu.Unlock()
		}
	})

	return allPRs, err
}

type repositoryVisitor func(projectID, projectName, repoID, repoName string)

// visitRepositories walks every repository in every project, one goroutine per project.
// The visitor is therefore called concurrently and must synchronise any shared state itself.
func (p *Provider) visitRepositories(ctx context.Context, visit repositoryVisitor) error {
	projects, err := p.client.ListProjects(ctx)
	if err != nil {
		return err
	}

	if projects == nil || len(*projects) == 0 {
		return nil
	}

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

			if repos == nil {
				return
			}

			for _, repo := range *repos {
				if repo.Id == nil {
					continue
				}
				visit(projectID, projectName, repo.Id.String(), common.GetString(repo.Name))
			}
		}(common.GetUUIDString(project.Id), common.GetString(project.Name))
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return <-errChan
	}

	return nil
}

func (p *Provider) GetPullRequest(ctx context.Context, identifier domain.PRIdentifier) (*domain.PullRequest, error) {
	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, identifier.Repository)
	if err != nil {
		return nil, err
	}

	projectName, repoName, err := parseRepositoryIdentifier(identifier.Repository)
	if err != nil {
		return nil, err
	}

	pr, err := p.client.GetPullRequest(ctx, projectID, repoID, identifier.Number)
	if err != nil {
		return nil, err
	}

	domainPR := p.toDomainPullRequest(pr, projectName, repoName, p.client.username)
	return &domainPR, nil
}

func (p *Provider) GetDiff(ctx context.Context, identifier domain.PRIdentifier) (*domain.Diff, error) {
	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, identifier.Repository)
	if err != nil {
		return nil, err
	}

	pr, err := p.client.GetPullRequest(ctx, projectID, repoID, identifier.Number)
	if err != nil {
		return nil, err
	}

	if pr.LastMergeSourceCommit == nil || pr.LastMergeTargetCommit == nil {
		logger.Log("AzureDevOps: PR #%d has no merge commits (LastMergeSourceCommit or LastMergeTargetCommit is nil), attempting to fetch diff anyway", identifier.Number)
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

func (p *Provider) GetComments(ctx context.Context, identifier domain.PRIdentifier) ([]domain.Comment, error) {
	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, identifier.Repository)
	if err != nil {
		return nil, err
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
		threadID := ""
		if thread.Id != nil {
			threadID = fmt.Sprintf("%d", *thread.Id)
		}
		threadState := threadStateOf(thread.Status)

		for _, comment := range *thread.Comments {
			isSystem := comment.CommentType != nil && *comment.CommentType == git.CommentTypeValues.System
			body := common.GetString(comment.Content)
			if isSystem {
				body = describeVote(body)
			}
			domainComment := domain.Comment{
				ID:          fmt.Sprintf("%d", *comment.Id),
				Body:        body,
				CreatedAt:   comment.PublishedDate.Time,
				UpdatedAt:   comment.LastUpdatedDate.Time,
				IsSystem:    isSystem,
				ThreadID:    threadID,
				ThreadState: threadState,
			}

			if comment.Author != nil {
				domainComment.Author = convertIdentity(comment.Author)
			}

			domainComment.FilePath, domainComment.Line, domainComment.Side = commentPosition(thread.ThreadContext)

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

	return p.client.CreateCommentThread(ctx, projectID, repoID, identifier.Number, body, filePath, line, domain.CommentSideRight)
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
		if err := p.client.CreateCommentThread(ctx, projectID, repoID, prNumber, comment.Body, comment.FilePath, comment.Line, comment.Side); err != nil {
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

	if review.Body != "" {
		logger.Log("AzureDevOps: Creating review body comment")
		if err := p.client.CreateCommentThread(ctx, projectID, repoID, prNumber, review.Body, "", 0, ""); err != nil {
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

func (p *Provider) MergePullRequest(ctx context.Context, identifier domain.PRIdentifier, mergeMethod string, deleteBranch bool) error {
	logger.Log("AzureDevOps: Completing PR #%d from %s (method: %s, deleteBranch: %v)",
		identifier.Number, identifier.Repository, mergeMethod, deleteBranch)

	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, identifier.Repository)
	if err != nil {
		logger.LogError("AZDO_MERGE_PR", identifier.Repository, err)
		return err
	}

	if err := p.client.CompletePullRequest(ctx, projectID, repoID, identifier.Number, mergeMethod, deleteBranch); err != nil {
		logger.LogError("AZDO_MERGE_PR", fmt.Sprintf("%s#%d", identifier.Repository, identifier.Number), err)
		return fmt.Errorf("failed to complete PR: %w", err)
	}

	logger.Log("AzureDevOps: Successfully completed PR #%d", identifier.Number)
	return nil
}

func (p *Provider) UpdatePullRequestDescription(ctx context.Context, identifier domain.PRIdentifier, description string) error {
	logger.Log("AzureDevOps: Updating PR #%d description from %s", identifier.Number, identifier.Repository)

	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, identifier.Repository)
	if err != nil {
		logger.LogError("AZDO_UPDATE_PR_DESC", identifier.Repository, err)
		return err
	}

	if err := p.client.UpdatePullRequestDescription(ctx, projectID, repoID, identifier.Number, description); err != nil {
		logger.LogError("AZDO_UPDATE_PR_DESC", fmt.Sprintf("%s#%d", identifier.Repository, identifier.Number), err)
		return fmt.Errorf("failed to update PR description: %w", err)
	}

	logger.Log("AzureDevOps: Successfully updated PR #%d description", identifier.Number)
	return nil
}

// toDomainPullRequest converts a pull request, falling back to a built web URL when the API
// response has no link to one.
func (p *Provider) toDomainPullRequest(pr *git.GitPullRequest, projectName, repoName, currentUser string) domain.PullRequest {
	domainPR := convertPullRequest(pr, currentUser)
	if domainPR.URL == "" {
		domainPR.URL = p.buildPRURL(projectName, repoName, domainPR.Number)
	}
	return domainPR
}

func (p *Provider) buildPRURL(projectName, repoName string, prNumber int) string {
	return fmt.Sprintf("https://dev.azure.com/%s/%s/_git/%s/pullrequest/%d",
		p.client.organization, projectName, repoName, prNumber)
}

func convertPullRequest(adoPR *git.GitPullRequest, currentUser string) domain.PullRequest {
	pr := domain.PullRequest{
		ID:             fmt.Sprintf("%d", *adoPR.PullRequestId),
		Number:         *adoPR.PullRequestId,
		Title:          common.GetString(adoPR.Title),
		Description:    common.GetString(adoPR.Description),
		Status:         mapPRStatus(adoPR.Status, adoPR.MergeStatus),
		Category:       determinePRCategory(adoPR, currentUser),
		ApprovalStatus: calculateApprovalStatus(adoPR),
		ReviewedByMe:   votedBy(adoPR, currentUser),
		CreatedAt:      adoPR.CreationDate.Time,
		UpdatedAt:      getUpdateTime(adoPR),
		URL:            buildPRWebURL(adoPR),
		IsDraft:        common.GetBool(adoPR.IsDraft),
		Mergeable:      isMergeable(adoPR.MergeStatus),
		SourceBranch:   extractBranchName(adoPR.SourceRefName),
		TargetBranch:   extractBranchName(adoPR.TargetRefName),
	}

	if adoPR.CreatedBy != nil {
		pr.Author = convertIdentity(adoPR.CreatedBy)
	}

	if adoPR.Repository != nil {
		pr.Repository = convertRepository(adoPR.Repository)
	}

	return pr
}

func calculateApprovalStatus(adoPR *git.GitPullRequest) domain.ApprovalStatus {
	if adoPR.Reviewers == nil || len(*adoPR.Reviewers) == 0 {
		return domain.ApprovalStatusNone
	}

	hasApproval := false
	hasRejection := false

	for _, reviewer := range *adoPR.Reviewers {
		if reviewer.Vote == nil {
			continue
		}
		vote := *reviewer.Vote
		if vote >= 10 {
			hasApproval = true
		} else if vote <= -10 {
			hasRejection = true
		}
	}

	if hasRejection {
		return domain.ApprovalStatusChangesRequested
	}
	if hasApproval {
		return domain.ApprovalStatusApproved
	}
	return domain.ApprovalStatusPending
}

func convertIdentity(identity *webapi.IdentityRef) domain.User {
	return domain.User{
		ID:       common.GetString(identity.Id),
		Username: common.GetString(identity.DisplayName),
		Email:    common.GetString(identity.UniqueName),
		Avatar:   common.GetString(identity.ImageUrl),
	}
}

func convertRepository(repo *git.GitRepository) domain.Repo {
	projectName := ""
	if repo.Project != nil {
		projectName = common.GetString(repo.Project.Name)
	}

	repoName := common.GetString(repo.Name)

	return domain.Repo{
		ID:       repo.Id.String(),
		Name:     repoName,
		FullName: buildRepositoryIdentifier(projectName, repoName),
		Owner:    projectName,
		URL:      common.GetString(repo.WebUrl),
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

// votedBy reports whether the user is a reviewer who has voted.
func votedBy(pr *git.GitPullRequest, currentUser string) bool {
	if pr.Reviewers == nil {
		return false
	}
	for _, reviewer := range *pr.Reviewers {
		identity := webapi.IdentityRef{Id: reviewer.Id, DisplayName: reviewer.DisplayName, UniqueName: reviewer.UniqueName}
		if matchesUser(&identity, currentUser) && reviewer.Vote != nil && *reviewer.Vote != 0 {
			return true
		}
	}
	return false
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

	displayName := common.GetString(identity.DisplayName)
	uniqueName := common.GetString(identity.UniqueName)

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
		if common.GetString(project.Name) == projectName {
			projectID = common.GetUUIDString(project.Id)
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
		if common.GetString(repo.Name) == repoName {
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
