package azuredevops

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func mustParseUUID(s string) uuid.UUID {
	u, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func createMockPR(prID int, title string, webURL *string) *git.GitPullRequest {
	now := time.Now()
	status := git.PullRequestStatusValues.Active
	mergeStatus := git.PullRequestAsyncStatusValues.Succeeded
	creationDate := azuredevops.Time{Time: now}
	repoName := "TestRepo"
	repoID := mustParseUUID("12345678-1234-1234-1234-123456789012")

	pr := &git.GitPullRequest{
		PullRequestId: &prID,
		Title:         &title,
		CreationDate:  &creationDate,
		Status:        &status,
		MergeStatus:   &mergeStatus,
	}

	if webURL != nil {
		pr.Repository = &git.GitRepository{
			WebUrl: webURL,
			Id:     &repoID,
			Name:   &repoName,
		}
	} else {
		pr.Repository = &git.GitRepository{
			Id:   &repoID,
			Name: &repoName,
		}
	}

	return pr
}

func TestBuildPRURL(t *testing.T) {
	tests := []struct {
		name        string
		org         string
		projectName string
		repoName    string
		prNumber    int
		expected    string
	}{
		{
			name:        "standard URL construction",
			org:         "myorg",
			projectName: "MyProject",
			repoName:    "MyRepo",
			prNumber:    123,
			expected:    "https://dev.azure.com/myorg/MyProject/_git/MyRepo/pullrequest/123",
		},
		{
			name:        "URL with spaces in project name",
			org:         "weapp",
			projectName: "My Project",
			repoName:    "api",
			prNumber:    456,
			expected:    "https://dev.azure.com/weapp/My Project/_git/api/pullrequest/456",
		},
		{
			name:        "URL with numbers",
			org:         "org123",
			projectName: "Project2024",
			repoName:    "repo-v2",
			prNumber:    1,
			expected:    "https://dev.azure.com/org123/Project2024/_git/repo-v2/pullrequest/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				organization: tt.org,
			}
			provider := &Provider{
				client: client,
			}

			result := provider.buildPRURL(tt.projectName, tt.repoName, tt.prNumber)
			if result != tt.expected {
				t.Errorf("buildPRURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConvertPullRequest_URLFromAPI(t *testing.T) {
	prID := 42
	title := "Test PR"
	webURL := "https://dev.azure.com/myorg/MyProject/_git/MyRepo"
	expectedURL := "https://dev.azure.com/myorg/MyProject/_git/MyRepo/pullrequest/42"

	adoPR := createMockPR(prID, title, &webURL)

	result := convertPullRequest(adoPR, "testuser")

	if result.URL != expectedURL {
		t.Errorf("convertPullRequest() URL = %q, want %q", result.URL, expectedURL)
	}
}

func TestConvertPullRequest_EmptyURLWhenNoRepo(t *testing.T) {
	prID := 42
	title := "Test PR"

	adoPR := createMockPR(prID, title, nil)
	adoPR.Repository = nil

	result := convertPullRequest(adoPR, "testuser")

	if result.URL != "" {
		t.Errorf("convertPullRequest() URL = %q, want empty string", result.URL)
	}
}

func TestConvertPullRequest_EmptyURLWhenNoWebURL(t *testing.T) {
	prID := 42
	title := "Test PR"

	adoPR := createMockPR(prID, title, nil)

	result := convertPullRequest(adoPR, "testuser")

	if result.URL != "" {
		t.Errorf("convertPullRequest() URL = %q, want empty string", result.URL)
	}
}

func TestBuildPRWebURL(t *testing.T) {
	tests := []struct {
		name     string
		pr       *git.GitPullRequest
		expected string
	}{
		{
			name: "valid PR with repository URL",
			pr: &git.GitPullRequest{
				PullRequestId: intPtr(123),
				Repository: &git.GitRepository{
					WebUrl: strPtr("https://dev.azure.com/org/project/_git/repo"),
				},
			},
			expected: "https://dev.azure.com/org/project/_git/repo/pullrequest/123",
		},
		{
			name: "PR without repository",
			pr: &git.GitPullRequest{
				PullRequestId: intPtr(123),
				Repository:    nil,
			},
			expected: "",
		},
		{
			name: "PR without WebUrl",
			pr: &git.GitPullRequest{
				PullRequestId: intPtr(123),
				Repository: &git.GitRepository{
					WebUrl: nil,
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPRWebURL(tt.pr)
			if result != tt.expected {
				t.Errorf("buildPRWebURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestConvertReviewActionToVote(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		expected int
	}{
		{
			name:     "approve returns 10",
			action:   "approve",
			expected: 10,
		},
		{
			name:     "request_changes returns -10",
			action:   "request_changes",
			expected: -10,
		},
		{
			name:     "comment returns 0",
			action:   "comment",
			expected: 0,
		},
		{
			name:     "unknown action returns 0",
			action:   "unknown",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var action domain.ReviewAction
			switch tt.action {
			case "approve":
				action = domain.ReviewActionApprove
			case "request_changes":
				action = domain.ReviewActionRequestChanges
			case "comment":
				action = domain.ReviewActionComment
			default:
				action = domain.ReviewAction(tt.action)
			}

			result := convertReviewActionToVote(action)
			if result != tt.expected {
				t.Errorf("convertReviewActionToVote(%s) = %d, want %d", tt.action, result, tt.expected)
			}
		})
	}
}
