package azuredevops

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func extractBranchName(refName *string) string {
	return strings.TrimPrefix(common.GetString(refName), "refs/heads/")
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

const (
	adoPathPrefix = "/"
	// firstColumn is the offset of a whole-line comment; Azure DevOps counts columns from 1.
	firstColumn = 1
)

// commentThreadContext anchors a thread to a line. Azure DevOps wants paths from the repository
// root with a leading slash, and a deleted line positioned in the left (old) file.
func commentThreadContext(filePath string, line int, side string) *git.CommentThreadContext {
	path := adoPathPrefix + strings.TrimPrefix(filePath, adoPathPrefix)
	start := &git.CommentPosition{Line: &line, Offset: intPtr(firstColumn)}
	end := &git.CommentPosition{Line: &line, Offset: intPtr(firstColumn)}
	if side == domain.CommentSideLeft {
		return &git.CommentThreadContext{FilePath: &path, LeftFileStart: start, LeftFileEnd: end}
	}
	return &git.CommentThreadContext{FilePath: &path, RightFileStart: start, RightFileEnd: end}
}

// commentPosition reads where a thread is anchored, as a path without the leading slash, the line
// and the side it counts in.
func commentPosition(context *git.CommentThreadContext) (string, int, string) {
	if context == nil || context.FilePath == nil {
		return "", 0, ""
	}
	path := strings.TrimPrefix(*context.FilePath, adoPathPrefix)
	if context.RightFileStart != nil && context.RightFileStart.Line != nil {
		return path, *context.RightFileStart.Line, domain.CommentSideRight
	}
	if context.LeftFileStart != nil && context.LeftFileStart.Line != nil {
		return path, *context.LeftFileStart.Line, domain.CommentSideLeft
	}
	return path, 0, ""
}

var voteMessage = regexp.MustCompile(`^(.+) voted (-?\d+)$`)

var voteDescriptions = map[string]string{
	"10":  "approved",
	"5":   "approved with suggestions",
	"0":   "reset their vote",
	"-5":  "is waiting for the author",
	"-10": "rejected",
}

// describeVote turns Azure DevOps' "<name> voted 10" system comments into "<name> approved".
func describeVote(content string) string {
	match := voteMessage.FindStringSubmatch(content)
	if match == nil {
		return content
	}
	description, known := voteDescriptions[match[2]]
	if !known {
		return content
	}
	return match[1] + " " + description
}
