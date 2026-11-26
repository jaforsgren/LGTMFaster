package domain

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	githubPullRequestURLFormat      = "https://github.com/%s/pull/%d"
	azureDevOpsPullRequestURLFormat = "https://dev.azure.com/%s/%s/_git/%s/pullrequest/%d"
	repositoryFullNameSeparator     = "/"
)

// GitRemote is the repository a local clone's remote points at.
type GitRemote struct {
	Provider ProviderType
	// Organization is the Azure DevOps organization; empty for GitHub.
	Organization string
	// RepositoryFullName is "owner/repo" on GitHub and "project/repo" on Azure DevOps, as in
	// PR identifiers.
	RepositoryFullName string
}

// PullRequestURL is the web address of the remote's pull request with the given number.
func (r GitRemote) PullRequestURL(number int) string {
	if r.Provider == ProviderGitHub {
		return fmt.Sprintf(githubPullRequestURLFormat, r.RepositoryFullName, number)
	}
	project, repository, _ := strings.Cut(r.RepositoryFullName, repositoryFullNameSeparator)
	return fmt.Sprintf(azureDevOpsPullRequestURLFormat,
		url.PathEscape(r.Organization), url.PathEscape(project), url.PathEscape(repository), number)
}

// MatchPAT picks the PAT that can reach the remote: of its provider and, on Azure DevOps, its
// organization; the primary one first, then a selected one.
func MatchPAT(pats []PAT, remote GitRemote) (PAT, error) {
	var match *PAT
	for i, pat := range pats {
		if !pat.canReach(remote) {
			continue
		}
		if match == nil || pat.rank() > match.rank() {
			match = &pats[i]
		}
	}
	if match == nil {
		if remote.Provider == ProviderAzureDevOps {
			return PAT{}, fmt.Errorf("no Azure DevOps PAT for organization %s", remote.Organization)
		}
		return PAT{}, fmt.Errorf("no %s PAT", remote.Provider)
	}
	return *match, nil
}

// SelectedPATs are the PATs the TUI works with.
func SelectedPATs(pats []PAT) []PAT {
	var selected []PAT
	for _, pat := range pats {
		if pat.IsSelected {
			selected = append(selected, pat)
		}
	}
	return selected
}

func (p PAT) canReach(remote GitRemote) bool {
	if p.Provider != remote.Provider {
		return false
	}
	return remote.Provider != ProviderAzureDevOps || strings.EqualFold(p.Organization, remote.Organization)
}

// PAT ranks for MatchPAT: a higher one is preferred.
const (
	patRankOther = iota
	patRankSelected
	patRankPrimary
)

func (p PAT) rank() int {
	switch {
	case p.IsPrimary:
		return patRankPrimary
	case p.IsSelected:
		return patRankSelected
	}
	return patRankOther
}
