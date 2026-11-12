package ui

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

type mockRepository struct {
	pats               map[string]*domain.PAT
	pipelineFavorites  []string
	pipelineLastViewed map[string]time.Time
	settings           domain.Settings
	session            domain.Session
}

func (m *mockRepository) ListPATs() ([]domain.PAT, error) {
	pats := make([]domain.PAT, 0, len(m.pats))
	for _, pat := range m.pats {
		pats = append(pats, *pat)
	}
	return pats, nil
}

func (m *mockRepository) GetPAT(id string) (*domain.PAT, error) {
	pat, ok := m.pats[id]
	if !ok {
		return nil, fmt.Errorf("PAT not found")
	}
	return pat, nil
}

func (m *mockRepository) SavePAT(pat domain.PAT) error {
	m.pats[pat.ID] = &pat
	return nil
}

func (m *mockRepository) UpdatePAT(pat domain.PAT) error {
	m.pats[pat.ID] = &pat
	return nil
}

func (m *mockRepository) DeletePAT(id string) error {
	delete(m.pats, id)
	return nil
}

func (m *mockRepository) GetSelectedPATs() ([]domain.PAT, error) {
	return nil, nil
}

func (m *mockRepository) SetSelectedPATs(ids []string, primaryID string) error {
	return nil
}

func (m *mockRepository) TogglePATSelection(id string) error {
	return nil
}

func (m *mockRepository) GetPrimaryPAT() (*domain.PAT, error) {
	return nil, nil
}

func (m *mockRepository) SetPrimaryPAT(id string) error {
	return nil
}

func (m *mockRepository) GetActivePAT() (*domain.PAT, error) {
	for _, pat := range m.pats {
		if pat.IsActive {
			return pat, nil
		}
	}
	return nil, fmt.Errorf("no active PAT")
}

func (m *mockRepository) SetActivePAT(id string) error {
	return nil
}

func (m *mockRepository) ListRepositoryLinks() ([]domain.RepositoryLink, error) {
	return nil, nil
}

func (m *mockRepository) GetRepositoryLink(id string) (*domain.RepositoryLink, error) {
	return nil, nil
}

func (m *mockRepository) GetRepositoryLinkByRepoFullName(repoFullName string) (*domain.RepositoryLink, error) {
	return nil, nil
}

func (m *mockRepository) SaveRepositoryLink(link domain.RepositoryLink) error {
	return nil
}

func (m *mockRepository) DeleteRepositoryLink(id string) error {
	return nil
}

func (m *mockRepository) ListPipelineFavorites() ([]string, error) {
	return m.pipelineFavorites, nil
}

func (m *mockRepository) TogglePipelineFavorite(key string) (bool, error) {
	for i, favorite := range m.pipelineFavorites {
		if favorite == key {
			m.pipelineFavorites = append(m.pipelineFavorites[:i], m.pipelineFavorites[i+1:]...)
			return false, nil
		}
	}
	m.pipelineFavorites = append(m.pipelineFavorites, key)
	return true, nil
}

func (m *mockRepository) ListPipelineLastViewed() (map[string]time.Time, error) {
	return m.pipelineLastViewed, nil
}

func (m *mockRepository) MarkPipelineViewed(key string, viewedAt time.Time) error {
	if m.pipelineLastViewed == nil {
		m.pipelineLastViewed = make(map[string]time.Time)
	}
	m.pipelineLastViewed[key] = viewedAt
	return nil
}

func (m *mockRepository) GetSettings() (domain.Settings, error) {
	settings := m.settings
	if settings.HideOldOtherPRsDays <= 0 {
		settings.HideOldOtherPRsDays = domain.DefaultHideOldOtherPRsDays
	}
	return settings, nil
}

func (m *mockRepository) SaveSettings(settings domain.Settings) error {
	m.settings = settings
	return nil
}

func (m *mockRepository) GetSession() (domain.Session, error) {
	return m.session, nil
}

func (m *mockRepository) SaveSession(session domain.Session) error {
	m.session = session
	return nil
}

func (m *mockRepository) ConfigPath() string {
	return ""
}

func (m *mockRepository) Reload() error {
	return nil
}

type mockProvider struct {
	submitReviewCalled bool
	lastReview         domain.Review
	pushedBranches     []domain.PushedBranch
	branches           domain.RepositoryBranches
	createdPR          *domain.PullRequest
	lastNewPR          *domain.NewPullRequest
	commits            []domain.Commit
	commitDiffs        map[string]*domain.Diff
	checks             map[int]domain.Checks
	linkedWorkItems    []domain.WorkItem
	assignedWorkItems  []domain.WorkItem
	linkedWorkItemID   int
}

func (m *mockProvider) ListPullRequests(ctx context.Context, username string) ([]domain.PullRequest, error) {
	return nil, nil
}

func (m *mockProvider) GetPullRequest(ctx context.Context, identifier domain.PRIdentifier) (*domain.PullRequest, error) {
	return nil, nil
}

func (m *mockProvider) GetDiff(ctx context.Context, identifier domain.PRIdentifier) (*domain.Diff, error) {
	return nil, nil
}

func (m *mockProvider) GetComments(ctx context.Context, identifier domain.PRIdentifier) ([]domain.Comment, error) {
	return nil, nil
}

func (m *mockProvider) AddComment(ctx context.Context, identifier domain.PRIdentifier, body, filePath string, line int) error {
	return nil
}

func (m *mockProvider) SubmitReview(ctx context.Context, review domain.Review) error {
	m.submitReviewCalled = true
	m.lastReview = review
	return nil
}

func (m *mockProvider) ValidateCredentials(ctx context.Context) error {
	return nil
}

func (m *mockProvider) MergePullRequest(ctx context.Context, identifier domain.PRIdentifier, mergeMethod string, deleteBranch bool) error {
	return nil
}

func (m *mockProvider) UpdatePullRequestDescription(ctx context.Context, identifier domain.PRIdentifier, description string) error {
	return nil
}

func (m *mockProvider) ListRecentlyPushedBranches(ctx context.Context, since time.Time) ([]domain.PushedBranch, error) {
	return m.pushedBranches, nil
}

func (m *mockProvider) ListBranches(ctx context.Context, repositoryFullName string) (domain.RepositoryBranches, error) {
	return m.branches, nil
}

func (m *mockProvider) CreatePullRequest(ctx context.Context, pr domain.NewPullRequest) (*domain.PullRequest, error) {
	m.lastNewPR = &pr
	return m.createdPR, nil
}

func (m *mockProvider) ListCommits(ctx context.Context, identifier domain.PRIdentifier) ([]domain.Commit, error) {
	return m.commits, nil
}

func (m *mockProvider) GetCommitDiff(ctx context.Context, identifier domain.PRIdentifier, commitID string) (*domain.Diff, error) {
	return m.commitDiffs[commitID], nil
}

func (m *mockProvider) GetChecks(ctx context.Context, identifier domain.PRIdentifier) (domain.Checks, error) {
	return m.checks[identifier.Number], nil
}

func (m *mockProvider) ListLinkedWorkItems(ctx context.Context, identifier domain.PRIdentifier) ([]domain.WorkItem, error) {
	return m.linkedWorkItems, nil
}

func (m *mockProvider) ListAssignedWorkItems(ctx context.Context, identifier domain.PRIdentifier) ([]domain.WorkItem, error) {
	return m.assignedWorkItems, nil
}

func (m *mockProvider) LinkWorkItem(ctx context.Context, identifier domain.PRIdentifier, workItemID int) error {
	m.linkedWorkItemID = workItemID
	return nil
}

func (m *mockProvider) GetType() domain.ProviderType {
	return domain.ProviderGitHub
}

func TestOwnPRValidation_ApproveConvertsToComment(t *testing.T) {
	repo := &mockRepository{
		pats: map[string]*domain.PAT{
			"pat-1": {
				ID:       "pat-1",
				Username: "testuser",
				Provider: domain.ProviderGitHub,
			},
		},
	}

	provider := &mockProvider{}

	prInspect := views.NewPRInspectView()
	reviewView := views.NewReviewView()

	m := Model{
		ctx:        context.Background(),
		repository: repo,
		prInspect:  prInspect,
		reviewView: reviewView,
		providers: map[string]domain.Provider{
			"pat-1": provider,
		},
	}

	pr := &domain.PullRequest{
		Number: 1,
		Author: domain.User{
			Username: "testuser",
		},
		Repository: domain.Repo{
			FullName: "owner/repo",
		},
		PATID:        "pat-1",
		ProviderType: domain.ProviderGitHub,
	}

	m.prInspect.SetPR(pr)
	m.reviewView.Activate(views.ReviewModeApprove)

	cmd := m.submitReview()
	msg := cmd()

	successMsg, ok := msg.(SuccessMsg)
	if !ok {
		t.Fatalf("Expected SuccessMsg, got %T", msg)
	}

	if !provider.submitReviewCalled {
		t.Error("Expected SubmitReview to be called")
	}

	if provider.lastReview.Action != domain.ReviewActionComment {
		t.Errorf("Expected action to be converted to comment, got %s", provider.lastReview.Action)
	}

	if !successMsg.reloadComments {
		t.Error("Expected reloadComments to be true")
	}

	if successMsg.reloadCommentsPR == nil {
		t.Error("Expected reloadCommentsPR to be set")
	}
}

func TestOwnPRValidation_RequestChangesConvertsToComment(t *testing.T) {
	repo := &mockRepository{
		pats: map[string]*domain.PAT{
			"pat-1": {
				ID:       "pat-1",
				Username: "testuser",
				Provider: domain.ProviderGitHub,
			},
		},
	}

	provider := &mockProvider{}

	prInspect := views.NewPRInspectView()
	reviewView := views.NewReviewView()

	m := Model{
		ctx:        context.Background(),
		repository: repo,
		prInspect:  prInspect,
		reviewView: reviewView,
		providers: map[string]domain.Provider{
			"pat-1": provider,
		},
	}

	pr := &domain.PullRequest{
		Number: 1,
		Author: domain.User{
			Username: "testuser",
		},
		Repository: domain.Repo{
			FullName: "owner/repo",
		},
		PATID:        "pat-1",
		ProviderType: domain.ProviderGitHub,
	}

	m.prInspect.SetPR(pr)
	m.reviewView.Activate(views.ReviewModeRequestChanges)

	cmd := m.submitReview()
	msg := cmd()

	successMsg, ok := msg.(SuccessMsg)
	if !ok {
		t.Fatalf("Expected SuccessMsg, got %T", msg)
	}

	if !provider.submitReviewCalled {
		t.Error("Expected SubmitReview to be called")
	}

	if provider.lastReview.Action != domain.ReviewActionComment {
		t.Errorf("Expected action to be converted to comment, got %s", provider.lastReview.Action)
	}

	if !successMsg.reloadComments {
		t.Error("Expected reloadComments to be true")
	}
}

func TestOwnPRValidation_CommentRemainsComment(t *testing.T) {
	repo := &mockRepository{
		pats: map[string]*domain.PAT{
			"pat-1": {
				ID:       "pat-1",
				Username: "testuser",
				Provider: domain.ProviderGitHub,
			},
		},
	}

	provider := &mockProvider{}

	prInspect := views.NewPRInspectView()
	reviewView := views.NewReviewView()

	m := Model{
		ctx:        context.Background(),
		repository: repo,
		prInspect:  prInspect,
		reviewView: reviewView,
		providers: map[string]domain.Provider{
			"pat-1": provider,
		},
	}

	pr := &domain.PullRequest{
		Number: 1,
		Author: domain.User{
			Username: "testuser",
		},
		Repository: domain.Repo{
			FullName: "owner/repo",
		},
		PATID:        "pat-1",
		ProviderType: domain.ProviderGitHub,
	}

	m.prInspect.SetPR(pr)
	m.reviewView.Activate(views.ReviewModeComment)

	cmd := m.submitReview()
	msg := cmd()

	successMsg, ok := msg.(SuccessMsg)
	if !ok {
		t.Fatalf("Expected SuccessMsg, got %T", msg)
	}

	if !provider.submitReviewCalled {
		t.Error("Expected SubmitReview to be called")
	}

	if provider.lastReview.Action != domain.ReviewActionComment {
		t.Errorf("Expected action to remain comment, got %s", provider.lastReview.Action)
	}

	if !successMsg.reloadComments {
		t.Error("Expected reloadComments to be true")
	}
}

func TestOwnPRValidation_OtherUserPRNotConverted(t *testing.T) {
	repo := &mockRepository{
		pats: map[string]*domain.PAT{
			"pat-1": {
				ID:       "pat-1",
				Username: "testuser",
				Provider: domain.ProviderGitHub,
			},
		},
	}

	provider := &mockProvider{}

	prInspect := views.NewPRInspectView()
	reviewView := views.NewReviewView()

	m := Model{
		ctx:        context.Background(),
		repository: repo,
		prInspect:  prInspect,
		reviewView: reviewView,
		providers: map[string]domain.Provider{
			"pat-1": provider,
		},
	}

	pr := &domain.PullRequest{
		Number: 1,
		Author: domain.User{
			Username: "otheruser",
		},
		Repository: domain.Repo{
			FullName: "owner/repo",
		},
		PATID:        "pat-1",
		ProviderType: domain.ProviderGitHub,
	}

	m.prInspect.SetPR(pr)
	m.reviewView.Activate(views.ReviewModeApprove)

	cmd := m.submitReview()
	msg := cmd()

	successMsg, ok := msg.(SuccessMsg)
	if !ok {
		t.Fatalf("Expected SuccessMsg, got %T", msg)
	}

	if !provider.submitReviewCalled {
		t.Error("Expected SubmitReview to be called")
	}

	if provider.lastReview.Action != domain.ReviewActionApprove {
		t.Errorf("Expected action to remain approve for other user's PR, got %s", provider.lastReview.Action)
	}

	if !successMsg.reloadComments {
		t.Error("Expected reloadComments to be true")
	}
}

func TestOwnPRValidation_NoPATIDNoConversion(t *testing.T) {
	repo := &mockRepository{
		pats: map[string]*domain.PAT{},
	}

	provider := &mockProvider{}

	prInspect := views.NewPRInspectView()
	reviewView := views.NewReviewView()

	m := Model{
		ctx:        context.Background(),
		repository: repo,
		prInspect:  prInspect,
		reviewView: reviewView,
		providers: map[string]domain.Provider{
			"": provider,
		},
		provider: provider,
	}

	pr := &domain.PullRequest{
		Number: 1,
		Author: domain.User{
			Username: "testuser",
		},
		Repository: domain.Repo{
			FullName: "owner/repo",
		},
		PATID:        "",
		ProviderType: domain.ProviderGitHub,
	}

	m.prInspect.SetPR(pr)
	m.reviewView.Activate(views.ReviewModeApprove)

	cmd := m.submitReview()
	msg := cmd()

	_, ok := msg.(SuccessMsg)
	if !ok {
		t.Fatalf("Expected SuccessMsg, got %T", msg)
	}

	if !provider.submitReviewCalled {
		t.Error("Expected SubmitReview to be called")
	}

	if provider.lastReview.Action != domain.ReviewActionApprove {
		t.Errorf("Expected action to remain approve when no PATID, got %s", provider.lastReview.Action)
	}
}

func TestSubmitReview_ApproveOtherUserPR_SubmitsApproval(t *testing.T) {
	repo := &mockRepository{
		pats: map[string]*domain.PAT{
			"pat-1": {
				ID:       "pat-1",
				Username: "reviewer",
				Provider: domain.ProviderGitHub,
			},
		},
	}

	provider := &mockProvider{}

	prInspect := views.NewPRInspectView()
	reviewView := views.NewReviewView()

	m := Model{
		ctx:        context.Background(),
		repository: repo,
		prInspect:  prInspect,
		reviewView: reviewView,
		providers: map[string]domain.Provider{
			"pat-1": provider,
		},
	}

	pr := &domain.PullRequest{
		Number: 42,
		Author: domain.User{
			Username: "prauthor",
		},
		Repository: domain.Repo{
			FullName: "owner/repo",
		},
		PATID:        "pat-1",
		ProviderType: domain.ProviderGitHub,
	}

	m.prInspect.SetPR(pr)
	m.reviewView.Activate(views.ReviewModeApprove)

	cmd := m.submitReview()
	msg := cmd()

	_, ok := msg.(SuccessMsg)
	if !ok {
		t.Fatalf("Expected SuccessMsg, got %T", msg)
	}

	if !provider.submitReviewCalled {
		t.Error("Expected SubmitReview to be called")
	}

	if provider.lastReview.Action != domain.ReviewActionApprove {
		t.Errorf("Expected action to be approve for other user's PR, got %s", provider.lastReview.Action)
	}

	if provider.lastReview.PRIdentifier != "owner/repo/42" {
		t.Errorf("Expected PR identifier to be owner/repo/42, got %s", provider.lastReview.PRIdentifier)
	}
}

func TestSubmitReview_RequestChanges_SubmitsCorrectAction(t *testing.T) {
	repo := &mockRepository{
		pats: map[string]*domain.PAT{
			"pat-1": {
				ID:       "pat-1",
				Username: "reviewer",
				Provider: domain.ProviderGitHub,
			},
		},
	}

	provider := &mockProvider{}

	prInspect := views.NewPRInspectView()
	reviewView := views.NewReviewView()

	m := Model{
		ctx:        context.Background(),
		repository: repo,
		prInspect:  prInspect,
		reviewView: reviewView,
		providers: map[string]domain.Provider{
			"pat-1": provider,
		},
	}

	pr := &domain.PullRequest{
		Number: 42,
		Author: domain.User{
			Username: "prauthor",
		},
		Repository: domain.Repo{
			FullName: "owner/repo",
		},
		PATID:        "pat-1",
		ProviderType: domain.ProviderGitHub,
	}

	m.prInspect.SetPR(pr)
	m.reviewView.Activate(views.ReviewModeRequestChanges)

	cmd := m.submitReview()
	msg := cmd()

	_, ok := msg.(SuccessMsg)
	if !ok {
		t.Fatalf("Expected SuccessMsg, got %T", msg)
	}

	if provider.lastReview.Action != domain.ReviewActionRequestChanges {
		t.Errorf("Expected action to be request_changes, got %s", provider.lastReview.Action)
	}
}

func TestSubmitReview_WithInlineComments_IncludesComments(t *testing.T) {
	repo := &mockRepository{
		pats: map[string]*domain.PAT{
			"pat-1": {
				ID:       "pat-1",
				Username: "reviewer",
				Provider: domain.ProviderGitHub,
			},
		},
	}

	provider := &mockProvider{}

	prInspect := views.NewPRInspectView()
	prInspect.SetSize(80, 24)
	reviewView := views.NewReviewView()

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "file1.go",
				Hunks: []domain.DiffHunk{
					{Header: "@@ -1,1 +1,1 @@", Lines: []domain.DiffLine{
						{Type: "add", Content: "+line1", NewLine: 1},
					}},
				},
			},
		},
	}
	prInspect.SetDiff(diff)
	prInspect.SwitchToDiff()
	prInspect.AddPendingComment("This needs fixing")

	m := Model{
		ctx:        context.Background(),
		repository: repo,
		prInspect:  prInspect,
		reviewView: reviewView,
		providers: map[string]domain.Provider{
			"pat-1": provider,
		},
	}

	pr := &domain.PullRequest{
		Number: 42,
		Author: domain.User{
			Username: "prauthor",
		},
		Repository: domain.Repo{
			FullName: "owner/repo",
		},
		PATID:        "pat-1",
		ProviderType: domain.ProviderGitHub,
	}

	m.prInspect.SetPR(pr)
	m.reviewView.Activate(views.ReviewModeApprove)

	cmd := m.submitReview()
	msg := cmd()

	_, ok := msg.(SuccessMsg)
	if !ok {
		t.Fatalf("Expected SuccessMsg, got %T", msg)
	}

	if len(provider.lastReview.Comments) != 1 {
		t.Errorf("Expected 1 inline comment, got %d", len(provider.lastReview.Comments))
	}

	if len(provider.lastReview.Comments) > 0 {
		if provider.lastReview.Comments[0].Body != "This needs fixing" {
			t.Errorf("Expected comment body to be 'This needs fixing', got '%s'", provider.lastReview.Comments[0].Body)
		}
		if provider.lastReview.Comments[0].FilePath != "file1.go" {
			t.Errorf("Expected comment file path to be 'file1.go', got '%s'", provider.lastReview.Comments[0].FilePath)
		}
	}
}

func TestSubmitReview_AzureDevOps_SubmitsCorrectly(t *testing.T) {
	repo := &mockRepository{
		pats: map[string]*domain.PAT{
			"pat-1": {
				ID:           "pat-1",
				Username:     "reviewer",
				Provider:     domain.ProviderAzureDevOps,
				Organization: "myorg",
			},
		},
	}

	provider := &mockProvider{}

	prInspect := views.NewPRInspectView()
	reviewView := views.NewReviewView()

	m := Model{
		ctx:        context.Background(),
		repository: repo,
		prInspect:  prInspect,
		reviewView: reviewView,
		providers: map[string]domain.Provider{
			"pat-1": provider,
		},
	}

	pr := &domain.PullRequest{
		Number: 42,
		Author: domain.User{
			Username: "prauthor",
		},
		Repository: domain.Repo{
			FullName: "MyProject/MyRepo",
		},
		PATID:        "pat-1",
		ProviderType: domain.ProviderAzureDevOps,
	}

	m.prInspect.SetPR(pr)
	m.reviewView.Activate(views.ReviewModeApprove)

	cmd := m.submitReview()
	msg := cmd()

	_, ok := msg.(SuccessMsg)
	if !ok {
		t.Fatalf("Expected SuccessMsg, got %T", msg)
	}

	if provider.lastReview.Action != domain.ReviewActionApprove {
		t.Errorf("Expected action to be approve, got %s", provider.lastReview.Action)
	}

	if provider.lastReview.PRIdentifier != "MyProject/MyRepo/42" {
		t.Errorf("Expected PR identifier to be MyProject/MyRepo/42, got %s", provider.lastReview.PRIdentifier)
	}
}

func TestSubmitReview_NoPRSelected_ReturnsError(t *testing.T) {
	repo := &mockRepository{
		pats: map[string]*domain.PAT{},
	}

	provider := &mockProvider{}

	prInspect := views.NewPRInspectView()
	reviewView := views.NewReviewView()

	m := Model{
		ctx:        context.Background(),
		repository: repo,
		prInspect:  prInspect,
		reviewView: reviewView,
		providers: map[string]domain.Provider{
			"": provider,
		},
		provider: provider,
	}

	m.reviewView.Activate(views.ReviewModeApprove)

	cmd := m.submitReview()
	msg := cmd()

	_, ok := msg.(ErrorMsg)
	if !ok {
		t.Fatalf("Expected ErrorMsg when no PR selected, got %T", msg)
	}
}
