package github

import (
	"slices"
	"testing"

	"github.com/google/go-github/v57/github"
)

func TestClosingIssueNumbers_FindsEachClosingKeywordOnce(t *testing.T) {
	body := "Fixes #12 and closes #7.\nResolved: #12, mentions #99, see owner/other#3"

	if got := closingIssueNumbers(body); !slices.Equal(got, []int{12, 7}) {
		t.Errorf("expected issues 12 and 7, got %v", got)
	}
}

func TestWithClosingReference(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"empty description", "", "Closes #4"},
		{"appended after a blank line", "Adds login\n", "Adds login\n\nCloses #4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := withClosingReference(tt.body, 4); got != tt.want {
				t.Errorf("withClosingReference(%q) = %q, want %q", tt.body, got, tt.want)
			}
			if !slices.Contains(closingIssueNumbers(withClosingReference(tt.body, 4)), 4) {
				t.Error("expected the added reference to be read back as a link")
			}
		})
	}
}

func TestIssueWorkItems_LeavesOutPullRequests(t *testing.T) {
	issues := []*github.Issue{
		{Number: github.Int(4), Title: github.String("Crash"), State: github.String("open"), HTMLURL: github.String("https://github.com/o/r/issues/4")},
		{Number: github.Int(5), PullRequestLinks: &github.PullRequestLinks{URL: github.String("https://api.github.com/pulls/5")}},
	}

	workItems := issueWorkItems(issues)

	if len(workItems) != 1 {
		t.Fatalf("expected only the issue, got %+v", workItems)
	}
	if item := workItems[0]; item.ID != 4 || item.Type != issueWorkItemType || item.Title != "Crash" || item.URL != "https://github.com/o/r/issues/4" {
		t.Errorf("expected the issue converted, got %+v", item)
	}
}
