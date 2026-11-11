package views

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/mattn/go-runewidth"
)

func getCategoryIndicator(category domain.PRCategory) string {
	switch category {
	case domain.PRCategoryAuthored:
		return " ✎ "
	case domain.PRCategoryAssigned:
		return " → "
	default:
		return " ○ "
	}
}

func getApprovalBadge(status domain.ApprovalStatus) string {
	switch status {
	case domain.ApprovalStatusApproved:
		return " x "
	case domain.ApprovalStatusChangesRequested:
		return " # "
	case domain.ApprovalStatusPending:
		return " o "
	default:
		return "   "
	}
}

type PRListViewModel struct {
	table table.Model

	// Source data (never mutated by sorting/filtering)
	sourcePRs    []domain.PullRequest
	sourceGroups []domain.PRGroup

	// Derived view data (filtered + sorted)
	visiblePRs []domain.PullRequest

	// UI state
	width    int
	height   int
	filter   listFilter
	settings domain.Settings
}

func NewPRListView() *PRListViewModel {
	columns := []table.Column{
		{Title: "", Width: 4},
		{Title: "", Width: 4},
		{Title: "", Width: 35},
		{Title: "", Width: 55},
		{Title: "", Width: 7},
		{Title: "", Width: 15},
		{Title: "", Width: 14},
		{Title: "", Width: 4},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.HiddenBorder()).
		Bold(false).
		Height(0)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#F59E0B")).
		Background(lipgloss.Color("#1F2937")).
		Bold(true)
	t.SetStyles(s)

	return &PRListViewModel{
		table:  t,
		filter: newListFilter("Filter by title, author, or PR number..."),
	}
}

func (m *PRListViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.table.SetHeight(max(1, height-7))
	m.updateColumnWidths()
}

func (m *PRListViewModel) updateColumnWidths() {
	const (
		categoryWidth = 4
		approvalWidth = 4
		repoWidth     = 40
		numberWidth   = 7
		authorWidth   = 15
		ageWidth      = 14
		rightPadWidth = 10
		minTitleWidth = 20
		padding       = 0
	)

	fixed := categoryWidth + approvalWidth + repoWidth + numberWidth +
		authorWidth + ageWidth + rightPadWidth + padding

	titleWidth := max(minTitleWidth, m.width-fixed)

	columns := []table.Column{
		{Title: "", Width: categoryWidth},
		{Title: "", Width: approvalWidth},
		{Title: "", Width: titleWidth},
		{Title: "", Width: repoWidth},
		{Title: "", Width: numberWidth},
		{Title: "", Width: authorWidth},
		{Title: "", Width: ageWidth},
		{Title: "", Width: rightPadWidth},
	}
	m.table.SetColumns(columns)
}

func (m *PRListViewModel) SetSettings(settings domain.Settings) {
	m.settings = settings
	m.rebuild()
}

func (m *PRListViewModel) SetPRs(prs []domain.PullRequest) {
	m.sourceGroups = nil
	m.sourcePRs = append([]domain.PullRequest(nil), prs...)
	m.rebuild()
}

func (m *PRListViewModel) SetPRGroups(groups []domain.PRGroup) {
	m.sourceGroups = groups
	m.sourcePRs = flattenGroups(groups)
	m.rebuild()
}

// source → filter → sort → visible → rows
func (m *PRListViewModel) rebuild() {
	filtered := m.filterOldOtherPRs(m.sourcePRs)
	filtered = m.filterPRs(filtered)
	sorted := sortPRs(filtered)
	m.visiblePRs = sorted
	m.table.SetRows(m.prsToRows(sorted))
	if len(sorted) > 0 {
		m.table.SetCursor(1)
	}
}

func sortPRs(prs []domain.PullRequest) []domain.PullRequest {
	out := append([]domain.PullRequest(nil), prs...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			order := map[domain.PRCategory]int{
				domain.PRCategoryAuthored: 0,
				domain.PRCategoryAssigned: 1,
				domain.PRCategoryOther:    2,
			}
			return order[out[i].Category] < order[out[j].Category]
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func (m *PRListViewModel) filterOldOtherPRs(prs []domain.PullRequest) []domain.PullRequest {
	if !m.settings.HideOldOtherPRs {
		return prs
	}

	maxAge := time.Duration(m.settings.HideOldOtherPRsDays) * 24 * time.Hour
	cutoff := time.Now().Add(-maxAge)

	var out []domain.PullRequest
	for _, pr := range prs {
		if pr.Category == domain.PRCategoryOther && pr.CreatedAt.Before(cutoff) {
			continue
		}
		out = append(out, pr)
	}
	return out
}

func (m *PRListViewModel) filterPRs(prs []domain.PullRequest) []domain.PullRequest {
	var out []domain.PullRequest
	for _, pr := range prs {
		if m.filter.Matches(pr.Title, pr.Author.Username, strconv.Itoa(pr.Number)) {
			out = append(out, pr)
		}
	}
	return out
}

func (m *PRListViewModel) prsToRows(prs []domain.PullRequest) []table.Row {
	cols := m.table.Columns()
	rows := make([]table.Row, len(prs)+1)

	rows[0] = m.headerRow(cols)

	for i, pr := range prs {
		rows[i+1] = table.Row{
			padToWidth(getCategoryIndicator(pr.Category), cols[0].Width),
			padToWidth(getApprovalBadge(pr.ApprovalStatus), cols[1].Width),
			padToWidth(truncateString(pr.Title, cols[2].Width), cols[2].Width),
			padToWidth(truncateString(pr.Repository.FullName, cols[3].Width), cols[3].Width),
			padToWidth(truncateString(fmt.Sprintf("#%d", pr.Number), cols[4].Width), cols[4].Width),
			padToWidth(truncateString(pr.Author.Username, cols[5].Width), cols[5].Width),
			padToWidth(truncateString(formatAge(pr.CreatedAt), cols[6].Width), cols[6].Width),
			padToWidth("", cols[7].Width),
		}
	}
	return rows
}

// Hack to get header alignment to work properly  - create a "header row" at index 0
func (m *PRListViewModel) headerRow(cols []table.Column) table.Row {
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	return table.Row{
		padToWidth("", cols[0].Width),
		padToWidth("", cols[1].Width),
		padToWidth(headerStyle.Render("Title"), cols[2].Width),
		padToWidth(headerStyle.Render("Repo"), cols[3].Width),
		padToWidth(headerStyle.Render("#"), cols[4].Width),
		padToWidth(headerStyle.Render("Author"), cols[5].Width),
		padToWidth(headerStyle.Render("Age"), cols[6].Width),
		padToWidth("", cols[7].Width),
	}
}

func (m *PRListViewModel) GetSelectedPR() *domain.PullRequest {
	idx := m.table.Cursor() - 1
	if idx < 0 || idx >= len(m.visiblePRs) {
		return nil
	}
	return &m.visiblePRs[idx]
}

func (m *PRListViewModel) GetCursorIndex() int {
	return m.table.Cursor()
}

func (m *PRListViewModel) RestoreCursor(index int) {
	rows := m.table.Rows()
	if len(rows) == 0 {
		return
	}

	if index >= len(rows) {
		index = len(rows) - 1
	}
	if index < 0 {
		index = 0
	}

	m.table.SetCursor(index)
}

func (m *PRListViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if m.filter.IsActive() {
		cmd = m.filter.Update(msg)
		m.rebuild()
	} else {
		m.table, cmd = m.table.Update(msg)
		if m.table.Cursor() == 0 && len(m.visiblePRs) > 0 {
			m.table.SetCursor(1)
		}
	}
	return cmd
}

func (m *PRListViewModel) ActivateFilter() {
	m.filter.Activate()
}

func (m *PRListViewModel) DeactivateFilter() {
	m.filter.Deactivate()
}

func (m *PRListViewModel) ClearFilter() {
	m.filter.Clear()
	m.rebuild()
}

func (m *PRListViewModel) View() string {
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Render("\n" + m.helpText())

	tableView := m.colorizeTableRows(m.table.View())

	if m.filter.IsActive() {
		return tableView + "\n" + m.filter.View() + help
	}
	return tableView + help
}

// post effect render rows.
func (m *PRListViewModel) colorizeTableRows(tableOutput string) string {
	lines := strings.Split(tableOutput, "\n")
	authoredStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#86EFAC"))
	otherStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	for i, line := range lines {
		if strings.Contains(line, " ✎ ") {
			lines[i] = authoredStyle.Render(line)
		} else if strings.Contains(line, " ○ ") {
			lines[i] = otherStyle.Render(line)
		}
	}
	return strings.Join(lines, "\n")
}

func (m *PRListViewModel) helpText() string {
	if m.filter.IsActive() {
		return "Type to filter | Enter/Esc: Close"
	}
	if m.filter.Text() != "" {
		return "Enter: Inspect | r: Refresh | /: Filter | Esc: Clear filter | q: Back"
	}
	return "Enter: Inspect | r: Refresh | /: Filter | q: Back"
}

func (m *PRListViewModel) IsFiltering() bool {
	return m.filter.IsActive()
}

func (m *PRListViewModel) UpdateFilter(msg tea.Msg) tea.Cmd {
	cmd := m.filter.Update(msg)
	m.rebuild()
	return cmd
}

func (m *PRListViewModel) GetFilterText() string {
	return m.filter.Text()
}

func truncateString(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return runewidth.Truncate(s, maxWidth, "")
	}
	return runewidth.Truncate(s, maxWidth-3, "") + "..."
}

func padToWidth(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return runewidth.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-w)
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

func flattenGroups(groups []domain.PRGroup) []domain.PullRequest {
	var out []domain.PullRequest
	for _, g := range groups {
		out = append(out, g.PRs...)
	}
	return out
}
