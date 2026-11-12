package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	dashboardTitle         = "Dashboard"
	dashboardEmptySection  = "Nothing here."
	dashboardChecksLoading = "  checking builds (%d left)..."
	dashboardFailingFormat = "%s  (failing: %s)"
	checksLoadingIcon      = "…"
	// dashboardFixedRows stay put above the scrolling sections: the title and the column header.
	dashboardFixedRows = listTitleRows + listHeaderRows
)

type dashboardSection struct {
	title    string
	includes func(pr domain.PullRequest, checks domain.Checks, checksLoaded bool) bool
}

// dashboardSections are what needs attention: reviews owed, own PRs that broke, own PRs that can go in.
var dashboardSections = []dashboardSection{
	{"Waiting for your review", func(pr domain.PullRequest, _ domain.Checks, _ bool) bool {
		return pr.Category == domain.PRCategoryAssigned && !pr.ReviewedByMe && !pr.IsDraft && pr.Status == domain.PRStatusOpen
	}},
	{"Failing checks", func(pr domain.PullRequest, checks domain.Checks, loaded bool) bool {
		return pr.Category == domain.PRCategoryAuthored && loaded && checks.State == domain.CheckStateFailing
	}},
	{"Ready to merge", func(pr domain.PullRequest, checks domain.Checks, loaded bool) bool {
		return pr.Category == domain.PRCategoryAuthored && pr.Status == domain.PRStatusOpen && !pr.IsDraft &&
			pr.ApprovalStatus == domain.ApprovalStatusApproved && pr.Mergeable &&
			loaded && (checks.State == domain.CheckStatePassing || checks.State == domain.CheckStateNone)
	}},
}

var checkAppearances = map[domain.CheckState]prCategoryAppearance{
	domain.CheckStatePassing: {icon: "✓", color: lipgloss.Color("#10B981")},
	domain.CheckStateFailing: {icon: "✗", color: lipgloss.Color("#EF4444")},
	domain.CheckStatePending: {icon: "●", color: lipgloss.Color("#3B82F6")},
	domain.CheckStateNone:    {icon: "○", color: lipgloss.Color("#6B7280")},
}

type dashboardEntry struct {
	section int
	pr      domain.PullRequest
}

// DashboardViewModel sorts the loaded pull requests into the dashboard sections. Checks are only
// fetched for the user's own pull requests, since the other sections do not need them.
type DashboardViewModel struct {
	prs           []domain.PullRequest
	checks        map[string]domain.Checks
	checksPending map[string]bool
	entries       []dashboardEntry
	cursor        int
	width         int
	height        int
}

func NewDashboardView() *DashboardViewModel {
	return &DashboardViewModel{checks: make(map[string]domain.Checks), checksPending: make(map[string]bool)}
}

func (m *DashboardViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *DashboardViewModel) SetPRs(prs []domain.PullRequest) {
	m.prs = prs
	m.rebuild()
}

// PendingChecks returns the user's open pull requests whose checks are not fetched yet, and marks
// them in flight.
func (m *DashboardViewModel) PendingChecks() []domain.PullRequest {
	var pending []domain.PullRequest
	for _, pr := range m.prs {
		key := pr.Key()
		if _, loaded := m.checks[key]; loaded || m.checksPending[key] || pr.Category != domain.PRCategoryAuthored || pr.Status != domain.PRStatusOpen {
			continue
		}
		m.checksPending[key] = true
		pending = append(pending, pr)
	}
	return pending
}

func (m *DashboardViewModel) SetChecks(prKey string, checks domain.Checks) {
	delete(m.checksPending, prKey)
	m.checks[prKey] = checks
	m.rebuild()
}

// ChecksFailed forgets the request; the pull request stays out of the sections that need checks.
func (m *DashboardViewModel) ChecksFailed(prKey string) {
	delete(m.checksPending, prKey)
}

// ResetChecks makes every check be fetched again, for a refresh.
func (m *DashboardViewModel) ResetChecks() {
	m.checks = make(map[string]domain.Checks)
	m.checksPending = make(map[string]bool)
	m.rebuild()
}

func (m *DashboardViewModel) SectionCounts() []int {
	counts := make([]int, len(dashboardSections))
	for _, entry := range m.entries {
		counts[entry.section]++
	}
	return counts
}

func (m *DashboardViewModel) GetSelectedPR() *domain.PullRequest {
	if m.cursor < 0 || m.cursor >= len(m.entries) {
		return nil
	}
	return &m.entries[m.cursor].pr
}

func (m *DashboardViewModel) MoveUp() {
	m.cursor = max(m.cursor-1, 0)
}

func (m *DashboardViewModel) MoveDown() {
	m.cursor = min(m.cursor+1, max(len(m.entries)-1, 0))
}

// rebuild sorts the pull requests into sections, newest update first, keeping the cursor on the
// same pull request in the same section.
func (m *DashboardViewModel) rebuild() {
	var selected *dashboardEntry
	if m.cursor < len(m.entries) {
		entry := m.entries[m.cursor]
		selected = &entry
	}

	m.entries = m.entries[:0]
	for section, definition := range dashboardSections {
		var members []domain.PullRequest
		for _, pr := range m.prs {
			checks, loaded := m.checks[pr.Key()]
			if definition.includes(pr, checks, loaded) {
				members = append(members, pr)
			}
		}
		sort.SliceStable(members, func(i, j int) bool { return members[i].UpdatedAt.After(members[j].UpdatedAt) })
		for _, pr := range members {
			m.entries = append(m.entries, dashboardEntry{section: section, pr: pr})
		}
	}

	m.cursor = min(m.cursor, max(len(m.entries)-1, 0))
	if selected == nil {
		return
	}
	for i, entry := range m.entries {
		if entry.section == selected.section && entry.pr.Key() == selected.pr.Key() {
			m.cursor = i
			return
		}
	}
}

func (m *DashboardViewModel) View() string {
	var b strings.Builder
	b.WriteString(renderListTitle(dashboardTitle))
	if loading := len(m.checksPending); loading > 0 {
		b.WriteString(listMutedStyle.Render(fmt.Sprintf(dashboardChecksLoading, loading)))
	}
	b.WriteString("\n\n")

	columns := prColumns(m.width, false)
	b.WriteString(renderListHeader(columns))
	b.WriteString("\n")
	b.WriteString(strings.Join(m.visibleLines(columns), "\n"))
	return RenderMainWindow(b.String(), m.width, m.height)
}

// visibleLines renders every section and keeps the window of lines around the selected row.
func (m *DashboardViewModel) visibleLines(columns []listColumn) []string {
	var lines []string
	cursorLine := 0
	counts := m.SectionCounts()
	entry := 0
	for section, definition := range dashboardSections {
		if section > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, listHeaderStyle.Render(fmt.Sprintf("%s (%d)", definition.title, counts[section])))
		if counts[section] == 0 {
			lines = append(lines, unselectedRowMarker+listMutedStyle.Render(dashboardEmptySection))
		}
		for ; entry < len(m.entries) && m.entries[entry].section == section; entry++ {
			if entry == m.cursor {
				cursorLine = len(lines)
			}
			lines = append(lines, renderListRow(m.entryRow(m.entries[entry]), columns, entry == m.cursor))
		}
	}

	visible := max(m.height-dashboardFixedRows, minVisibleListRows)
	if len(lines) <= visible {
		return lines
	}
	start := min(max(cursorLine-visible/2, 0), len(lines)-visible)
	return lines[start : start+visible]
}

func (m *DashboardViewModel) entryRow(entry dashboardEntry) listRow {
	row := prRow(entry.pr, false)
	checks, loaded := m.checks[entry.pr.Key()]
	switch {
	case loaded:
		appearance := checkAppearances[checks.State]
		row.icon = renderListIcon(appearance.icon, appearance.color)
	case m.checksPending[entry.pr.Key()]:
		row.icon = renderListIcon(checksLoadingIcon, lipgloss.Color("#6B7280"))
	}
	if failing := checks.Failing(); len(failing) > 0 {
		row.values[1] = fmt.Sprintf(dashboardFailingFormat, row.values[1], strings.Join(failing, ", "))
	}
	return row
}
