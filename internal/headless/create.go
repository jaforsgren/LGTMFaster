package headless

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

var errSameBranch = errors.New("the current branch is the target branch; check out the branch to open a pull request from, or pass -t")

// ErrNotAllLinked means the pull request was created but some work items could not be linked.
var ErrNotAllLinked = errors.New("pull request created, but not every work item was linked")

// Create opens the drafted pull request with the PAT that reaches its repository, links the work
// items and reports the new pull request to out.
func Create(ctx context.Context, pats []domain.PAT, newProvider domain.ProviderFactory, draft domain.NewPullRequestDraft, out io.Writer) error {
	request := newPullRequest(draft)
	if request.SourceBranch == request.TargetBranch {
		return fmt.Errorf("%w (%s)", errSameBranch, request.TargetBranch)
	}
	pat, err := domain.MatchPAT(pats, draft.Remote)
	if err != nil {
		return err
	}
	provider, err := newProvider(pat)
	if err != nil {
		return err
	}

	pr, err := provider.CreatePullRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}
	failures := domain.LinkWorkItems(ctx, provider, pr.Identifier(), draft.WorkItemIDs)
	printCreated(out, *pr, request, draft, failures)
	if len(failures) > 0 {
		return fmt.Errorf("%w: %s", ErrNotAllLinked, domain.DescribeLinkFailures(failures))
	}
	return nil
}

const (
	createdMark      = "✔"
	failedMark       = "✘"
	linkMark         = "🔗"
	branchArrow      = "→"
	createdIndent    = "  "
	createdFormat    = createdMark + " Created pull request #%d in %s\n"
	titleFormat      = createdIndent + "%s\n"
	branchesFormat   = createdIndent + "%s " + branchArrow + " %s\n"
	urlFormat        = createdIndent + linkMark + " %s\n"
	linkedFormat     = createdIndent + createdMark + " Linked work item #%d\n"
	linkFailedFormat = createdIndent + failedMark + " Could not link work item #%d: %v\n"
)

// printCreated confirms the new pull request with its link and says how each work item fared.
func printCreated(out io.Writer, pr domain.PullRequest, request domain.NewPullRequest, draft domain.NewPullRequestDraft, failures []domain.LinkFailure) {
	prURL := pr.URL
	if prURL == "" {
		prURL = draft.Remote.PullRequestURL(pr.Number)
	}
	fmt.Fprintf(out, createdFormat, pr.Number, request.RepositoryFullName)
	fmt.Fprintf(out, titleFormat, request.Title)
	fmt.Fprintf(out, branchesFormat, request.SourceBranch, request.TargetBranch)
	fmt.Fprintf(out, urlFormat, prURL)
	for _, workItemID := range draft.WorkItemIDs {
		failure := slices.IndexFunc(failures, func(failure domain.LinkFailure) bool { return failure.WorkItemID == workItemID })
		if failure < 0 {
			fmt.Fprintf(out, linkedFormat, workItemID)
			continue
		}
		fmt.Fprintf(out, linkFailedFormat, workItemID, failures[failure].Err)
	}
}

func newPullRequest(draft domain.NewPullRequestDraft) domain.NewPullRequest {
	target := draft.TargetBranch
	if target == "" {
		target = DefaultTargetBranch
	}
	title := domain.TitleFromBranch(draft.SourceBranch)
	if title == "" {
		title = draft.SourceBranch
	}
	return domain.NewPullRequest{
		RepositoryFullName: draft.Remote.RepositoryFullName,
		SourceBranch:       draft.SourceBranch,
		TargetBranch:       target,
		Title:              title,
		Description:        draft.Description,
	}
}

// SelectedPAT is the PAT the TUI opens the drafted pull request with: it only works with the
// selected ones.
func SelectedPAT(pats []domain.PAT, draft domain.NewPullRequestDraft) (domain.PAT, error) {
	pat, err := domain.MatchPAT(domain.SelectedPATs(pats), draft.Remote)
	if err != nil {
		return domain.PAT{}, fmt.Errorf("%w among the selected PATs; select one with :pats first", err)
	}
	return pat, nil
}
