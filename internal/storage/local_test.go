package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestNewLocalRepository(t *testing.T) {
	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	if repo == nil {
		t.Fatal("Repository is nil")
	}
}

func TestSaveAndLoadPAT(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	testPAT := domain.PAT{
		ID:       "test-id",
		Name:     "Test PAT",
		Token:    "ghp_test123",
		Provider: domain.ProviderGitHub,
		Username: "testuser",
		IsActive: false,
	}

	err = repo.SavePAT(testPAT)
	if err != nil {
		t.Fatalf("Failed to save PAT: %v", err)
	}

	pats, err := repo.ListPATs()
	if err != nil {
		t.Fatalf("Failed to list PATs: %v", err)
	}

	if len(pats) != 1 {
		t.Fatalf("Expected 1 PAT, got %d", len(pats))
	}

	if pats[0].Name != testPAT.Name {
		t.Errorf("Expected PAT name %s, got %s", testPAT.Name, pats[0].Name)
	}

	if pats[0].Token != testPAT.Token {
		t.Errorf("Expected PAT token %s, got %s", testPAT.Token, pats[0].Token)
	}
}

func TestUpdatePAT(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	originalPAT := domain.PAT{
		ID:       "test-id",
		Name:     "Original Name",
		Token:    "ghp_original",
		Provider: domain.ProviderGitHub,
		Username: "original",
		IsActive: false,
	}

	err = repo.SavePAT(originalPAT)
	if err != nil {
		t.Fatalf("Failed to save original PAT: %v", err)
	}

	updatedPAT := domain.PAT{
		ID:       "test-id",
		Name:     "Updated Name",
		Token:    "ghp_updated",
		Provider: domain.ProviderGitHub,
		Username: "updated",
		IsActive: false,
	}

	err = repo.SavePAT(updatedPAT)
	if err != nil {
		t.Fatalf("Failed to update PAT: %v", err)
	}

	pats, err := repo.ListPATs()
	if err != nil {
		t.Fatalf("Failed to list PATs: %v", err)
	}

	if len(pats) != 1 {
		t.Fatalf("Expected 1 PAT after update, got %d", len(pats))
	}

	if pats[0].Name != "Updated Name" {
		t.Errorf("Expected updated name 'Updated Name', got %s", pats[0].Name)
	}

	if pats[0].Token != "ghp_updated" {
		t.Errorf("Expected updated token 'ghp_updated', got %s", pats[0].Token)
	}

	if pats[0].Username != "updated" {
		t.Errorf("Expected updated username 'updated', got %s", pats[0].Username)
	}
}

func TestDeletePAT(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	testPAT := domain.PAT{
		ID:       "test-id",
		Name:     "Test PAT",
		Token:    "ghp_test123",
		Provider: domain.ProviderGitHub,
		Username: "testuser",
		IsActive: false,
	}

	err = repo.SavePAT(testPAT)
	if err != nil {
		t.Fatalf("Failed to save PAT: %v", err)
	}

	err = repo.DeletePAT("test-id")
	if err != nil {
		t.Fatalf("Failed to delete PAT: %v", err)
	}

	pats, err := repo.ListPATs()
	if err != nil {
		t.Fatalf("Failed to list PATs: %v", err)
	}

	if len(pats) != 0 {
		t.Fatalf("Expected 0 PATs after delete, got %d", len(pats))
	}
}

func TestSetActivePAT(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	testPAT := domain.PAT{
		ID:       "test-id",
		Name:     "Test PAT",
		Token:    "ghp_test123",
		Provider: domain.ProviderGitHub,
		Username: "testuser",
		IsActive: false,
	}

	err = repo.SavePAT(testPAT)
	if err != nil {
		t.Fatalf("Failed to save PAT: %v", err)
	}

	err = repo.SetActivePAT("test-id")
	if err != nil {
		t.Fatalf("Failed to set active PAT: %v", err)
	}

	activePAT, err := repo.GetActivePAT()
	if err != nil {
		t.Fatalf("Failed to get active PAT: %v", err)
	}

	if activePAT.ID != "test-id" {
		t.Errorf("Expected active PAT ID 'test-id', got %s", activePAT.ID)
	}

	if !activePAT.IsActive {
		t.Error("Expected PAT to be active, but IsActive is false")
	}
}

func TestConfigFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, ".lgtmfaster", "config.json")
	if repo.configPath != expectedPath {
		t.Errorf("Expected config path %s, got %s", expectedPath, repo.configPath)
	}
}

func TestSaveAndListRepositoryLinks(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	link := domain.RepositoryLink{
		ID:           "link-1",
		RemoteURL:    "git@github.com:owner/repo.git",
		RepoFullName: "owner/repo",
		LocalPath:    "/home/user/projects/repo",
	}

	err = repo.SaveRepositoryLink(link)
	if err != nil {
		t.Fatalf("Failed to save repository link: %v", err)
	}

	links, err := repo.ListRepositoryLinks()
	if err != nil {
		t.Fatalf("Failed to list repository links: %v", err)
	}

	if len(links) != 1 {
		t.Fatalf("Expected 1 link, got %d", len(links))
	}

	if links[0].RepoFullName != link.RepoFullName {
		t.Errorf("Expected RepoFullName %s, got %s", link.RepoFullName, links[0].RepoFullName)
	}

	if links[0].LocalPath != link.LocalPath {
		t.Errorf("Expected LocalPath %s, got %s", link.LocalPath, links[0].LocalPath)
	}
}

func TestGetRepositoryLinkByRepoFullName(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	link := domain.RepositoryLink{
		ID:           "link-1",
		RemoteURL:    "git@github.com:owner/repo.git",
		RepoFullName: "owner/repo",
		LocalPath:    "/home/user/projects/repo",
	}

	err = repo.SaveRepositoryLink(link)
	if err != nil {
		t.Fatalf("Failed to save repository link: %v", err)
	}

	found, err := repo.GetRepositoryLinkByRepoFullName("owner/repo")
	if err != nil {
		t.Fatalf("Failed to get link by repo name: %v", err)
	}

	if found == nil {
		t.Fatal("Expected to find link, got nil")
	}

	if found.ID != link.ID {
		t.Errorf("Expected ID %s, got %s", link.ID, found.ID)
	}

	notFound, err := repo.GetRepositoryLinkByRepoFullName("other/repo")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if notFound != nil {
		t.Error("Expected nil for non-existent repo, got link")
	}
}

func TestUpdateRepositoryLink(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	link := domain.RepositoryLink{
		ID:           "link-1",
		RemoteURL:    "git@github.com:owner/repo.git",
		RepoFullName: "owner/repo",
		LocalPath:    "/home/user/old-path",
	}

	err = repo.SaveRepositoryLink(link)
	if err != nil {
		t.Fatalf("Failed to save repository link: %v", err)
	}

	link.LocalPath = "/home/user/new-path"
	err = repo.SaveRepositoryLink(link)
	if err != nil {
		t.Fatalf("Failed to update repository link: %v", err)
	}

	links, err := repo.ListRepositoryLinks()
	if err != nil {
		t.Fatalf("Failed to list repository links: %v", err)
	}

	if len(links) != 1 {
		t.Fatalf("Expected 1 link, got %d", len(links))
	}

	if links[0].LocalPath != "/home/user/new-path" {
		t.Errorf("Expected updated LocalPath, got %s", links[0].LocalPath)
	}
}

func TestDeleteRepositoryLink(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	link := domain.RepositoryLink{
		ID:           "link-1",
		RemoteURL:    "git@github.com:owner/repo.git",
		RepoFullName: "owner/repo",
		LocalPath:    "/home/user/projects/repo",
	}

	err = repo.SaveRepositoryLink(link)
	if err != nil {
		t.Fatalf("Failed to save repository link: %v", err)
	}

	err = repo.DeleteRepositoryLink("link-1")
	if err != nil {
		t.Fatalf("Failed to delete repository link: %v", err)
	}

	links, err := repo.ListRepositoryLinks()
	if err != nil {
		t.Fatalf("Failed to list repository links: %v", err)
	}

	if len(links) != 0 {
		t.Errorf("Expected 0 links after deletion, got %d", len(links))
	}

	err = repo.DeleteRepositoryLink("non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent link")
	}
}

func TestGetRepositoryLink(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	link := domain.RepositoryLink{
		ID:           "link-1",
		RemoteURL:    "git@github.com:owner/repo.git",
		RepoFullName: "owner/repo",
		LocalPath:    "/home/user/projects/repo",
	}

	err = repo.SaveRepositoryLink(link)
	if err != nil {
		t.Fatalf("Failed to save repository link: %v", err)
	}

	found, err := repo.GetRepositoryLink("link-1")
	if err != nil {
		t.Fatalf("Failed to get link by ID: %v", err)
	}

	if found == nil {
		t.Fatal("Expected to find link, got nil")
	}

	if found.RepoFullName != link.RepoFullName {
		t.Errorf("Expected RepoFullName %s, got %s", link.RepoFullName, found.RepoFullName)
	}

	_, err = repo.GetRepositoryLink("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent link")
	}
}
