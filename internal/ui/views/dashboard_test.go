package views

import (
	"strings"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func dashboardPR(number int, title string, category domain.PRCategory, edit func(pr *domain.PullRequest)) domain.PullRequest {
	pr := domain.PullRequest{
		Number: number, Title: title, Category: category, Status: domain.PRStatusOpen,
		Repository: domain.Repo{FullName: "Proj/api"}, UpdatedAt: time.Now().Add(-time.Duration(number) * time.Minute),
	}
	if edit != nil {
		edit(&pr)
	}
	return pr
}

func dashboardTestPRs() []domain.PullRequest {
	approvedAndMergeable := func(pr *domain.PullRequest) {
		pr.ApprovalStatus = domain.ApprovalStatusApproved
		pr.Mergeable = true
	}
	return []domain.PullRequest{
		dashboardPR(1, "Needs my eyes", domain.PRCategoryAssigned, nil),
		dashboardPR(2, "Already reviewed", domain.PRCategoryAssigned, func(pr *domain.PullRequest) { pr.ReviewedByMe = true }),
		dashboardPR(3, "Draft for me", domain.PRCategoryAssigned, func(pr *domain.PullRequest) { pr.IsDraft = true }),
		dashboardPR(4, "Broken build", domain.PRCategoryAuthored, approvedAndMergeable),
		dashboardPR(5, "Good to go", domain.PRCategoryAuthored, approvedAndMergeable),
		dashboardPR(6, "Not approved", domain.PRCategoryAuthored, nil),
		dashboardPR(7, "Someone else's", domain.PRCategoryOther, nil),
	}
}

func dashboardTitles(m *DashboardViewModel) []string {
	var titles []string
	for _, entry := range m.entries {
		titles = append(titles, entry.pr.Title)
	}
	return titles
}

func TestDashboard_SortsPullRequestsIntoSections(t *testing.T) {
	m := NewDashboardView()
	m.SetSize(160, 40)
	m.SetPRs(dashboardTestPRs())

	pending := m.PendingChecks()
	if len(pending) != 3 {
		t.Fatalf("expected checks for the three authored PRs only, got %d", len(pending))
	}
	if len(m.PendingChecks()) != 0 {
		t.Error("expected checks in flight not to be requested again")
	}
	if got := strings.Join(dashboardTitles(m), ","); got != "Needs my eyes" {
		t.Errorf("expected only the review section before checks load, got %s", got)
	}

	for _, pr := range pending {
		state := domain.CheckStatePassing
		if pr.Number == 4 {
			state = domain.CheckStateFailing
		}
		m.SetChecks(pr.Key(), domain.SummarizeChecks([]domain.Check{{Name: "ci", State: state}}))
	}

	if got := strings.Join(dashboardTitles(m), ","); got != "Needs my eyes,Broken build,Good to go" {
		t.Errorf("unexpected sections %s", got)
	}
	if counts := m.SectionCounts(); counts[0] != 1 || counts[1] != 1 || counts[2] != 1 {
		t.Errorf("expected one PR per section, got %v", counts)
	}
	view := m.View()
	for _, expected := range []string{"Waiting for your review (1)", "Failing checks (1)", "Ready to merge (1)", "Broken build  (failing: ci)"} {
		if !strings.Contains(view, expected) {
			t.Errorf("expected %q in the dashboard", expected)
		}
	}
}

func TestDashboard_KeepsTheSelectionWhenChecksArrive(t *testing.T) {
	m := NewDashboardView()
	m.SetSize(160, 40)
	m.SetPRs(dashboardTestPRs())
	pending := m.PendingChecks()
	m.SetChecks(pending[1].Key(), domain.SummarizeChecks([]domain.Check{{State: domain.CheckStatePassing}}))
	m.MoveDown()
	if m.GetSelectedPR().Title != "Good to go" {
		t.Fatalf("expected Good to go selected, got %s", m.GetSelectedPR().Title)
	}

	m.SetChecks(pending[0].Key(), domain.SummarizeChecks([]domain.Check{{State: domain.CheckStateFailing}}))

	if m.GetSelectedPR().Title != "Good to go" {
		t.Errorf("expected the selection to stay on Good to go, got %s", m.GetSelectedPR().Title)
	}
}

func TestDashboard_EmptySectionsSaySo(t *testing.T) {
	m := NewDashboardView()
	m.SetSize(160, 40)

	if strings.Count(m.View(), dashboardEmptySection) != len(dashboardSections) {
		t.Error("expected every empty section to say so")
	}
}
