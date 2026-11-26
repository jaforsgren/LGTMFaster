package headless

import (
	"fmt"
	"os"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/git"
)

// detachedHead is what git names the branch when none is checked out.
const detachedHead = "HEAD"

// Editor lets the user write text, starting from initial.
type Editor func(initial string) (string, error)

// Draft is the pull request the flags describe, from the branch checked out in repoPath, which
// must be on origin already.
func Draft(repoPath string, f *Flags, edit Editor) (domain.NewPullRequestDraft, error) {
	workItemIDs, err := domain.ParseWorkItemIDs(f.WorkItems)
	if err != nil {
		return domain.NewPullRequestDraft{}, fmt.Errorf("-%s: %w", flagWorkItems, err)
	}

	branch, err := git.GetCurrentBranch(repoPath)
	if err != nil {
		return domain.NewPullRequestDraft{}, fmt.Errorf("not in a git repository: %w", err)
	}
	if branch == detachedHead {
		return domain.NewPullRequestDraft{}, fmt.Errorf("no branch is checked out")
	}
	remote, err := git.OriginRemote(repoPath)
	if err != nil {
		return domain.NewPullRequestDraft{}, err
	}
	onRemote, err := git.IsBranchOnRemote(repoPath, branch)
	if err != nil {
		return domain.NewPullRequestDraft{}, err
	}
	if !onRemote {
		return domain.NewPullRequestDraft{}, fmt.Errorf("branch %s is not on origin yet; push it first: git push -u origin %s", branch, branch)
	}

	description, err := f.description(edit)
	if err != nil {
		return domain.NewPullRequestDraft{}, err
	}
	return domain.NewPullRequestDraft{
		Remote:       remote,
		SourceBranch: branch,
		TargetBranch: f.Target,
		Description:  description,
		WorkItemIDs:  workItemIDs,
	}, nil
}

func (f *Flags) description(edit Editor) (string, error) {
	switch {
	case f.DescriptionFile != "":
		path, err := git.ExpandPath(f.DescriptionFile)
		if err != nil {
			return "", err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read the description file: %w", err)
		}
		return string(content), nil
	case f.DescriptionEditor:
		return edit("")
	}
	return f.Description, nil
}
