package views

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const testSortPATID = "pat"

func sortTestPRs() []domain.PullRequest {
	now := time.Now()
	return []domain.PullRequest{
		{Number: 1, Title: "beta", Category: domain.PRCategoryOther, Author: domain.User{Username: "zed"},
			Repository: domain.Repo{FullName: "Proj/web"}, CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now, PATID: testSortPATID},
		{Number: 2, Title: "Alpha", Category: domain.PRCategoryAuthored, Author: domain.User{Username: "amy"},
			Repository: domain.Repo{FullName: "Proj/web"}, CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour), PATID: testSortPATID},
		{Number: 3, Title: "gamma", Category: domain.PRCategoryAssigned, Author: domain.User{Username: "Bob"},
			Repository: domain.Repo{FullName: "Proj/api"}, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-time.Hour), PATID: testSortPATID},
	}
}

func visibleNumbers(m *PRListViewModel) []int {
	numbers := make([]int, len(m.visiblePRs))
	for i, pr := range m.visiblePRs {
		numbers[i] = pr.Number
	}
	return numbers
}

func TestPRListSortModes(t *testing.T) {
	tests := []struct {
		mode domain.PRSortMode
		want []int
	}{
		{domain.PRSortDefault, []int{2, 3, 1}},
		{domain.PRSortNewest, []int{2, 3, 1}},
		{domain.PRSortTitle, []int{2, 1, 3}},
		{domain.PRSortRepository, []int{3, 2, 1}},
		{domain.PRSortAuthor, []int{2, 3, 1}},
	}
	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			m := NewPRListView()
			m.SetPRs(sortTestPRs())
			m.SetSortMode(tt.mode)
			if got := visibleNumbers(m); !slices.Equal(got, tt.want) {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestPRListSortByNewest_GoesByCreationNotUpdate(t *testing.T) {
	m := NewPRListView()
	prs := sortTestPRs()
	m.SetPRs([]domain.PullRequest{prs[0], prs[2]})

	m.SetSortMode(domain.PRSortNewest)

	if got := visibleNumbers(m); !slices.Equal(got, []int{3, 1}) {
		t.Errorf("expected the most recently created first, got %v", got)
	}
}

func TestPRListSortByWorkItem_PutsUnlinkedPRsLast(t *testing.T) {
	m := NewPRListView()
	prs := sortTestPRs()
	m.SetPRs(prs)
	m.SetSortMode(domain.PRSortWorkItem)

	m.SetWorkItemIDs(map[string][]int{prs[0].Key(): {40, 7}, prs[1].Key(): {12}, prs[2].Key(): nil}, nil)

	if got := visibleNumbers(m); !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("expected PRs by lowest linked work item, unlinked last, got %v", got)
	}
}

func TestPRListSortMode_ClosedListDatesByClosing(t *testing.T) {
	m := NewPRListView()
	now := time.Now()
	m.SetClosedPRs([]domain.PullRequest{
		{Number: 1, CreatedAt: now, ClosedAt: now.Add(-time.Hour), Status: domain.PRStatusMerged},
		{Number: 2, CreatedAt: now.Add(-time.Hour), ClosedAt: now, Status: domain.PRStatusMerged},
	})
	m.ShowClosed(true)

	m.SetSortMode(domain.PRSortNewest)

	if got := visibleNumbers(m); !slices.Equal(got, []int{2, 1}) {
		t.Errorf("expected the most recently closed first, got %v", got)
	}
}

func TestPRListSettings_RestoreTheSortMode(t *testing.T) {
	m := NewPRListView()

	m.SetSettings(domain.Settings{PRSortMode: domain.PRSortTitle})

	if m.SortMode() != domain.PRSortTitle {
		t.Errorf("expected the saved sort mode, got %q", m.SortMode())
	}
}

func TestPRListFilter_MatchesRepositoryAndLinkedWorkItems(t *testing.T) {
	m := NewPRListView()
	prs := sortTestPRs()
	m.SetPRs(prs)
	m.SetWorkItemIDs(map[string][]int{prs[1].Key(): {4711}}, nil)

	m.filter.text = "proj/api"
	m.rebuild()
	if got := visibleNumbers(m); !slices.Equal(got, []int{3}) {
		t.Errorf("expected the repository to match, got %v", got)
	}

	m.filter.text = "4711"
	m.rebuild()
	if got := visibleNumbers(m); !slices.Equal(got, []int{2}) {
		t.Errorf("expected the linked work item to match, got %v", got)
	}
}

func TestPendingWorkItemIDRequests_OnlyWhileSortingOrFilteringByThem(t *testing.T) {
	m := NewPRListView()
	m.SetPRs(sortTestPRs())

	if pending := m.PendingWorkItemIDRequests(); len(pending) != 0 {
		t.Fatalf("expected no lookups by default, got %d", len(pending))
	}

	m.SetSortMode(domain.PRSortWorkItem)
	if pending := m.PendingWorkItemIDRequests(); len(pending) != 3 {
		t.Fatalf("expected every PR looked up, got %d", len(pending))
	}
	if pending := m.PendingWorkItemIDRequests(); len(pending) != 0 {
		t.Errorf("expected lookups in flight not requested again, got %d", len(pending))
	}
}

func TestSetWorkItemIDs_FailedLookupsAreRequestedAgain(t *testing.T) {
	m := NewPRListView()
	prs := sortTestPRs()
	m.SetPRs(prs)
	m.SetSortMode(domain.PRSortWorkItem)
	m.PendingWorkItemIDRequests()

	m.SetWorkItemIDs(map[string][]int{prs[0].Key(): {1}, prs[1].Key(): nil}, []string{prs[2].Key()})

	pending := m.PendingWorkItemIDRequests()
	if len(pending) != 1 || pending[0].Number != 3 {
		t.Errorf("expected only the failed PR requested again, got %+v", pending)
	}
}

func TestPRInfo_ShowsLinkedWorkItemsOnceLookedUp(t *testing.T) {
	m := NewPRListView()
	prs := sortTestPRs()
	m.SetPRs(prs)
	pr := prs[1]

	if lines := m.prInfoLines(pr); slices.ContainsFunc(lines, containsText(workItemsPaneTitle)) {
		t.Error("expected no work items row before the lookup")
	}
	m.SetWorkItemIDs(map[string][]int{pr.Key(): {12, 7}}, nil)
	if lines := m.prInfoLines(pr); !slices.ContainsFunc(lines, containsText("#12, #7")) {
		t.Errorf("expected the linked work items listed, got %v", lines)
	}
}

func containsText(text string) func(string) bool {
	return func(line string) bool { return strings.Contains(line, text) }
}
