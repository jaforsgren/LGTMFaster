package github

import (
	"context"

	"github.com/google/go-github/v57/github"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
)

const (
	checkRunCompleted  = "completed"
	reviewStatePending = "PENDING"
)

var statusCheckStates = map[string]domain.CheckState{
	"success": domain.CheckStatePassing,
	"pending": domain.CheckStatePending,
	"failure": domain.CheckStateFailing,
	"error":   domain.CheckStateFailing,
}

var checkRunConclusionStates = map[string]domain.CheckState{
	"success":         domain.CheckStatePassing,
	"neutral":         domain.CheckStatePassing,
	"skipped":         domain.CheckStatePassing,
	"failure":         domain.CheckStateFailing,
	"timed_out":       domain.CheckStateFailing,
	"cancelled":       domain.CheckStateFailing,
	"action_required": domain.CheckStateFailing,
}

// GetChecks combines the commit statuses and check runs of the pull request's head commit.
func (p *Provider) GetChecks(ctx context.Context, identifier domain.PRIdentifier) (domain.Checks, error) {
	owner, repo, err := common.ParseGitHubRepository(identifier.Repository)
	if err != nil {
		return domain.Checks{}, err
	}
	pr, err := p.client.GetPullRequest(ctx, owner, repo, identifier.Number)
	if err != nil {
		return domain.Checks{}, err
	}
	sha := pr.GetHead().GetSHA()

	status, err := p.client.GetCombinedStatus(ctx, owner, repo, sha)
	if err != nil {
		return domain.Checks{}, err
	}
	runs, err := p.client.ListCheckRuns(ctx, owner, repo, sha)
	if err != nil {
		return domain.Checks{}, err
	}
	return domain.SummarizeChecks(append(statusChecks(status), checkRunChecks(runs)...)), nil
}

func statusChecks(status *github.CombinedStatus) []domain.Check {
	var checks []domain.Check
	for _, repoStatus := range status.Statuses {
		if state, known := statusCheckStates[repoStatus.GetState()]; known {
			checks = append(checks, domain.Check{Name: repoStatus.GetContext(), State: state})
		}
	}
	return checks
}

func checkRunChecks(runs []*github.CheckRun) []domain.Check {
	checks := make([]domain.Check, 0, len(runs))
	for _, run := range runs {
		state := domain.CheckStatePending
		if run.GetStatus() == checkRunCompleted {
			conclusionState, known := checkRunConclusionStates[run.GetConclusion()]
			if !known {
				continue
			}
			state = conclusionState
		}
		checks = append(checks, domain.Check{Name: run.GetName(), State: state})
	}
	return checks
}

// reviewedBy reports whether the user submitted a review, of any kind.
func reviewedBy(reviews []*github.PullRequestReview, username string) bool {
	for _, review := range reviews {
		if review.GetUser().GetLogin() == username && review.GetState() != reviewStatePending {
			return true
		}
	}
	return false
}
