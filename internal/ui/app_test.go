package ui

import (
	"context"
	"fmt"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

type mockRepository struct {
	pats map[string]*domain.PAT
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

type mockProvider struct {
	submitReviewCalled bool
	lastReview         domain.Review
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
