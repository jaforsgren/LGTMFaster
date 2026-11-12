package github

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/go-github/v57/github"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
)

const (
	issueWorkItemType = "Issue"
	// closingReferenceFormat is the keyword GitHub links an issue to a pull request by.
	closingReferenceFormat = "Closes #%d"
	closingReferenceSpacer = "\n\n"
)

// closingReference matches GitHub's closing keywords followed by an issue in the same repository.
var closingReference = regexp.MustCompile(`(?i)\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?):?\s+#(\d+)\b`)

// ListLinkedWorkItems returns the issues the pull request's description closes; GitHub's REST API
// has no other link between pull requests and issues.
func (p *Provider) ListLinkedWorkItems(ctx context.Context, identifier domain.PRIdentifier) ([]domain.WorkItem, error) {
	owner, repo, err := common.ParseGitHubRepository(identifier.Repository)
	if err != nil {
		return nil, err
	}
	pr, err := p.client.GetPullRequest(ctx, owner, repo, identifier.Number)
	if err != nil {
		return nil, err
	}
	numbers := closingIssueNumbers(pr.GetBody())
	workItems := make([]domain.WorkItem, 0, len(numbers))
	for _, number := range numbers {
		issue, err := p.client.GetIssue(ctx, owner, repo, number)
		if err != nil {
			logger.LogError("GITHUB_GET_ISSUE", fmt.Sprintf("%s#%d", identifier.Repository, number), err)
			return nil, err
		}
		workItems = append(workItems, toDomainWorkItem(issue))
	}
	return workItems, nil
}

func (p *Provider) ListAssignedWorkItems(ctx context.Context, identifier domain.PRIdentifier) ([]domain.WorkItem, error) {
	owner, repo, err := common.ParseGitHubRepository(identifier.Repository)
	if err != nil {
		return nil, err
	}
	issues, err := p.client.ListAssignedOpenIssues(ctx, owner, repo)
	if err != nil {
		logger.LogError("GITHUB_LIST_ASSIGNED_ISSUES", identifier.Repository, err)
		return nil, err
	}
	return issueWorkItems(issues), nil
}

// LinkWorkItem adds a closing keyword for the issue to the pull request's description.
func (p *Provider) LinkWorkItem(ctx context.Context, identifier domain.PRIdentifier, workItemID int) error {
	owner, repo, err := common.ParseGitHubRepository(identifier.Repository)
	if err != nil {
		return err
	}
	pr, err := p.client.GetPullRequest(ctx, owner, repo, identifier.Number)
	if err != nil {
		return err
	}
	if slices.Contains(closingIssueNumbers(pr.GetBody()), workItemID) {
		return nil
	}
	update := &github.PullRequest{Body: github.String(withClosingReference(pr.GetBody(), workItemID))}
	if _, err := p.client.UpdatePullRequest(ctx, owner, repo, identifier.Number, update); err != nil {
		logger.LogError("GITHUB_LINK_ISSUE", fmt.Sprintf("%s#%d", identifier.Repository, identifier.Number), err)
		return fmt.Errorf("%s", common.ExtractErrorMessage(err))
	}
	logger.Log("GitHub: Linked issue #%d to PR #%d", workItemID, identifier.Number)
	return nil
}

// closingIssueNumbers returns each issue the text closes once, in order of mention.
func closingIssueNumbers(text string) []int {
	var numbers []int
	for _, match := range closingReference.FindAllStringSubmatch(text, -1) {
		number, err := strconv.Atoi(match[1])
		if err == nil && !slices.Contains(numbers, number) {
			numbers = append(numbers, number)
		}
	}
	return numbers
}

func withClosingReference(body string, number int) string {
	reference := fmt.Sprintf(closingReferenceFormat, number)
	if strings.TrimSpace(body) == "" {
		return reference
	}
	return strings.TrimRight(body, "\n") + closingReferenceSpacer + reference
}

func issueWorkItems(issues []*github.Issue) []domain.WorkItem {
	workItems := make([]domain.WorkItem, 0, len(issues))
	for _, issue := range issues {
		if !issue.IsPullRequest() {
			workItems = append(workItems, toDomainWorkItem(issue))
		}
	}
	return workItems
}

func toDomainWorkItem(issue *github.Issue) domain.WorkItem {
	return domain.WorkItem{
		ID:          issue.GetNumber(),
		Type:        issueWorkItemType,
		Title:       issue.GetTitle(),
		State:       issue.GetState(),
		Description: issue.GetBody(),
		URL:         issue.GetHTMLURL(),
	}
}
