package views

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
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
	width       int
	height      int
	filterInput textinput.Model
	filtering   bool
	filterText  string
}

func NewPRListView() *PRListViewModel {
	columns := []table.Column{
		{Title: "", Width: 4},
		{Title: "", Width: 4},
		{Title: "", Width: 50},
		{Title: "", Width: 22},
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

	ti := textinput.New()
	ti.Placeholder = "Filter by title, author, or PR number..."
	ti.CharLimit = 100

	return &PRListViewModel{
		table:       t,
		filterInput: ti,
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
		repoWidth     = 22
		numberWidth   = 7
		authorWidth   = 15
		ageWidth      = 14
		rightPadWidth = 4
		minTitleWidth = 20
		maxTitleWidth = 100
		padding       = 0
	)

	fixed := categoryWidth + approvalWidth + repoWidth + numberWidth +
		authorWidth + ageWidth + rightPadWidth + padding

	available := max(0, m.width-fixed)
	titleWidth := max(minTitleWidth, min(available, maxTitleWidth))

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
	filtered := m.filterPRs(m.sourcePRs)
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

func (m *PRListViewModel) filterPRs(prs []domain.PullRequest) []domain.PullRequest {
	if m.filterText == "" {
		return prs
	}

	filter := strings.ToLower(m.filterText)
	var out []domain.PullRequest

	for _, pr := range prs {
		if strings.Contains(strings.ToLower(pr.Title), filter) ||
			strings.Contains(strings.ToLower(pr.Author.Username), filter) ||
			strings.Contains(strconv.Itoa(pr.Number), filter) {
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

func (m *PRListViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if m.filtering {
		m.filterInput, cmd = m.filterInput.Update(msg)
	} else {
		m.table, cmd = m.table.Update(msg)
		if m.table.Cursor() == 0 && len(m.visiblePRs) > 0 {
			m.table.SetCursor(1)
		}
	}
	return cmd
}

func (m *PRListViewModel) ActivateFilter() {
	m.filtering = true
	m.filterInput.SetValue(m.filterText)
	m.filterInput.Focus()
}

func (m *PRListViewModel) DeactivateFilter() {
	m.filtering = false
	m.filterInput.Blur()
}

func (m *PRListViewModel) ApplyFilter() {
	m.filterText = m.filterInput.Value()
	m.filtering = false
	m.filterInput.Blur()
	m.rebuild()
}

func (m *PRListViewModel) ClearFilter() {
	m.filterText = ""
	m.filterInput.SetValue("")
	m.filtering = false
	m.filterInput.Blur()
	m.rebuild()
}

func (m *PRListViewModel) View() string {
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Render("\n" + m.helpText())

	tableView := m.colorizeTableRows(m.table.View())

	var content string
	if m.filtering {
		filterStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true)
		content = tableView +
			"\n" + filterStyle.Render("Filter: ") + m.filterInput.View() +
			help
	} else {
		content = tableView + help
	}

	return content
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
	if m.filtering {
		return "Type to filter | Enter/Esc: Close"
	}
	if m.filterText != "" {
		return "Enter: Inspect | r: Refresh | /: Filter | Esc: Clear filter | q: Back"
	}
	return "Enter: Inspect | r: Refresh | /: Filter | q: Back"
}

func (m *PRListViewModel) IsFiltering() bool {
	return m.filtering
}

func (m *PRListViewModel) UpdateFilterInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	return cmd
}

func (m *PRListViewModel) ApplyFilterFromInput() {
	m.filterText = m.filterInput.Value()
	m.rebuild()
}

func (m *PRListViewModel) GetFilterText() string {
	return m.filterText
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
