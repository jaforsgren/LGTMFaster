package domain

import (
	"strings"
	"testing"
)

func TestMatchPAT_PicksTheAzureDevOpsPATOfTheOrganization(t *testing.T) {
	pats := []PAT{
		{ID: "gh", Provider: ProviderGitHub, IsPrimary: true},
		{ID: "other", Provider: ProviderAzureDevOps, Organization: "other", IsPrimary: true},
		{ID: "contoso", Provider: ProviderAzureDevOps, Organization: "Contoso"},
	}

	pat, err := MatchPAT(pats, GitRemote{Provider: ProviderAzureDevOps, Organization: "contoso"})

	if err != nil || pat.ID != "contoso" {
		t.Errorf("expected the contoso PAT, got %q, %v", pat.ID, err)
	}
}

func TestMatchPAT_PrefersThePrimaryThenASelectedPAT(t *testing.T) {
	remote := GitRemote{Provider: ProviderGitHub}
	pats := []PAT{
		{ID: "plain", Provider: ProviderGitHub},
		{ID: "selected", Provider: ProviderGitHub, IsSelected: true},
	}
	if pat, _ := MatchPAT(pats, remote); pat.ID != "selected" {
		t.Errorf("expected the selected PAT, got %q", pat.ID)
	}

	pats = append(pats, PAT{ID: "primary", Provider: ProviderGitHub, IsSelected: true, IsPrimary: true})
	if pat, _ := MatchPAT(pats, remote); pat.ID != "primary" {
		t.Errorf("expected the primary PAT, got %q", pat.ID)
	}
}

func TestMatchPAT_SaysWhichPATIsMissing(t *testing.T) {
	_, err := MatchPAT([]PAT{{Provider: ProviderGitHub}}, GitRemote{Provider: ProviderAzureDevOps, Organization: "contoso"})

	if err == nil || !strings.Contains(err.Error(), "organization contoso") {
		t.Errorf("expected the missing organization named, got %v", err)
	}
}

func TestGitRemote_PullRequestURL(t *testing.T) {
	tests := map[string]GitRemote{
		"https://github.com/johanforsgren/lgtmfaster/pull/7":                {Provider: ProviderGitHub, RepositoryFullName: "johanforsgren/lgtmfaster"},
		"https://dev.azure.com/contoso/My%20Project/_git/api/pullrequest/7": {Provider: ProviderAzureDevOps, Organization: "contoso", RepositoryFullName: "My Project/api"},
	}
	for want, remote := range tests {
		if got := remote.PullRequestURL(7); got != want {
			t.Errorf("expected %s, got %s", want, got)
		}
	}
}
