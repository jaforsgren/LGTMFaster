package domain

import (
	"sort"
	"time"
)

type ActivityKind string

const (
	ActivityKindOpened  ActivityKind = "opened"
	ActivityKindPush    ActivityKind = "push"
	ActivityKindComment ActivityKind = "comment"
	// ActivityKindReview is a vote (Azure DevOps) or a submitted review (GitHub).
	ActivityKindReview ActivityKind = "review"

	openedSummary = "opened the pull request"
)

// Activity is one thing that happened on a pull request.
type Activity struct {
	Kind   ActivityKind
	Author User
	At     time.Time
	// Summary says what happened without naming the author: "approved", "pushed",
	// "commented: <first line>".
	Summary string
	ByMe    bool
}

// OpenedActivity is the pull request's opening, which the providers' activity leaves out: GitHub's
// timeline has no event for it.
func (pr PullRequest) OpenedActivity() Activity {
	return Activity{
		Kind:    ActivityKindOpened,
		Author:  pr.Author,
		At:      pr.CreatedAt,
		Summary: openedSummary,
		ByMe:    pr.Category == PRCategoryAuthored,
	}
}

// ActivitySince returns what others did after since, newest first.
func ActivitySince(activity []Activity, since time.Time) []Activity {
	var news []Activity
	for _, item := range activity {
		if !item.ByMe && item.At.After(since) {
			news = append(news, item)
		}
	}
	sort.SliceStable(news, func(i, j int) bool { return news[i].At.After(news[j].At) })
	return news
}
