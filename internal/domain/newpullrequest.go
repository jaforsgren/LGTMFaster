package domain

import "time"

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
