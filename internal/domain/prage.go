package domain

import "time"

const hoursPerDay = 24

// WithoutOldOtherPRs drops the pull requests neither authored by nor assigned to the user that
// were opened more than maxAgeDays before now.
func WithoutOldOtherPRs(prs []PullRequest, maxAgeDays int, now time.Time) []PullRequest {
	cutoff := now.Add(-time.Duration(maxAgeDays) * hoursPerDay * time.Hour)
	var out []PullRequest
	for _, pr := range prs {
		if pr.Category == PRCategoryOther && pr.CreatedAt.Before(cutoff) {
			continue
		}
		out = append(out, pr)
	}
	return out
}
