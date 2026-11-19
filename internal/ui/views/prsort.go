package views

import (
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

var prSortModeLabels = map[domain.PRSortMode]string{
	domain.PRSortDefault:    "default",
	domain.PRSortNewest:     "newest",
	domain.PRSortTitle:      "title",
	domain.PRSortWorkItem:   "linked work item",
	domain.PRSortRepository: "repository",
	domain.PRSortAuthor:     "author",
}

func PRSortModeLabel(mode domain.PRSortMode) string {
	return prSortModeLabels[mode.OrDefault()]
}

// prSorter orders a list of pull requests. dateOf is the date the list shows (created for open
// pull requests, closed for closed ones), which the newest mode and every tie-break go by.
type prSorter struct {
	mode        domain.PRSortMode
	dateOf      func(domain.PullRequest) time.Time
	workItemIDs map[string][]int
}

func (s prSorter) sort(prs []domain.PullRequest) []domain.PullRequest {
	out := append([]domain.PullRequest(nil), prs...)
	sort.SliceStable(out, func(i, j int) bool {
		if order := s.compare(out[i], out[j]); order != 0 {
			return order < 0
		}
		return s.dateOf(out[i]).After(s.dateOf(out[j]))
	})
	return out
}

func (s prSorter) compare(a, b domain.PullRequest) int {
	switch s.mode {
	case domain.PRSortTitle:
		return strings.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title))
	case domain.PRSortWorkItem:
		return compareWorkItemIDs(s.workItemIDs[a.Key()], s.workItemIDs[b.Key()])
	case domain.PRSortRepository:
		return strings.Compare(strings.ToLower(a.Repository.FullName), strings.ToLower(b.Repository.FullName))
	case domain.PRSortAuthor:
		return strings.Compare(strings.ToLower(a.Author.Username), strings.ToLower(b.Author.Username))
	default:
		return 0
	}
}

// compareWorkItemIDs orders by the lowest linked work item; pull requests without one go last.
func compareWorkItemIDs(a, b []int) int {
	switch {
	case len(a) == 0 && len(b) == 0:
		return 0
	case len(a) == 0:
		return 1
	case len(b) == 0:
		return -1
	default:
		return slices.Min(a) - slices.Min(b)
	}
}
