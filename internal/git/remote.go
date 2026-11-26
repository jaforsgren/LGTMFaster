package git

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	remoteName           = "origin"
	urlSchemeSeparator   = "://"
	scpHostSeparator     = ":"
	scpUserSeparator     = "@"
	gitSuffix            = ".git"
	pathSeparator        = "/"
	githubHost           = "github.com"
	azureDevOpsHost      = "dev.azure.com"
	azureDevOpsSSHHost   = "ssh.dev.azure.com"
	visualStudioSSHHost  = "vs-ssh.visualstudio.com"
	visualStudioSuffix   = ".visualstudio.com"
	azureDevOpsGitMarker = "_git"
	azureDevOpsSSHPrefix = "v3"
	// defaultCollection is the collection old visualstudio.com URLs name before the project.
	defaultCollection = "DefaultCollection"
)

// ParseRemote tells which GitHub or Azure DevOps repository a remote URL points at, in any of
// the HTTPS and SSH forms the two hand out.
func ParseRemote(remoteURL string) (domain.GitRemote, error) {
	host, path, err := splitRemoteURL(remoteURL)
	if err != nil {
		return domain.GitRemote{}, err
	}
	segments := strings.Split(strings.TrimSuffix(strings.Trim(path, pathSeparator), gitSuffix), pathSeparator)

	switch {
	case host == githubHost && len(segments) == 2:
		return domain.GitRemote{Provider: domain.ProviderGitHub, RepositoryFullName: strings.Join(segments, pathSeparator)}, nil
	case host == azureDevOpsHost && len(segments) == 4 && segments[2] == azureDevOpsGitMarker:
		return azureDevOpsRemote(segments[0], segments[1], segments[3]), nil
	case (host == azureDevOpsSSHHost || host == visualStudioSSHHost) && len(segments) == 4 && segments[0] == azureDevOpsSSHPrefix:
		return azureDevOpsRemote(segments[1], segments[2], segments[3]), nil
	case strings.HasSuffix(host, visualStudioSuffix):
		if len(segments) == 4 && segments[0] == defaultCollection {
			segments = segments[1:]
		}
		if len(segments) == 3 && segments[1] == azureDevOpsGitMarker {
			return azureDevOpsRemote(strings.TrimSuffix(host, visualStudioSuffix), segments[0], segments[2]), nil
		}
	}
	return domain.GitRemote{}, fmt.Errorf("%s is not a GitHub or Azure DevOps repository", redactURL(remoteURL))
}

func azureDevOpsRemote(organization, project, repository string) domain.GitRemote {
	return domain.GitRemote{
		Provider:           domain.ProviderAzureDevOps,
		Organization:       organization,
		RepositoryFullName: project + pathSeparator + repository,
	}
}

// splitRemoteURL returns the host and the unescaped path of a URL ("https://host/path",
// "ssh://git@host/path") or of scp-like syntax ("git@host:path").
func splitRemoteURL(remoteURL string) (string, string, error) {
	if strings.Contains(remoteURL, urlSchemeSeparator) {
		parsed, err := url.Parse(remoteURL)
		if err != nil {
			return "", "", fmt.Errorf("invalid remote URL %s", redactURL(remoteURL))
		}
		return strings.ToLower(parsed.Hostname()), parsed.Path, nil
	}
	hostPart, path, ok := strings.Cut(remoteURL, scpHostSeparator)
	if !ok {
		return "", "", fmt.Errorf("invalid remote URL %s", remoteURL)
	}
	if _, host, hasUser := strings.Cut(hostPart, scpUserSeparator); hasUser {
		hostPart = host
	}
	unescaped, err := url.PathUnescape(path)
	if err != nil {
		return "", "", fmt.Errorf("invalid remote URL %s", remoteURL)
	}
	return strings.ToLower(hostPart), unescaped, nil
}

// redactURL drops credentials embedded in a remote URL, so they are not printed in errors.
func redactURL(remoteURL string) string {
	parsed, err := url.Parse(remoteURL)
	if err != nil || parsed.User == nil {
		return remoteURL
	}
	parsed.User = nil
	return parsed.String()
}

// OriginRemote is the origin remote's repository, which pull requests from the repository at
// repoPath are opened in.
func OriginRemote(repoPath string) (domain.GitRemote, error) {
	remoteURL, err := GetRemoteURL(repoPath)
	if err != nil {
		return domain.GitRemote{}, err
	}
	return ParseRemote(remoteURL)
}

// IsBranchOnRemote reports whether origin has the branch, which a pull request needs.
func IsBranchOnRemote(repoPath, branch string) (bool, error) {
	cmd := exec.Command("git", "-C", repoPath, "ls-remote", "--exit-code", "--heads", remoteName, branch)
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == lsRemoteNoMatchExitCode {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to read the branches of %s: %w", remoteName, err)
	}
	return true, nil
}

// lsRemoteNoMatchExitCode is what git ls-remote --exit-code exits with when no ref matches.
const lsRemoteNoMatchExitCode = 2
