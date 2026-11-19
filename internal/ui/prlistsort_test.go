package ui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func sortTestModel(t *testing.T) (Model, *mockRepository) {
	t.Helper()
	provider := &mockProvider{linkedWorkItemIDs: map[int][]int{1: {900}, 2: {100, 500}}}
	m := newPRTestModel(t, provider)
	repository := &mockRepository{}
	m.repository = repository
	now := time.Now()
	m.prListView.SetPRs([]domain.PullRequest{
		{Number: 1, Title: "Older", CreatedAt: now.Add(-2 * time.Hour), PATID: testPATID, Repository: domain.Repo{FullName: "Proj/api"}},
		{Number: 2, Title: "Newer", CreatedAt: now.Add(-time.Hour), PATID: testPATID, Repository: domain.Repo{FullName: "Proj/web"}},
		{Number: 3, Title: "No ticket", CreatedAt: now, PATID: testPATID, Repository: domain.Repo{FullName: "Proj/api"}},
	})
	return m, repository
}

// listedTitles returns the given titles in the order the PR list shows them.
func listedTitles(t *testing.T, m Model, titles ...string) []string {
	t.Helper()
	view := m.prListView.View()
	listed := slices.Clone(titles)
	for _, title := range titles {
		if !strings.Contains(view, title) {
			listed = slices.DeleteFunc(listed, func(listedTitle string) bool { return listedTitle == title })
		}
	}
	slices.SortFunc(listed, func(a, b string) int { return strings.Index(view, a) - strings.Index(view, b) })
	return listed
}

func TestPRSortKey_SortsTheListAndSavesTheMode(t *testing.T) {
	m, repository := sortTestModel(t)

	m, _ = pressKey(t, m, "1")

	if got := listedTitles(t, m, "Older", "Newer", "No ticket"); !slices.Equal(got, []string{"No ticket", "Newer", "Older"}) {
		t.Errorf("expected newest first, got %v", got)
	}
	if repository.settings.PRSortMode != domain.PRSortNewest {
		t.Errorf("expected the sort mode saved, got %q", repository.settings.PRSortMode)
	}
}

func TestPRSortKey_WorkItemModeLooksUpTheLinkedWorkItems(t *testing.T) {
	m, _ := sortTestModel(t)

	m.prListView.SetSortMode(domain.PRSortWorkItem)
	m, _ = m.handlePRWorkItemIDsLoaded(findMsg[PRWorkItemIDsLoadedMsg](t, m.loadPRListWorkItemIDs()))

	if got := listedTitles(t, m, "Older", "Newer", "No ticket"); !slices.Equal(got, []string{"Newer", "Older", "No ticket"}) {
		t.Errorf("expected PRs by lowest linked work item, unlinked last, got %v", got)
	}
}

func TestPRListFilter_MatchesLinkedWorkItemsOnceLookedUp(t *testing.T) {
	m, _ := sortTestModel(t)
	m.prListView.ActivateFilter()

	m, lookup := pressKey(t, m, "9")
	m, _ = pressKey(t, m, "0")
	m, _ = pressKey(t, m, "0")
	m, _ = m.handlePRWorkItemIDsLoaded(findMsg[PRWorkItemIDsLoadedMsg](t, lookup))

	if got := listedTitles(t, m, "Older", "Newer", "No ticket"); !slices.Equal(got, []string{"Older"}) {
		t.Errorf("expected only the PR linked to 900, got %v", got)
	}
}

func TestLoadPRListWorkItemIDs_NotNeededWithoutWorkItemSortOrFilter(t *testing.T) {
	m, _ := sortTestModel(t)

	if cmd := m.loadPRListWorkItemIDs(); cmd != nil {
		t.Error("expected no lookups while the list neither sorts nor filters by work items")
	}
}
