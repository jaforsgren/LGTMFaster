package git

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestParseRemote(t *testing.T) {
	github := func(fullName string) domain.GitRemote {
		return domain.GitRemote{Provider: domain.ProviderGitHub, RepositoryFullName: fullName}
	}
	ado := func(organization, fullName string) domain.GitRemote {
		return domain.GitRemote{Provider: domain.ProviderAzureDevOps, Organization: organization, RepositoryFullName: fullName}
	}
	tests := map[string]domain.GitRemote{
		"git@github.com:johanforsgren/lgtmfaster.git":                      github("johanforsgren/lgtmfaster"),
		"https://github.com/johanforsgren/lgtmfaster":                      github("johanforsgren/lgtmfaster"),
		"https://user:token@github.com/johanforsgren/lgtmfaster.git":       github("johanforsgren/lgtmfaster"),
		"ssh://git@github.com/johanforsgren/lgtmfaster.git":                github("johanforsgren/lgtmfaster"),
		"https://contoso@dev.azure.com/contoso/My%20Project/_git/api":      ado("contoso", "My Project/api"),
		"https://dev.azure.com/contoso/Proj/_git/api":                      ado("contoso", "Proj/api"),
		"git@ssh.dev.azure.com:v3/contoso/My%20Project/api":                ado("contoso", "My Project/api"),
		"contoso@vs-ssh.visualstudio.com:v3/contoso/Proj/api":              ado("contoso", "Proj/api"),
		"https://contoso.visualstudio.com/Proj/_git/api":                   ado("contoso", "Proj/api"),
		"https://contoso.visualstudio.com/DefaultCollection/Proj/_git/api": ado("contoso", "Proj/api"),
	}
	for remoteURL, want := range tests {
		got, err := ParseRemote(remoteURL)
		if err != nil || got != want {
			t.Errorf("ParseRemote(%q) = %+v, %v; want %+v", remoteURL, got, err, want)
		}
	}
}

func TestParseRemote_RejectsOtherHostsWithoutPrintingCredentials(t *testing.T) {
	_, err := ParseRemote("https://user:secret@gitlab.com/group/repo.git")

	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Errorf("expected an error without the credentials, got %v", err)
	}
}

func TestIsBranchOnRemote(t *testing.T) {
	remote := setupTestGitRepo(t)
	if output, err := exec.Command("git", "-C", remote, "commit", "-q", "--allow-empty", "-m", "initial").CombinedOutput(); err != nil {
		t.Skipf("git commit failed, skipping test: %v %s", err, output)
	}
	clone := t.TempDir()
	if output, err := exec.Command("git", "clone", "-q", remote, clone).CombinedOutput(); err != nil {
		t.Fatalf("clone: %v %s", err, output)
	}
	branch, err := GetCurrentBranch(remote)
	if err != nil {
		t.Fatal(err)
	}

	if onRemote, err := IsBranchOnRemote(clone, branch); err != nil || !onRemote {
		t.Errorf("expected %s on the remote, got %v, %v", branch, onRemote, err)
	}
	if onRemote, err := IsBranchOnRemote(clone, "never-pushed"); err != nil || onRemote {
		t.Errorf("expected never-pushed missing, got %v, %v", onRemote, err)
	}
}
