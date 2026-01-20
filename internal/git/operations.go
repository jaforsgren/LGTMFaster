package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/johanforsgren/lgtmfaster/internal/logger"
)

type RepoStatus struct {
	IsGitRepo           bool
	CurrentBranch       string
	HasUncommittedChanges bool
	LocalPath           string
}

func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[1:])
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}

func IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func GetCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func HasUncommittedChanges(repoPath string) (bool, error) {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

func GetRepoStatus(repoPath string) (*RepoStatus, error) {
	expandedPath, err := ExpandPath(repoPath)
	if err != nil {
		return nil, err
	}

	status := &RepoStatus{
		LocalPath: expandedPath,
		IsGitRepo: IsGitRepo(expandedPath),
	}

	if !status.IsGitRepo {
		return status, nil
	}

	branch, err := GetCurrentBranch(expandedPath)
	if err != nil {
		logger.LogError("GET_BRANCH", expandedPath, err)
	} else {
		status.CurrentBranch = branch
	}

	hasChanges, err := HasUncommittedChanges(expandedPath)
	if err != nil {
		logger.LogError("CHECK_CHANGES", expandedPath, err)
	} else {
		status.HasUncommittedChanges = hasChanges
	}

	return status, nil
}

func CheckoutBranch(repoPath, branchName string) error {
	expandedPath, err := ExpandPath(repoPath)
	if err != nil {
		return err
	}

	hasChanges, err := HasUncommittedChanges(expandedPath)
	if err != nil {
		return err
	}
	if hasChanges {
		return fmt.Errorf("cannot checkout: uncommitted changes in repository")
	}

	cmd := exec.Command("git", "-C", expandedPath, "fetch", "--prune")
	if err := cmd.Run(); err != nil {
		logger.Log("git fetch failed (continuing): %v", err)
	}

	cmd = exec.Command("git", "-C", expandedPath, "checkout", branchName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		checkoutErr := fmt.Errorf("checkout failed: %s", strings.TrimSpace(string(output)))
		cmd = exec.Command("git", "-C", expandedPath, "checkout", "-b", branchName, "origin/"+branchName)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%v; also failed to create tracking branch: %s", checkoutErr, strings.TrimSpace(string(output)))
		}
	}

	logger.Log("Checked out branch %s in %s", branchName, expandedPath)
	return nil
}

func Pull(repoPath string) error {
	expandedPath, err := ExpandPath(repoPath)
	if err != nil {
		return err
	}

	hasChanges, err := HasUncommittedChanges(expandedPath)
	if err != nil {
		return err
	}
	if hasChanges {
		return fmt.Errorf("cannot pull: uncommitted changes in repository")
	}

	cmd := exec.Command("git", "-C", expandedPath, "pull", "--ff-only")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pull failed: %s", strings.TrimSpace(string(output)))
	}

	logger.Log("Pulled latest changes in %s: %s", expandedPath, strings.TrimSpace(string(output)))
	return nil
}

func OpenFolder(path string) error {
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(expandedPath)
	if err != nil {
		return fmt.Errorf("path does not exist: %s", expandedPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", expandedPath)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", expandedPath)
	case "linux":
		cmd = exec.Command("xdg-open", expandedPath)
	case "windows":
		cmd = exec.Command("explorer", expandedPath)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open folder: %w", err)
	}

	logger.Log("Opened folder: %s", expandedPath)
	return nil
}

func GetRemoteURL(repoPath string) (string, error) {
	expandedPath, err := ExpandPath(repoPath)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("git", "-C", expandedPath, "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func ParseRepoFullNameFromURL(remoteURL string) string {
	url := remoteURL
	url = strings.TrimSuffix(url, ".git")

	if strings.HasPrefix(url, "git@github.com:") {
		return strings.TrimPrefix(url, "git@github.com:")
	}

	if strings.HasPrefix(url, "https://github.com/") {
		return strings.TrimPrefix(url, "https://github.com/")
	}

	if strings.HasPrefix(url, "git@") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}

	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		parts := strings.Split(url, "/")
		if len(parts) >= 2 {
			return strings.Join(parts[len(parts)-2:], "/")
		}
	}

	return ""
}

func ListDirectoryContents(path string) ([]string, error) {
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(expandedPath)
	if err != nil {
		return nil, err
	}

	var results []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() {
			results = append(results, name+"/")
		}
	}

	return results, nil
}
