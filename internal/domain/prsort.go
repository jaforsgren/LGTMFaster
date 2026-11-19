package domain

import "slices"

type PRSortMode string

const (
	PRSortDefault    PRSortMode = "default"
	PRSortNewest     PRSortMode = "newest"
	PRSortTitle      PRSortMode = "title"
	PRSortWorkItem   PRSortMode = "work_item"
	PRSortRepository PRSortMode = "repository"
	PRSortAuthor     PRSortMode = "author"
)

// PRSortModes is ordered by the number key that selects each mode.
var PRSortModes = []PRSortMode{
	PRSortDefault,
	PRSortNewest,
	PRSortTitle,
	PRSortWorkItem,
	PRSortRepository,
	PRSortAuthor,
}

// OrDefault maps unset or unknown modes (e.g. hand-edited settings) to the default mode.
func (m PRSortMode) OrDefault() PRSortMode {
	if slices.Contains(PRSortModes, m) {
		return m
	}
	return PRSortDefault
}
