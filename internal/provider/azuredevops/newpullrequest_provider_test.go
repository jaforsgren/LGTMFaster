package azuredevops

import (
	"context"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

const (
	testUserID     = "fb02028f-4708-6070-83a0-d6fb0136613e"
	testRepository = "Proj/api"
)

func newPullRequestTestProvider(gitClient *mockGitClient) *Provider {
	return &Provider{
		client: &Client{gitClient: gitClient, organization: testOrganization, userID: testUserID},
		repoCache: map[string]*ResolvedRepository{
			testRepository: {ProjectID: "proj-id", RepoID: "repo-id", CachedAt: time.Now()},
		},
		cacheTTL: time.Hour,
	}
}

func refUpdate(name, newObjectID string) git.GitRefUpdate {
	return git.GitRefUpdate{Name: strPtr(name), NewObjectId: strPtr(newObjectID)}
}

func TestPushedBranches_KeepsCreatedAndUpdatedBranchesOnly(t *testing.T) {
	pushedAt := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	updates := []git.GitRefUpdate{
		refUpdate("refs/heads/feature/login", "abc"),
		refUpdate("refs/heads/old-branch", deletedRefObjectID),
		refUpdate("refs/tags/v1.0", "def"),
	}
	pushes := []git.GitPush{
		{Date: &azuredevops.Time{Time: pushedAt}, RefUpdates: &updates},
		{RefUpdates: &updates},
	}
	repo := domain.Repo{FullName: testRepository}

	branches := pushedBranches(pushes, repo)

	if len(branches) != 1 {
		t.Fatalf("expected only the updated branch, got %+v", branches)
	}
	if branches[0].Branch != "feature/login" || !branches[0].PushedAt.Equal(pushedAt) || branches[0].Key() != "Proj/api:feature/login" {
		t.Errorf("unexpected branch %+v", branches[0])
	}
}

func TestListRecentPushes_SearchesTheUsersPushesSince(t *testing.T) {
	gitClient := &mockGitClient{pushes: &[]git.GitPush{}}
	provider := newPullRequestTestProvider(gitClient)
	since := time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)

	if _, err := provider.client.ListRecentPushes(context.Background(), "proj-id", "repo-id", since); err != nil {
		t.Fatal(err)
	}

	criteria := gitClient.lastPushesArgs.SearchCriteria
	if !criteria.FromDate.Time.Equal(since) || criteria.PusherId.String() != testUserID || !*criteria.IncludeRefUpdates {
		t.Errorf("unexpected search criteria %+v", criteria)
	}
}

func TestListBranches_ReturnsSortedBranchesOfEveryPageAndTheDefault(t *testing.T) {
	gitClient := &mockGitClient{
		repository: &git.GitRepository{DefaultBranch: strPtr("refs/heads/main")},
		refPages: []*git.GetRefsResponseValue{
			{Value: []git.GitRef{{Name: strPtr("refs/heads/main")}, {Name: strPtr("refs/heads/develop")}}, ContinuationToken: "next"},
			{Value: []git.GitRef{{Name: strPtr("refs/heads/release")}}},
		},
	}

	branches, err := newPullRequestTestProvider(gitClient).ListBranches(context.Background(), testRepository)
	if err != nil {
		t.Fatal(err)
	}

	if branches.Default != "main" {
		t.Errorf("expected main as default, got %q", branches.Default)
	}
	expected := []string{"develop", "main", "release"}
	if len(branches.Names) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, branches.Names)
	}
	for i, name := range expected {
		if branches.Names[i] != name {
			t.Errorf("expected %v, got %v", expected, branches.Names)
		}
	}
}

func TestCreatePullRequest_SendsBranchRefsAndDraftFlag(t *testing.T) {
	prID := 42
	gitClient := &mockGitClient{createdPR: &git.GitPullRequest{
		PullRequestId: &prID,
		Title:         strPtr("Add login"),
		CreationDate:  &azuredevops.Time{Time: time.Now()},
	}}

	pr, err := newPullRequestTestProvider(gitClient).CreatePullRequest(context.Background(), domain.NewPullRequest{
		RepositoryFullName: testRepository,
		SourceBranch:       "feature/login",
		TargetBranch:       "main",
		Title:              "Add login",
		Description:        "Adds the login page",
		IsDraft:            true,
	})
	if err != nil {
		t.Fatal(err)
	}

	sent := gitClient.lastCreateArgs.GitPullRequestToCreate
	if *sent.SourceRefName != "refs/heads/feature/login" || *sent.TargetRefName != "refs/heads/main" {
		t.Errorf("expected full branch refs, got %s -> %s", *sent.SourceRefName, *sent.TargetRefName)
	}
	if !*sent.IsDraft || *sent.Title != "Add login" || *sent.Description != "Adds the login page" {
		t.Errorf("unexpected pull request sent %+v", sent)
	}
	if *gitClient.lastCreateArgs.RepositoryId != "repo-id" || *gitClient.lastCreateArgs.Project != "proj-id" {
		t.Error("expected the resolved project and repository IDs")
	}
	if pr.Number != prID || pr.URL == "" {
		t.Errorf("expected the created pull request with a URL, got %+v", pr)
	}
}
