package azuredevops

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
)

var (
	ErrDiffNotImplemented         = errors.New("GetDiff not yet implemented for Azure DevOps")
	ErrCommentsNotImplemented     = errors.New("GetComments not yet implemented for Azure DevOps")
	ErrAddCommentNotImplemented   = errors.New("AddComment not yet implemented for Azure DevOps")
	ErrSubmitReviewNotImplemented = errors.New("SubmitReview not yet implemented for Azure DevOps")
)

type Provider struct {
	client *Client
}

func NewProvider(token string, organization string, username string) (*Provider, error) {
	client, err := NewClient(token, organization, username)
	if err != nil {
		return nil, err
	}
	return &Provider{
		client: client,
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
	return &domainPR, nil
}

func (p *Provider) GetDiff(ctx context.Context, identifier domain.PRIdentifier) (*domain.Diff, error) {
	return nil, ErrDiffNotImplemented
}

func (p *Provider) GetComments(ctx context.Context, identifier domain.PRIdentifier) ([]domain.Comment, error) {
	return nil, ErrCommentsNotImplemented
}

func (p *Provider) AddComment(ctx context.Context, identifier domain.PRIdentifier, body string, filePath string, line int) error {
	return ErrAddCommentNotImplemented
}

func (p *Provider) SubmitReview(ctx context.Context, review domain.Review) error {
	return ErrSubmitReviewNotImplemented
}

func (p *Provider) ValidateCredentials(ctx context.Context) error {
	return p.client.ValidateCredentials(ctx)
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
