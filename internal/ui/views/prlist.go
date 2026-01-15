package views

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type SectionHeaderItem struct {
	text string
}

func (i SectionHeaderItem) FilterValue() string { return "" }
func (i SectionHeaderItem) Title() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Padding(0, 1)
	return style.Render("── " + i.text + " ──")
}
func (i SectionHeaderItem) Description() string { return "" }

type PRItem struct {
	pr domain.PullRequest
}

func (i PRItem) FilterValue() string { return i.pr.Title }

func (i PRItem) Title() string {
	categoryIndicator := ""
	style := lipgloss.NewStyle()

	switch i.pr.Category {
	case domain.PRCategoryAuthored:
		categoryIndicator = "✎"
		style = style.Foreground(lipgloss.Color("#10B981")).Bold(true)
	case domain.PRCategoryAssigned:
		categoryIndicator = "→"
		style = style.Foreground(lipgloss.Color("#F59E0B")).Bold(true)
	default:
		categoryIndicator = "○"
		style = style.Foreground(lipgloss.Color("#6B7280"))
	}

	if i.pr.IsDraft {
		style = style.Italic(true)
	}

	approvalBadge := formatApprovalBadge(i.pr.ApprovalStatus)

	title := fmt.Sprintf("%s %s%s", categoryIndicator, approvalBadge, i.pr.Title)
	return style.Render(title)
}

func formatApprovalBadge(status domain.ApprovalStatus) string {
	switch status {
	case domain.ApprovalStatusApproved:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("✓ ")
	case domain.ApprovalStatusChangesRequested:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render("✗ ")
	case domain.ApprovalStatusPending:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render("◯ ")
	default:
		return ""
	}
}

func (i PRItem) Description() string {
	repo := i.pr.Repository.FullName
	number := fmt.Sprintf("#%d", i.pr.Number)
	author := i.pr.Author.Username
	age := formatAge(i.pr.CreatedAt)

	return fmt.Sprintf("%s %s by %s • %s", repo, number, author, age)
}

type PRListViewModel struct {
	list        list.Model
	prs         []domain.PullRequest
	allPRs      []domain.PullRequest
	allGroups   []domain.PRGroup
	width       int
	height      int
	filterInput textinput.Model
	filtering   bool
	filterText  string
}

func NewPRListView() *PRListViewModel {
	items := []list.Item{}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Pull Requests"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)

	ti := textinput.New()
	ti.Placeholder = "Filter by title, author, or PR number..."
	ti.CharLimit = 100

	return &PRListViewModel{
		list:        l,
		prs:         []domain.PullRequest{},
		filterInput: ti,
	}
}

func (m *PRListViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height-5)
}

func (m *PRListViewModel) SetPRs(prs []domain.PullRequest) {
	m.prs = prs
	m.allPRs = prs
	m.allGroups = nil

	m.applyFilter()
}

func (m *PRListViewModel) sortAndSetItems(prs []domain.PullRequest) {
	sort.SliceStable(prs, func(i, j int) bool {
		if prs[i].Category != prs[j].Category {
			categoryOrder := map[domain.PRCategory]int{
				domain.PRCategoryAuthored: 0,
				domain.PRCategoryAssigned: 1,
				domain.PRCategoryOther:    2,
			}
			return categoryOrder[prs[i].Category] < categoryOrder[prs[j].Category]
		}
		return prs[i].UpdatedAt.After(prs[j].UpdatedAt)
	})

	items := make([]list.Item, len(prs))
	for i, pr := range prs {
		items[i] = PRItem{pr: pr}
	}
	m.list.SetItems(items)

	authored := 0
	assigned := 0
	other := 0
	for _, pr := range prs {
		switch pr.Category {
		case domain.PRCategoryAuthored:
			authored++
		case domain.PRCategoryAssigned:
			assigned++
		default:
			other++
		}
	}

	title := fmt.Sprintf("Pull Requests (✎ %d | → %d | ○ %d)", authored, assigned, other)
	if m.filterText != "" {
		title = fmt.Sprintf("Pull Requests [filter: %s] (✎ %d | → %d | ○ %d)", m.filterText, authored, assigned, other)
	}
	m.list.Title = title
}

func (m *PRListViewModel) SetPRGroups(groups []domain.PRGroup) {
	var allPRs []domain.PullRequest
	for _, group := range groups {
		allPRs = append(allPRs, group.PRs...)
	}

	m.allGroups = groups
	m.allPRs = allPRs
	m.prs = allPRs

	m.applyFilter()
}

func (m *PRListViewModel) applyFilter() {
	if m.allGroups != nil && len(m.allGroups) > 0 {
		m.applyFilterToGroups()
	} else {
		m.applyFilterToFlat()
	}
}

func (m *PRListViewModel) applyFilterToFlat() {
	filtered := m.filterPRs(m.allPRs)
	m.prs = filtered
	m.sortAndSetItems(filtered)
}

func (m *PRListViewModel) applyFilterToGroups() {
	var items []list.Item

	groups := m.allGroups
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].IsPrimary != groups[j].IsPrimary {
			return groups[i].IsPrimary
		}
		if groups[i].Provider != groups[j].Provider {
			return groups[i].Provider < groups[j].Provider
		}
		return groups[i].Username < groups[j].Username
	})

	var filteredPRs []domain.PullRequest
	for _, group := range groups {
		filteredGroupPRs := m.filterPRs(group.PRs)
		if len(filteredGroupPRs) == 0 {
			continue
		}

		headerText := fmt.Sprintf("%s: %s", group.Provider, group.Username)
		if group.IsPrimary {
			headerText += " ●"
		}
		items = append(items, SectionHeaderItem{text: headerText})

		prs := filteredGroupPRs
		sort.SliceStable(prs, func(i, j int) bool {
			if prs[i].Category != prs[j].Category {
				categoryOrder := map[domain.PRCategory]int{
					domain.PRCategoryAuthored: 0,
					domain.PRCategoryAssigned: 1,
					domain.PRCategoryOther:    2,
				}
				return categoryOrder[prs[i].Category] < categoryOrder[prs[j].Category]
			}
			return prs[i].UpdatedAt.After(prs[j].UpdatedAt)
		})

		for _, pr := range prs {
			items = append(items, PRItem{pr: pr})
			filteredPRs = append(filteredPRs, pr)
		}
	}

	m.list.SetItems(items)
	m.prs = filteredPRs

	authored := 0
	assigned := 0
	other := 0
	for _, pr := range filteredPRs {
		switch pr.Category {
		case domain.PRCategoryAuthored:
			authored++
		case domain.PRCategoryAssigned:
			assigned++
		default:
			other++
		}
	}

	title := fmt.Sprintf("Pull Requests (✎ %d | → %d | ○ %d)", authored, assigned, other)
	if m.filterText != "" {
		title = fmt.Sprintf("Pull Requests [filter: %s] (✎ %d | → %d | ○ %d)", m.filterText, authored, assigned, other)
	}
	m.list.Title = title
}

func (m *PRListViewModel) filterPRs(prs []domain.PullRequest) []domain.PullRequest {
	if m.filterText == "" {
		return prs
	}

	filter := strings.ToLower(m.filterText)
	var filtered []domain.PullRequest

	for _, pr := range prs {
		titleMatch := strings.Contains(strings.ToLower(pr.Title), filter)
		authorMatch := strings.Contains(strings.ToLower(pr.Author.Username), filter)
		numberMatch := strings.Contains(strconv.Itoa(pr.Number), filter)

		if titleMatch || authorMatch || numberMatch {
			filtered = append(filtered, pr)
		}
	}

	return filtered
}

func (m *PRListViewModel) GetSelectedPR() *domain.PullRequest {
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}

	if _, ok := item.(SectionHeaderItem); ok {
		return nil
	}

	prItem, ok := item.(PRItem)
	if !ok {
		return nil
	}

	return &prItem.pr
}

func (m *PRListViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return cmd
}

func (m *PRListViewModel) UpdateFilterInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
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

func (m *PRListViewModel) IsFiltering() bool {
	return m.filtering
}

func (m *PRListViewModel) ApplyFilterInput() {
	m.filterText = m.filterInput.Value()
	m.filtering = false
	m.filterInput.Blur()
	m.applyFilter()
}

func (m *PRListViewModel) ClearFilter() {
	m.filterText = ""
	m.filtering = false
	m.filterInput.SetValue("")
	m.filterInput.Blur()
	m.applyFilter()
}

func (m *PRListViewModel) GetFilterText() string {
	return m.filterText
}

func (m *PRListViewModel) View() string {
	var helpText string
	if m.filtering {
		helpText = "Enter: Apply filter | Esc: Cancel"
	} else if m.filterText != "" {
		helpText = "Enter: Inspect | r: Refresh | /: Filter | Esc: Clear filter | q: Back"
	} else {
		helpText = "Enter: Inspect | r: Refresh | /: Filter | q: Back"
	}

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Render("\n" + helpText)

	if m.filtering {
		filterStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true)
		filterLabel := filterStyle.Render("Filter: ")
		filterLine := "\n" + filterLabel + m.filterInput.View()
		return m.list.View() + filterLine + help
	}

	return m.list.View() + help
}

func formatAge(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
