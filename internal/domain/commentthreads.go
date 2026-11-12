package domain

import "fmt"

// ThreadState is whether a comment thread still needs attention.
type ThreadState string

const (
	ThreadStateActive   ThreadState = "active"
	ThreadStateResolved ThreadState = "resolved"
)

// CommentThreads counts a pull request's discussion threads, leaving out the provider's own (votes).
type CommentThreads struct {
	Total    int
	Resolved int
	// ResolutionKnown is false when the provider does not say which threads are resolved.
	ResolutionKnown bool
}

func SummarizeThreads(comments []Comment) CommentThreads {
	states := make(map[string]ThreadState)
	for _, comment := range comments {
		if comment.IsSystem {
			continue
		}
		threadID := comment.ThreadID
		if threadID == "" {
			threadID = comment.ID
		}
		states[threadID] = comment.ThreadState
	}

	summary := CommentThreads{Total: len(states)}
	for _, state := range states {
		if state != "" {
			summary.ResolutionKnown = true
		}
		if state == ThreadStateResolved {
			summary.Resolved++
		}
	}
	return summary
}

// Describe reads "1/3 resolved", or just the count when resolution is unknown.
func (c CommentThreads) Describe() string {
	if !c.ResolutionKnown {
		return fmt.Sprintf("%d", c.Total)
	}
	return fmt.Sprintf("%d/%d resolved", c.Resolved, c.Total)
}
