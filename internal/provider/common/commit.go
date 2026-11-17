package common

import "strings"

// CommitSummary is the first line of a commit message.
func CommitSummary(message string) string {
	summary, _, _ := strings.Cut(strings.TrimSpace(message), "\n")
	return strings.TrimSpace(summary)
}
