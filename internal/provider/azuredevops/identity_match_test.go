package azuredevops

import (
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
)

const (
	testDisplayName = "Johan Forsgren"
	testUniqueName  = "johan@weapp.se"
)

func TestIdentityMatchesUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{"full unique name", testUniqueName, true},
		{"full unique name different case", "Johan@WeApp.se", true},
		{"unique name local part", "johan", true},
		{"display name", testDisplayName, true},
		{"display name different case", "johan forsgren", true},
		{"other user", "someone@weapp.se", false},
		{"partial local part", "joh", false},
		{"empty username", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := identityMatchesUsername(strPtr(testDisplayName), strPtr(testUniqueName), tt.username)
			if got != tt.want {
				t.Errorf("identityMatchesUsername(%q) = %v, want %v", tt.username, got, tt.want)
			}
		})
	}
}

func TestIdentityMatchesUsername_EmptyUsernameNeverMatchesEmptyIdentity(t *testing.T) {
	if identityMatchesUsername(strPtr(""), strPtr(""), "") {
		t.Error("Expected empty username not to match an empty identity")
	}
}

func TestDeterminePRCategory_AuthoredWithEmailUsername(t *testing.T) {
	pr := &git.GitPullRequest{
		CreatedBy: &webapi.IdentityRef{DisplayName: strPtr(testDisplayName), UniqueName: strPtr(testUniqueName)},
	}

	if got := determinePRCategory(pr, testUniqueName); got != domain.PRCategoryAuthored {
		t.Errorf("determinePRCategory() = %q, want %q", got, domain.PRCategoryAuthored)
	}
}

func TestDeterminePRCategory_AssignedWithEmailUsername(t *testing.T) {
	pr := &git.GitPullRequest{
		CreatedBy: &webapi.IdentityRef{DisplayName: strPtr("Someone Else"), UniqueName: strPtr("someone@weapp.se")},
		Reviewers: &[]git.IdentityRefWithVote{
			{DisplayName: strPtr(testDisplayName), UniqueName: strPtr(testUniqueName)},
		},
	}

	if got := determinePRCategory(pr, testUniqueName); got != domain.PRCategoryAssigned {
		t.Errorf("determinePRCategory() = %q, want %q", got, domain.PRCategoryAssigned)
	}
}

func TestVotedBy_WithEmailUsername(t *testing.T) {
	approved := 10
	pr := &git.GitPullRequest{
		Reviewers: &[]git.IdentityRefWithVote{
			{DisplayName: strPtr(testDisplayName), UniqueName: strPtr(testUniqueName), Vote: &approved},
		},
	}

	if !votedBy(pr, testUniqueName) {
		t.Error("Expected votedBy to match reviewer by full unique name")
	}
}
