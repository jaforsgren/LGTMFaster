package azuredevops

import (
	"context"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func validationBuild(definitionID int, name string, status build.BuildStatus, result *build.BuildResult) build.Build {
	return build.Build{Status: &status, Result: result, Definition: &build.DefinitionReference{Id: &definitionID, Name: strPtr(name)}}
}

func TestGetChecks_UsesTheLatestBuildOfEachPipelineOnTheMergeRef(t *testing.T) {
	failed, succeeded := build.BuildResultValues.Failed, build.BuildResultValues.Succeeded
	buildClient := &mockBuildClient{builds: []build.Build{
		validationBuild(1, "ci", build.BuildStatusValues.Completed, &failed),
		validationBuild(1, "ci", build.BuildStatusValues.Completed, &succeeded),
		validationBuild(2, "lint", build.BuildStatusValues.Completed, &succeeded),
	}}
	provider := &Provider{
		client:    &Client{buildClient: buildClient},
		repoCache: map[string]*ResolvedRepository{testRepository: {ProjectID: "proj-id", RepoID: "repo-id", CachedAt: time.Now()}},
		cacheTTL:  time.Hour,
	}

	checks, err := provider.GetChecks(context.Background(), domain.PRIdentifier{Repository: testRepository, Number: 42})
	if err != nil {
		t.Fatal(err)
	}

	if *buildClient.lastGetBuildsArgs.BranchName != "refs/pull/42/merge" || *buildClient.lastGetBuildsArgs.RepositoryId != "repo-id" {
		t.Errorf("expected the PR's merge ref builds, got %+v", buildClient.lastGetBuildsArgs)
	}
	if checks.State != domain.CheckStateFailing || len(checks.Checks) != 2 || checks.Failing()[0] != "ci" {
		t.Errorf("expected ci failing from its latest build, got %+v", checks)
	}
}

func TestVotedBy_OnlyCountsTheUsersOwnVote(t *testing.T) {
	voted, notVoted := 10, 0
	pr := &git.GitPullRequest{Reviewers: &[]git.IdentityRefWithVote{
		{DisplayName: strPtr("Anna"), Vote: &voted},
		{DisplayName: strPtr("Johan"), Vote: &notVoted},
	}}

	if votedBy(pr, "Johan") {
		t.Error("expected a reviewer without a vote not to count as reviewed")
	}
	if !votedBy(pr, "Anna") {
		t.Error("expected a reviewer with a vote to count as reviewed")
	}
}
