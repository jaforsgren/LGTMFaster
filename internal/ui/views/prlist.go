package views

import (
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/list"
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

	title := fmt.Sprintf("%s %s", categoryIndicator, i.pr.Title)
	return style.Render(title)
}

func (i PRItem) Description() string {
	repo := i.pr.Repository.FullName
	number := fmt.Sprintf("#%d", i.pr.Number)
	author := i.pr.Author.Username
	age := formatAge(i.pr.CreatedAt)

	return fmt.Sprintf("%s %s by %s • %s", repo, number, author, age)
}

type PRListViewModel struct {
	list   list.Model
	prs    []domain.PullRequest
	width  int
	height int
}

func NewPRListView() *PRListViewModel {
	items := []list.Item{}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Pull Requests"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return &PRListViewModel{
		list: l,
		prs:  []domain.PullRequest{},
	}
}

func (m *PRListViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height-5)
}

func (m *PRListViewModel) SetPRs(prs []domain.PullRequest) {
	m.prs = prs

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

	m.list.Title = fmt.Sprintf("Pull Requests (✎ %d | → %d | ○ %d)", authored, assigned, other)
}

func (m *PRListViewModel) SetPRGroups(groups []domain.PRGroup) {
	var items []list.Item

	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].IsPrimary != groups[j].IsPrimary {
			return groups[i].IsPrimary
		}
		if groups[i].Provider != groups[j].Provider {
			return groups[i].Provider < groups[j].Provider
		}
		return groups[i].Username < groups[j].Username
	})

	var allPRs []domain.PullRequest
	for _, group := range groups {
		headerText := fmt.Sprintf("%s: %s", group.Provider, group.Username)
		if group.IsPrimary {
			headerText += " ●"
		}
		items = append(items, SectionHeaderItem{text: headerText})

		prs := group.PRs
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
			allPRs = append(allPRs, pr)
		}
	}

	m.list.SetItems(items)
	m.prs = allPRs

	authored := 0
	assigned := 0
	other := 0
	for _, pr := range allPRs {
		switch pr.Category {
		case domain.PRCategoryAuthored:
			authored++
		case domain.PRCategoryAssigned:
			assigned++
		default:
			other++
		}
	}

	m.list.Title = fmt.Sprintf("Pull Requests (✎ %d | → %d | ○ %d)", authored, assigned, other)
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

func (m *PRListViewModel) GetCursorIndex() int {
	return m.list.Cursor()
}

func (m *PRListViewModel) RestoreCursor(index int) {
	items := m.list.Items()
	if len(items) == 0 {
		return
	}

	if index >= len(items) {
		index = len(items) - 1
	}
	if index < 0 {
		index = 0
	}

	if _, ok := items[index].(SectionHeaderItem); ok {
		if index+1 < len(items) {
			index++
		}
	}

	m.list.Select(index)
}

func (m *PRListViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return cmd
}

func (m *PRListViewModel) View() string {
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Render("\nEnter: Inspect | r: Refresh | /: Filter | q: Back")

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
