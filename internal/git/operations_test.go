package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExpandPath_TildeExpansion(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	path, err := ExpandPath("~/test")
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(homeDir, "test")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestExpandPath_RelativePath(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	path, err := ExpandPath("test")
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(wd, "test")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestExpandPath_EmptyPath(t *testing.T) {
	_, err := ExpandPath("")
	if err == nil {
		t.Error("Expected error for empty path")
	}
}

func TestIsGitRepo_ValidRepo(t *testing.T) {
	tmpDir := t.TempDir()

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	if !IsGitRepo(tmpDir) {
		t.Error("Expected directory to be recognized as git repo")
	}
}

func TestIsGitRepo_NotARepo(t *testing.T) {
	tmpDir := t.TempDir()

	if IsGitRepo(tmpDir) {
		t.Error("Expected directory to not be recognized as git repo")
	}
}

func TestIsGitRepo_NonExistentPath(t *testing.T) {
	if IsGitRepo("/nonexistent/path") {
		t.Error("Expected non-existent path to not be recognized as git repo")
	}
}

func TestParseRepoFullNameFromURL_GitSSH(t *testing.T) {
	url := "git@github.com:johanforsgren/lgtmfaster.git"
	expected := "johanforsgren/lgtmfaster"

	result := ParseRepoFullNameFromURL(url)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestParseRepoFullNameFromURL_HTTPS(t *testing.T) {
	url := "https://github.com/johanforsgren/lgtmfaster.git"
	expected := "johanforsgren/lgtmfaster"

	result := ParseRepoFullNameFromURL(url)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestParseRepoFullNameFromURL_HTTPSNoGit(t *testing.T) {
	url := "https://github.com/johanforsgren/lgtmfaster"
	expected := "johanforsgren/lgtmfaster"

	result := ParseRepoFullNameFromURL(url)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestListDirectoryContents_HiddenFilesExcluded(t *testing.T) {
	tmpDir := t.TempDir()

	os.Mkdir(filepath.Join(tmpDir, "visible"), 0755)
	os.Mkdir(filepath.Join(tmpDir, ".hidden"), 0755)

	contents, err := ListDirectoryContents(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(contents) != 1 {
		t.Errorf("Expected 1 visible directory, got %d", len(contents))
	}

	if contents[0] != "visible/" {
		t.Errorf("Expected 'visible/', got %s", contents[0])
	}
}

func TestListDirectoryContents_OnlyDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	os.Mkdir(filepath.Join(tmpDir, "dir1"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)

	contents, err := ListDirectoryContents(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(contents) != 1 {
		t.Errorf("Expected 1 directory, got %d", len(contents))
	}

	if contents[0] != "dir1/" {
		t.Errorf("Expected 'dir1/', got %s", contents[0])
	}
}

func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git init failed, skipping test: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	return tmpDir
}

func TestGetCurrentBranch(t *testing.T) {
	tmpDir := setupTestGitRepo(t)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git commit failed, skipping test: %v", err)
	}

	branch, err := GetCurrentBranch(tmpDir)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if branch != "main" && branch != "master" {
		t.Errorf("Expected branch to be main or master, got %s", branch)
	}
}

func TestHasUncommittedChanges_Clean(t *testing.T) {
	tmpDir := setupTestGitRepo(t)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git commit failed, skipping test: %v", err)
	}

	hasChanges, err := HasUncommittedChanges(tmpDir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges failed: %v", err)
	}

	if hasChanges {
		t.Error("Expected no uncommitted changes")
	}
}

func TestHasUncommittedChanges_Dirty(t *testing.T) {
	tmpDir := setupTestGitRepo(t)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	hasChanges, err := HasUncommittedChanges(tmpDir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges failed: %v", err)
	}

	if !hasChanges {
		t.Error("Expected uncommitted changes")
	}
}

func TestGetRepoStatus(t *testing.T) {
	tmpDir := setupTestGitRepo(t)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git commit failed, skipping test: %v", err)
	}

	status, err := GetRepoStatus(tmpDir)
	if err != nil {
		t.Fatalf("GetRepoStatus failed: %v", err)
	}

	if !status.IsGitRepo {
		t.Error("Expected IsGitRepo to be true")
	}

	if status.CurrentBranch != "main" && status.CurrentBranch != "master" {
		t.Errorf("Expected branch to be main or master, got %s", status.CurrentBranch)
	}

	if status.HasUncommittedChanges {
		t.Error("Expected no uncommitted changes")
	}

	if status.LocalPath != tmpDir {
		t.Errorf("Expected LocalPath %s, got %s", tmpDir, status.LocalPath)
	}
}

func TestCheckoutBranch_WithUncommittedChanges(t *testing.T) {
	tmpDir := setupTestGitRepo(t)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	err := CheckoutBranch(tmpDir, "newbranch")
	if err == nil {
		t.Error("Expected error when checking out with uncommitted changes")
	}
}

func TestPull_WithUncommittedChanges(t *testing.T) {
	tmpDir := setupTestGitRepo(t)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	err := Pull(tmpDir)
	if err == nil {
		t.Error("Expected error when pulling with uncommitted changes")
	}
}
