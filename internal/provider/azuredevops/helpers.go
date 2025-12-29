package azuredevops

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func getString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func getBool(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

func extractBranchName(refName *string) string {
	if refName == nil {
		return ""
	}
	return strings.TrimPrefix(*refName, "refs/heads/")
}

func parseRepositoryIdentifier(repository string) (project, repo string, err error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: expected 'project/repo', got '%s'", repository)
	}
	return parts[0], parts[1], nil
}

func buildRepositoryIdentifier(projectName, repoName string) string {
	return fmt.Sprintf("%s/%s", projectName, repoName)
}

func buildPRWebURL(pr *git.GitPullRequest) string {
	if pr.Repository == nil || pr.Repository.WebUrl == nil {
		return ""
	}
	return fmt.Sprintf("%s/pullrequest/%d", *pr.Repository.WebUrl, *pr.PullRequestId)
}

func isMergeable(mergeStatus *git.PullRequestAsyncStatus) bool {
	if mergeStatus == nil {
		return false
	}
	return *mergeStatus == git.PullRequestAsyncStatusValues.Succeeded ||
		*mergeStatus == git.PullRequestAsyncStatusValues.Queued
}

func getUUIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
