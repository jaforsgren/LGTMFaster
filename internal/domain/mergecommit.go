package domain

import "time"

// MergeCommit is the commit a merged pull request left on its target branch: the squash commit when
// squashed, the merge commit, or the rebased tip. The pipeline runs that built the merge are found by it.
type MergeCommit struct {
	Repository   string
	TargetBranch string
	CommitID     string
	MergedAt     time.Time
}

// MergeCommit returns what the pull request merged as; false unless it was merged.
func (pr PullRequest) MergeCommit() (MergeCommit, bool) {
	if pr.Status != PRStatusMerged || pr.MergeCommitID == "" {
		return MergeCommit{}, false
	}
	return MergeCommit{
		Repository:   pr.Repository.FullName,
		TargetBranch: pr.TargetBranch,
		CommitID:     pr.MergeCommitID,
		MergedAt:     pr.ClosedAt,
	}, true
}
