package azuredevops

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

type mockGitClient struct {
	iterations       *[]git.GitPullRequestIteration
	iterationChanges *git.GitPullRequestIterationChanges
	blobContent      map[string]string
	getIterationsErr error
	getChangesErr    error
	getBlobErr       error
}

func (m *mockGitClient) GetRepositories(ctx context.Context, args git.GetRepositoriesArgs) (*[]git.GitRepository, error) {
	return nil, nil
}

func (m *mockGitClient) GetPullRequests(ctx context.Context, args git.GetPullRequestsArgs) (*[]git.GitPullRequest, error) {
	return nil, nil
}

func (m *mockGitClient) GetPullRequest(ctx context.Context, args git.GetPullRequestArgs) (*git.GitPullRequest, error) {
	return nil, nil
}

func (m *mockGitClient) GetPullRequestCommits(ctx context.Context, args git.GetPullRequestCommitsArgs) (*git.GetPullRequestCommitsResponseValue, error) {
	return nil, nil
}

func (m *mockGitClient) GetPullRequestIterations(ctx context.Context, args git.GetPullRequestIterationsArgs) (*[]git.GitPullRequestIteration, error) {
	if m.getIterationsErr != nil {
		return nil, m.getIterationsErr
	}
	return m.iterations, nil
}

func (m *mockGitClient) GetPullRequestIterationChanges(ctx context.Context, args git.GetPullRequestIterationChangesArgs) (*git.GitPullRequestIterationChanges, error) {
	if m.getChangesErr != nil {
		return nil, m.getChangesErr
	}
	return m.iterationChanges, nil
}

func (m *mockGitClient) GetBlobContent(ctx context.Context, args git.GetBlobContentArgs) (io.ReadCloser, error) {
	if m.getBlobErr != nil {
		return nil, m.getBlobErr
	}
	if args.Sha1 == nil {
		return nil, nil
	}
	content, exists := m.blobContent[*args.Sha1]
	if !exists {
		return nil, nil
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func (m *mockGitClient) GetThreads(ctx context.Context, args git.GetThreadsArgs) (*[]git.GitPullRequestCommentThread, error) {
	return nil, nil
}

func (m *mockGitClient) CreateThread(ctx context.Context, args git.CreateThreadArgs) (*git.GitPullRequestCommentThread, error) {
	return nil, nil
}

func (m *mockGitClient) CreatePullRequestReviewer(ctx context.Context, args git.CreatePullRequestReviewerArgs) (*git.IdentityRefWithVote, error) {
	return nil, nil
}

func (m *mockGitClient) UpdatePullRequest(ctx context.Context, args git.UpdatePullRequestArgs) (*git.GitPullRequest, error) {
	return nil, nil
}

func TestGetPullRequestIterationChanges_AddedFile(t *testing.T) {
	iterationID := 1
	changeType := git.VersionControlChangeTypeValues.Add

	iterations := []git.GitPullRequestIteration{
		{Id: &iterationID},
	}

	path := "/src/newfile.go"
	objectId := "abc123"
	isFolder := false
	item := map[string]interface{}{
		"path":     path,
		"objectId": objectId,
		"isFolder": isFolder,
	}

	changes := git.GitPullRequestIterationChanges{
		ChangeEntries: &[]git.GitPullRequestChange{
			{
				ChangeType: &changeType,
				Item:       item,
			},
		},
	}

	mockClient := &mockGitClient{
		iterations:       &iterations,
		iterationChanges: &changes,
		blobContent: map[string]string{
			objectId: "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}",
		},
	}

	client := &Client{
		gitClient: mockClient,
	}

	result, err := client.GetPullRequestIterationChanges(context.Background(), "project1", "repo1", 42)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.Contains(result, "diff --git a/src/newfile.go b/src/newfile.go") {
		t.Errorf("Expected diff header for added file")
	}
	if !strings.Contains(result, "--- /dev/null") {
		t.Errorf("Expected /dev/null for added file")
	}
	if !strings.Contains(result, "+++ b/src/newfile.go") {
		t.Errorf("Expected +++ b/src/newfile.go for added file")
	}
	if !strings.Contains(result, "+package main") {
		t.Errorf("Expected added line with + prefix")
	}
}

func TestGetPullRequestIterationChanges_DeletedFile(t *testing.T) {
	iterationID := 1
	changeType := git.VersionControlChangeTypeValues.Delete

	iterations := []git.GitPullRequestIteration{
		{Id: &iterationID},
	}

	path := "/src/oldfile.go"
	originalObjectId := "def456"
	isFolder := false
	item := map[string]interface{}{
		"path":             path,
		"originalObjectId": originalObjectId,
		"isFolder":         isFolder,
		"originalPath":     path,
	}

	changes := git.GitPullRequestIterationChanges{
		ChangeEntries: &[]git.GitPullRequestChange{
			{
				ChangeType: &changeType,
				Item:       item,
			},
		},
	}

	mockClient := &mockGitClient{
		iterations:       &iterations,
		iterationChanges: &changes,
		blobContent: map[string]string{
			originalObjectId: "package main\n\nfunc old() {\n\tprintln(\"Goodbye\")\n}",
		},
	}

	client := &Client{
		gitClient: mockClient,
	}

	result, err := client.GetPullRequestIterationChanges(context.Background(), "project1", "repo1", 42)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.Contains(result, "diff --git a/src/oldfile.go b/src/oldfile.go") {
		t.Errorf("Expected diff header for deleted file")
	}
	if !strings.Contains(result, "--- a/src/oldfile.go") {
		t.Errorf("Expected --- a/src/oldfile.go for deleted file")
	}
	if !strings.Contains(result, "+++ /dev/null") {
		t.Errorf("Expected /dev/null for deleted file")
	}
	if !strings.Contains(result, "-package main") {
		t.Errorf("Expected deleted line with - prefix")
	}
}

func TestGetPullRequestIterationChanges_EditedFile(t *testing.T) {
	iterationID := 1
	changeType := git.VersionControlChangeTypeValues.Edit

	iterations := []git.GitPullRequestIteration{
		{Id: &iterationID},
	}

	path := "/src/modified.go"
	objectId := "new789"
	originalObjectId := "old789"
	isFolder := false
	item := map[string]interface{}{
		"path":             path,
		"objectId":         objectId,
		"originalObjectId": originalObjectId,
		"isFolder":         isFolder,
		"originalPath":     path,
	}

	changes := git.GitPullRequestIterationChanges{
		ChangeEntries: &[]git.GitPullRequestChange{
			{
				ChangeType: &changeType,
				Item:       item,
			},
		},
	}

	mockClient := &mockGitClient{
		iterations:       &iterations,
		iterationChanges: &changes,
		blobContent: map[string]string{
			originalObjectId: "package main\n\nfunc hello() {\n\tprintln(\"Hello\")\n}",
			objectId:         "package main\n\nfunc hello() {\n\tprintln(\"Hello World\")\n}",
		},
	}

	client := &Client{
		gitClient: mockClient,
	}

	result, err := client.GetPullRequestIterationChanges(context.Background(), "project1", "repo1", 42)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.Contains(result, "diff --git a/src/modified.go b/src/modified.go") {
		t.Errorf("Expected diff header for edited file")
	}
	if !strings.Contains(result, "--- a/src/modified.go") {
		t.Errorf("Expected --- a/src/modified.go for edited file")
	}
	if !strings.Contains(result, "+++ b/src/modified.go") {
		t.Errorf("Expected +++ b/src/modified.go for edited file")
	}
	if !strings.Contains(result, "-\tprintln(\"Hello\")") {
		t.Errorf("Expected old line with - prefix")
	}
	if !strings.Contains(result, "+\tprintln(\"Hello World\")") {
		t.Errorf("Expected new line with + prefix")
	}
}

func TestGetPullRequestIterationChanges_SkipsFolders(t *testing.T) {
	iterationID := 1
	changeType := git.VersionControlChangeTypeValues.Add

	iterations := []git.GitPullRequestIteration{
		{Id: &iterationID},
	}

	path := "/src"
	isFolder := true
	item := map[string]interface{}{
		"path":     path,
		"isFolder": isFolder,
	}

	changes := git.GitPullRequestIterationChanges{
		ChangeEntries: &[]git.GitPullRequestChange{
			{
				ChangeType: &changeType,
				Item:       item,
			},
		},
	}

	mockClient := &mockGitClient{
		iterations:       &iterations,
		iterationChanges: &changes,
		blobContent:      map[string]string{},
	}

	client := &Client{
		gitClient: mockClient,
	}

	result, err := client.GetPullRequestIterationChanges(context.Background(), "project1", "repo1", 42)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty result for folder changes, got: %s", result)
	}
}

func TestGetPullRequestIterationChanges_MultipleFiles(t *testing.T) {
	iterationID := 1
	addType := git.VersionControlChangeTypeValues.Add
	editType := git.VersionControlChangeTypeValues.Edit

	iterations := []git.GitPullRequestIteration{
		{Id: &iterationID},
	}

	path1 := "/src/new.go"
	objectId1 := "abc111"
	isFolder1 := false
	item1 := map[string]interface{}{
		"path":     path1,
		"objectId": objectId1,
		"isFolder": isFolder1,
	}

	path2 := "/src/edit.go"
	objectId2 := "abc222"
	originalObjectId2 := "old222"
	isFolder2 := false
	item2 := map[string]interface{}{
		"path":             path2,
		"objectId":         objectId2,
		"originalObjectId": originalObjectId2,
		"isFolder":         isFolder2,
		"originalPath":     path2,
	}

	changes := git.GitPullRequestIterationChanges{
		ChangeEntries: &[]git.GitPullRequestChange{
			{
				ChangeType: &addType,
				Item:       item1,
			},
			{
				ChangeType: &editType,
				Item:       item2,
			},
		},
	}

	mockClient := &mockGitClient{
		iterations:       &iterations,
		iterationChanges: &changes,
		blobContent: map[string]string{
			objectId1:         "package main",
			objectId2:         "package test\n",
			originalObjectId2: "package main\n",
		},
	}

	client := &Client{
		gitClient: mockClient,
	}

	result, err := client.GetPullRequestIterationChanges(context.Background(), "project1", "repo1", 42)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.Contains(result, "diff --git a/src/new.go b/src/new.go") {
		t.Errorf("Expected diff for first file")
	}
	if !strings.Contains(result, "diff --git a/src/edit.go b/src/edit.go") {
		t.Errorf("Expected diff for second file")
	}
}

func TestGetPullRequestIterationChanges_EmptyIteration(t *testing.T) {
	iterations := []git.GitPullRequestIteration{}

	mockClient := &mockGitClient{
		iterations:  &iterations,
		blobContent: map[string]string{},
	}

	client := &Client{
		gitClient: mockClient,
	}

	_, err := client.GetPullRequestIterationChanges(context.Background(), "project1", "repo1", 42)
	if err == nil {
		t.Errorf("Expected error for empty iterations")
	}
	if !strings.Contains(err.Error(), "no iterations found") {
		t.Errorf("Expected 'no iterations found' error, got: %v", err)
	}
}

func TestGetPullRequestIterationChanges_NoChanges(t *testing.T) {
	iterationID := 1
	iterations := []git.GitPullRequestIteration{
		{Id: &iterationID},
	}

	changes := git.GitPullRequestIterationChanges{
		ChangeEntries: &[]git.GitPullRequestChange{},
	}

	mockClient := &mockGitClient{
		iterations:       &iterations,
		iterationChanges: &changes,
		blobContent:      map[string]string{},
	}

	client := &Client{
		gitClient: mockClient,
	}

	_, err := client.GetPullRequestIterationChanges(context.Background(), "project1", "repo1", 42)
	if err == nil {
		t.Errorf("Expected error for no changes")
	}
	if !strings.Contains(err.Error(), "no changes found") {
		t.Errorf("Expected 'no changes found' error, got: %v", err)
	}
}
