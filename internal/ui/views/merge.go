package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type MergeViewModel struct {
	active      bool
	width       int
	height      int
	selectedIdx int
	options     []MergeOption
	pr          *domain.PullRequest
	provider    domain.ProviderType
}

type MergeOption struct {
	method      string
	label       string
	description string
}

func NewMergeView() *MergeViewModel {
	return &MergeViewModel{
		active:      false,
		selectedIdx: 0,
	}
}

func (m *MergeViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *MergeViewModel) Activate(pr *domain.PullRequest, provider domain.ProviderType) {
	m.active = true
	m.pr = pr
	m.provider = provider
	m.selectedIdx = 0
	m.options = m.buildOptions()
}

func (m *MergeViewModel) Deactivate() {
	m.active = false
	m.pr = nil
	m.selectedIdx = 0
	m.options = nil
}

func (m *MergeViewModel) IsActive() bool {
	return m.active
}

func (m *MergeViewModel) GetSelectedMethod() string {
	if m.selectedIdx >= 0 && m.selectedIdx < len(m.options) {
		return m.options[m.selectedIdx].method
	}
	return ""
}

func (m *MergeViewModel) GetPR() *domain.PullRequest {
	return m.pr
}

func (m *MergeViewModel) NextOption() {
	if m.selectedIdx < len(m.options)-1 {
		m.selectedIdx++
	}
}

func (m *MergeViewModel) PrevOption() {
	if m.selectedIdx > 0 {
		m.selectedIdx--
	}
}

func (m *MergeViewModel) Update(msg tea.Msg) tea.Cmd {
	return nil
}

func (m *MergeViewModel) buildOptions() []MergeOption {
	if m.provider == domain.ProviderGitHub {
		return []MergeOption{
			{
				method:      "merge",
				label:       "Merge commit",
				description: "Create a merge commit (preserves all commits)",
			},
			{
				method:      "squash",
				label:       "Squash and merge",
				description: "Combine all commits into one",
			},
			{
				method:      "rebase",
				label:       "Rebase and merge",
				description: "Rebase commits onto target branch",
			},
		}
	} else if m.provider == domain.ProviderAzureDevOps {
		return []MergeOption{
			{
				method:      "noFastForward",
				label:       "Merge (no fast-forward)",
				description: "Standard merge with merge commit",
			},
			{
				method:      "squash",
				label:       "Squash commit",
				description: "Combine all commits into one",
			},
			{
				method:      "rebase",
				label:       "Rebase and fast-forward",
				description: "Rebase commits onto target branch",
			},
		}
	}
	return []MergeOption{}
}

func (m *MergeViewModel) View() string {
	if !m.active || m.pr == nil {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Padding(1, 0)

	b.WriteString(titleStyle.Render("Merge Pull Request"))
	b.WriteString("\n\n")

	prInfoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15"))

	b.WriteString(prInfoStyle.Render(fmt.Sprintf("Title: %s", m.pr.Title)))
	b.WriteString("\n")
	b.WriteString(prInfoStyle.Render(fmt.Sprintf("Branch: %s → %s", m.pr.SourceBranch, m.pr.TargetBranch)))
	b.WriteString("\n\n")

	if !m.pr.Mergeable {
		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

		b.WriteString(warningStyle.Render("⚠ Warning: This PR has merge conflicts"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("246")).Render("Resolve conflicts before merging"))
		b.WriteString("\n\n")
	} else {
		successStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

		b.WriteString(successStyle.Render("✓ This PR is mergeable"))
		b.WriteString("\n\n")
	}

	mergeMethodTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Bold(true)

	b.WriteString(mergeMethodTitle.Render("Select merge method:"))
	b.WriteString("\n\n")

	for i, option := range m.options {
		selected := i == m.selectedIdx
		var optionStyle lipgloss.Style

		if selected {
			optionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)
		} else {
			optionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15"))
		}

		marker := " "
		if selected {
			marker = "●"
		} else {
			marker = "○"
		}

		b.WriteString(optionStyle.Render(fmt.Sprintf(" %s %s", marker, option.label)))
		b.WriteString("\n")

		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")).
			PaddingLeft(4)

		b.WriteString(descStyle.Render(option.description))
		b.WriteString("\n\n")
	}

	noteStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("246")).
		Italic(true)

	b.WriteString(noteStyle.Render("Note: Source branch will be deleted after merge"))
	b.WriteString("\n\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)

	help := "↑↓: Navigate | Enter: Confirm | Esc: Cancel"
	b.WriteString(helpStyle.Render(help))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(1, 2).
		Width(min(80, m.width-4))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxStyle.Render(b.String()))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
