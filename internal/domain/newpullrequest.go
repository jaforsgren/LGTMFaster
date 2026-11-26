package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// PushedBranch is a branch the user pushed recently, which a new pull request can be opened from.
type PushedBranch struct {
	Repository Repo
	Branch     string
	PushedAt   time.Time
	PATID      string
}

// Key identifies the branch in suggestions and commands, as "<repository full name>:<branch>".
func (b PushedBranch) Key() string {
	return b.Repository.FullName + PushedBranchSeparator + b.Branch
}

const PushedBranchSeparator = ":"

// RepositoryBranches are the branches a new pull request can target.
type RepositoryBranches struct {
	Default string
	Names   []string
}

type NewPullRequest struct {
	RepositoryFullName string
	SourceBranch       string
	TargetBranch       string
	Title              string
	Description        string
	IsDraft            bool
}

const (
	branchPathSeparator  = "/"
	branchWordSeparators = "-_"
)

// TitleFromBranch turns the last segment of a branch name into a sentence: "feature/add-login"
// becomes "Add login".
func TitleFromBranch(branch string) string {
	segments := strings.Split(branch, branchPathSeparator)
	words := strings.FieldsFunc(segments[len(segments)-1], func(r rune) bool {
		return strings.ContainsRune(branchWordSeparators, r)
	})
	title := strings.Join(words, " ")
	if title == "" {
		return ""
	}
	runes := []rune(title)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// NewPullRequestDraft is a pull request to open from a local repository's branch, as given on the
// command line. Empty fields are left for the form, or the defaults, to fill in.
type NewPullRequestDraft struct {
	Remote       GitRemote
	SourceBranch string
	TargetBranch string
	Description  string
	WorkItemIDs  []int
}

// LinkFailure is a work item that could not be linked to a new pull request.
type LinkFailure struct {
	WorkItemID int
	Err        error
}

// LinkWorkItems links every work item it can, so one that does not exist leaves the others linked.
func LinkWorkItems(ctx context.Context, provider Provider, identifier PRIdentifier, workItemIDs []int) []LinkFailure {
	var failures []LinkFailure
	for _, workItemID := range workItemIDs {
		if err := provider.LinkWorkItem(ctx, identifier, workItemID); err != nil {
			failures = append(failures, LinkFailure{WorkItemID: workItemID, Err: err})
		}
	}
	return failures
}

// DescribeLinkFailures says which work items could not be linked and why.
func DescribeLinkFailures(failures []LinkFailure) string {
	described := make([]string, 0, len(failures))
	for _, failure := range failures {
		described = append(described, fmt.Sprintf("could not link #%d: %v", failure.WorkItemID, failure.Err))
	}
	return strings.Join(described, "; ")
}
