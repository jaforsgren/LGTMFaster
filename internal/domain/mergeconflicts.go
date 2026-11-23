package domain

// MergeConflictStatus says whether a pull request conflicts with its target branch. Providers
// compute it in the background, so it can be not known yet.
type MergeConflictStatus string

const (
	MergeConflictUnknown  MergeConflictStatus = ""
	MergeConflictChecking MergeConflictStatus = "checking"
	MergeConflictNone     MergeConflictStatus = "none"
	MergeConflictFound    MergeConflictStatus = "conflicts"
)
