package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/git"
)

type RepoLinkItem struct {
	Link       domain.RepositoryLink
	Status     *git.RepoStatus
	IsSelected bool
}

type ReposViewModel struct {
	links       []RepoLinkItem
	cursor      int
	width       int
	height      int
	active      bool
	scrollTop   int
	visibleRows int
}

func NewReposView() *ReposViewModel {
	return &ReposViewModel{
		links:       []RepoLinkItem{},
		cursor:      0,
		active:      false,
		visibleRows: 10,
	}
}

func (m *ReposViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.visibleRows = (height - 15) / 2
	if m.visibleRows < 3 {
		m.visibleRows = 3
	}
}

func (m *ReposViewModel) Activate() {
	m.active = true
	m.cursor = 0
	m.scrollTop = 0
}

func (m *ReposViewModel) Deactivate() {
	m.active = false
}

func (m *ReposViewModel) IsActive() bool {
	return m.active
}

func (m *ReposViewModel) SetLinks(links []domain.RepositoryLink) {
	m.links = make([]RepoLinkItem, len(links))
	for i, link := range links {
		status, _ := git.GetRepoStatus(link.LocalPath)
		m.links[i] = RepoLinkItem{
			Link:   link,
			Status: status,
		}
	}
}

func (m *ReposViewModel) GetSelectedLink() *domain.RepositoryLink {
	if m.cursor >= 0 && m.cursor < len(m.links) {
		return &m.links[m.cursor].Link
	}
	return nil
}

func (m *ReposViewModel) GetSelectedStatus() *git.RepoStatus {
	if m.cursor >= 0 && m.cursor < len(m.links) {
		return m.links[m.cursor].Status
	}
	return nil
}

func (m *ReposViewModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
		if m.cursor < m.scrollTop {
			m.scrollTop = m.cursor
		}
	}
}

func (m *ReposViewModel) MoveDown() {
	if m.cursor < len(m.links)-1 {
		m.cursor++
		if m.cursor >= m.scrollTop+m.visibleRows {
			m.scrollTop = m.cursor - m.visibleRows + 1
		}
	}
}

func (m *ReposViewModel) RefreshStatus() {
	for i := range m.links {
		status, _ := git.GetRepoStatus(m.links[i].Link.LocalPath)
		m.links[i].Status = status
	}
}

func (m *ReposViewModel) Update(msg tea.Msg) tea.Cmd {
	return nil
}

func (m *ReposViewModel) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Padding(1, 0)

	b.WriteString(titleStyle.Render("Linked Repositories"))
	b.WriteString("\n\n")

	if len(m.links) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true)
		b.WriteString(emptyStyle.Render("No repositories linked yet."))
		b.WriteString("\n")
		b.WriteString(emptyStyle.Render("Use :link <path> in PR view to link a repository."))
		b.WriteString("\n\n")
	} else {
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Bold(true)

		repoColWidth := 30
		pathColWidth := 40
		branchColWidth := 20
		statusColWidth := 15

		header := fmt.Sprintf("  %-*s %-*s %-*s %s",
			repoColWidth, "REPOSITORY",
			pathColWidth, "LOCAL PATH",
			branchColWidth, "BRANCH",
			"STATUS")
		b.WriteString(headerStyle.Render(header))
		b.WriteString("\n")

		divider := strings.Repeat("─", repoColWidth+pathColWidth+branchColWidth+statusColWidth+10)
		dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#374151"))
		b.WriteString(dividerStyle.Render(divider))
		b.WriteString("\n")

		endIdx := m.scrollTop + m.visibleRows
		if endIdx > len(m.links) {
			endIdx = len(m.links)
		}

		for i := m.scrollTop; i < endIdx; i++ {
			item := m.links[i]
			isSelected := i == m.cursor

			repoName := truncateString(item.Link.RepoFullName, repoColWidth-1)
			localPath := truncateString(item.Link.LocalPath, pathColWidth-1)

			branch := "-"
			statusText := ""
			statusColor := "#6B7280"

			if item.Status != nil {
				if item.Status.IsGitRepo {
					branch = item.Status.CurrentBranch
					if item.Status.HasUncommittedChanges {
						statusText = "dirty"
						statusColor = "#F59E0B"
					} else {
						statusText = "clean"
						statusColor = "#10B981"
					}
				} else {
					statusText = "not a repo"
					statusColor = "#EF4444"
				}
			} else {
				statusText = "unknown"
			}

			branch = truncateString(branch, branchColWidth-1)

			cursor := "  "
			if isSelected {
				cursor = "▸ "
			}

			var rowStyle lipgloss.Style
			if isSelected {
				rowStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#7C3AED")).
					Bold(true)
			} else {
				rowStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#D1D5DB"))
			}

			statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))

			row := fmt.Sprintf("%s%-*s %-*s %-*s",
				cursor,
				repoColWidth, repoName,
				pathColWidth, localPath,
				branchColWidth, branch)

			b.WriteString(rowStyle.Render(row))
			b.WriteString(statusStyle.Render(statusText))
			b.WriteString("\n")
		}

		if len(m.links) > m.visibleRows {
			scrollInfo := fmt.Sprintf("\n[%d-%d of %d]", m.scrollTop+1, endIdx, len(m.links))
			scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
			b.WriteString(scrollStyle.Render(scrollInfo))
		}
	}

	b.WriteString("\n\n")
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)

	help := "j/k: Navigate | o: Open folder | d: Delete link | r: Refresh | Esc: Close"
	b.WriteString(helpStyle.Render(help))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(1, 2).
		Width(m.width - 4)

	return boxStyle.Render(b.String())
}
