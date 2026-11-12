package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	dashboardEmptySection  = "Nothing here."
	dashboardChecksLoading = " · checking builds (%d left)..."
	dashboardFailingFormat = "%s  (failing: %s)"
	checksLoadingIcon      = "…"
	// dashboardPaneChromeRows are the rows of a pane that are not pull requests: its title, the blank
	// line under it, the column header and the scroll position.
	dashboardPaneChromeRows = listTitleRows + listHeaderRows + listScrollInfoRows
	dashboardColumns        = 2
	dashboardRows           = 2
	noCursor                = -1
)

type dashboardSection struct {
	title    string
	includes func(pr domain.PullRequest, checks domain.Checks, checksLoaded bool) bool
}

func isOwnOpenPR(pr domain.PullRequest) bool {
	return pr.Category == domain.PRCategoryAuthored && pr.Status == domain.PRStatusOpen
}

// dashboardSections fill the panes column by column: the left column holds what waits on someone,
// the right one the user's pull requests that need action.
var dashboardSections = []dashboardSection{
	{"Waiting for your review", func(pr domain.PullRequest, _ domain.Checks, _ bool) bool {
		return pr.Category == domain.PRCategoryAssigned && !pr.ReviewedByMe && !pr.IsDraft && pr.Status == domain.PRStatusOpen
	}},
	{"Created by you", func(pr domain.PullRequest, _ domain.Checks, _ bool) bool {
		return isOwnOpenPR(pr)
	}},
	{"Failing checks", func(pr domain.PullRequest, checks domain.Checks, loaded bool) bool {
		return isOwnOpenPR(pr) && loaded && checks.State == domain.CheckStateFailing
	}},
	{"Ready to merge", func(pr domain.PullRequest, checks domain.Checks, loaded bool) bool {
		return isOwnOpenPR(pr) && !pr.IsDraft && pr.ApprovalStatus == domain.ApprovalStatusApproved && pr.Mergeable &&
			loaded && (checks.State == domain.CheckStatePassing || checks.State == domain.CheckStateNone)
	}},
}

var checkAppearances = map[domain.CheckState]prCategoryAppearance{
	domain.CheckStatePassing: {icon: "✓", color: lipgloss.Color("#10B981")},
	domain.CheckStateFailing: {icon: "✗", color: lipgloss.Color("#EF4444")},
	domain.CheckStatePending: {icon: "●", color: lipgloss.Color("#3B82F6")},
	domain.CheckStateNone:    {icon: "○", color: lipgloss.Color("#6B7280")},
}

// DashboardViewModel shows the loaded pull requests in a fixed grid of panes, one per section, each
// with its own cursor. Checks are only fetched for the user's own pull requests.
type DashboardViewModel struct {
	prs           []domain.PullRequest
	checks        map[string]domain.Checks
	checksPending map[string]bool
	sections      [][]domain.PullRequest
	cursors       []listCursor
	focus         int
	// focusChosen stops the focus from following the first filled pane once the user moved it.
	focusChosen bool
	width       int
	height      int
}

func NewDashboardView() *DashboardViewModel {
	cursors := make([]listCursor, len(dashboardSections))
	for i := range cursors {
		cursors[i] = newListCursor()
	}
	return &DashboardViewModel{
		checks:        make(map[string]domain.Checks),
		checksPending: make(map[string]bool),
		sections:      make([][]domain.PullRequest, len(dashboardSections)),
		cursors:       cursors,
	}
}

func (m *DashboardViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	layout := m.layout()
	for i := range m.cursors {
		m.cursors[i].SetVisibleRows(layout.heights[i%dashboardRows] - dashboardPaneChromeRows)
	}
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
		if _, loaded := m.checks[key]; loaded || m.checksPending[key] || !isOwnOpenPR(pr) {
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
	counts := make([]int, len(m.sections))
	for i, section := range m.sections {
		counts[i] = len(section)
	}
	return counts
}

// GetSelectedPR is the pull request under the cursor of the focused pane.
func (m *DashboardViewModel) GetSelectedPR() *domain.PullRequest {
	section := m.sections[m.focus]
	index := m.cursors[m.focus].Index()
	if index < 0 || index >= len(section) {
		return nil
	}
	return &section[index]
}

func (m *DashboardViewModel) MoveUp() {
	m.cursors[m.focus].MoveUp()
}

func (m *DashboardViewModel) MoveDown() {
	m.cursors[m.focus].MoveDown(len(m.sections[m.focus]))
}

// FocusNext and FocusPrev cycle through the panes column by column.
func (m *DashboardViewModel) FocusNext() {
	m.setFocus((m.focus + 1) % len(dashboardSections))
}

func (m *DashboardViewModel) FocusPrev() {
	m.setFocus((m.focus + len(dashboardSections) - 1) % len(dashboardSections))
}

// FocusRight and FocusLeft move to the pane beside the focused one.
func (m *DashboardViewModel) FocusRight() {
	if m.focus+dashboardRows < len(dashboardSections) {
		m.setFocus(m.focus + dashboardRows)
	}
}

func (m *DashboardViewModel) FocusLeft() {
	if m.focus >= dashboardRows {
		m.setFocus(m.focus - dashboardRows)
	}
}

func (m *DashboardViewModel) setFocus(section int) {
	m.focus = section
	m.focusChosen = true
}

// rebuild sorts the pull requests into the sections, newest update first, keeping each pane's
// cursor on the same pull request.
func (m *DashboardViewModel) rebuild() {
	for section, definition := range dashboardSections {
		selectedKey := ""
		if index := m.cursors[section].Index(); index < len(m.sections[section]) {
			selectedKey = m.sections[section][index].Key()
		}

		var members []domain.PullRequest
		for _, pr := range m.prs {
			checks, loaded := m.checks[pr.Key()]
			if definition.includes(pr, checks, loaded) {
				members = append(members, pr)
			}
		}
		sort.SliceStable(members, func(i, j int) bool { return members[i].UpdatedAt.After(members[j].UpdatedAt) })
		m.sections[section] = members

		m.cursors[section].Clamp(len(members))
		for i, pr := range members {
			if pr.Key() == selectedKey {
				m.cursors[section].SetIndex(i, len(members))
			}
		}
	}

	if !m.focusChosen {
		for section, members := range m.sections {
			if len(members) > 0 {
				m.focus = section
				break
			}
		}
	}
}

// dashboardLayout holds the inner sizes of the grid's columns and rows, which together occupy
// exactly the main window.
type dashboardLayout struct {
	widths  []int
	heights []int
}

func (m *DashboardViewModel) layout() dashboardLayout {
	return dashboardLayout{
		widths:  splitEvenly(outerSize(m.width, paneHorizontalPadding), dashboardColumns, paneHorizontalPadding),
		heights: splitEvenly(outerSize(m.height, paneVerticalPadding), dashboardRows, paneVerticalPadding),
	}
}

// splitEvenly divides an outer size into parts, the last one taking the rounding remainder, and
// returns their inner sizes.
func splitEvenly(outer, parts, padding int) []int {
	sizes := make([]int, parts)
	for i := range sizes {
		part := outer / parts
		if i == parts-1 {
			part = outer - part*(parts-1)
		}
		sizes[i] = innerSize(part, padding)
	}
	return sizes
}

func (m *DashboardViewModel) View() string {
	layout := m.layout()
	columns := make([]string, dashboardColumns)
	for column := range columns {
		panes := make([]string, dashboardRows)
		for row := range panes {
			section := column*dashboardRows + row
			borderColor := mainWindowBorderColor
			if section == m.focus {
				borderColor = focusedPaneBorderColor
			}
			width, height := layout.widths[column], layout.heights[row]
			panes[row] = renderPane(m.renderSection(section, width), width, height, borderColor)
		}
		columns[column] = lipgloss.JoinVertical(lipgloss.Left, panes...)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

func (m *DashboardViewModel) renderSection(section, width int) string {
	members := m.sections[section]
	title := fmt.Sprintf("%s (%d)", dashboardSections[section].title, len(members))
	if loading := len(m.checksPending); loading > 0 && section >= dashboardRows {
		title += fmt.Sprintf(dashboardChecksLoading, loading)
	}

	var b strings.Builder
	b.WriteString(listTitleStyle.Render(title))
	b.WriteString("\n\n")
	if len(members) == 0 {
		b.WriteString(listMutedStyle.Render(dashboardEmptySection))
		return b.String()
	}

	cursor := m.cursors[section]
	if section != m.focus {
		// Only the focused pane shows its cursor; the copy keeps the pane's scroll position.
		cursor.index = noCursor
	}
	b.WriteString(renderListRows(prColumns(width, true), len(members), cursor, func(i int) listRow {
		return m.prRow(members[i])
	}))
	return b.String()
}

// prRow shows the checks in the icon column once they are known, and names the failing ones.
func (m *DashboardViewModel) prRow(pr domain.PullRequest) listRow {
	row := prRow(pr, true)
	checks, loaded := m.checks[pr.Key()]
	switch {
	case loaded:
		appearance := checkAppearances[checks.State]
		row.icon = renderListIcon(appearance.icon, appearance.color)
	case m.checksPending[pr.Key()]:
		row.icon = renderListIcon(checksLoadingIcon, lipgloss.Color("#6B7280"))
	}
	if failing := checks.Failing(); len(failing) > 0 {
		row.values[1] = fmt.Sprintf(dashboardFailingFormat, row.values[1], strings.Join(failing, ", "))
	}
	return row
}
