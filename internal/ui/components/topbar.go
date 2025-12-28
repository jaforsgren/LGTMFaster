package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type TopBarModel struct {
	width       int
	totalPRs    int
	repoCount   int
	currentRepo string
	currentPR   string
	activePAT   string
}

func NewTopBar() *TopBarModel {
	return &TopBarModel{}
}

func (m *TopBarModel) SetWidth(width int) {
	m.width = width
}

func (m *TopBarModel) SetStats(totalPRs, repoCount int) {
	m.totalPRs = totalPRs
	m.repoCount = repoCount
}

func (m *TopBarModel) SetContext(repo, pr string) {
	m.currentRepo = repo
	m.currentPR = pr
}

func (m *TopBarModel) SetActivePAT(pat string) {
	m.activePAT = pat
}

func (m *TopBarModel) View() string {
	leftSection := m.renderLeftSection()
	rightSection := m.renderRightSection()

	availableWidth := m.width - lipgloss.Width(leftSection) - lipgloss.Width(rightSection)
	if availableWidth < 0 {
		availableWidth = 0
	}

	middle := strings.Repeat(" ", availableWidth)

	content := leftSection + middle + rightSection

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F9FAFB")).
		Background(lipgloss.Color("#7C3AED")).
		Bold(true).
		Width(m.width)

	return style.Render(content)
}

func (m *TopBarModel) renderLeftSection() string {
	parts := []string{"LGTMFaster"}

	if m.currentRepo != "" {
		parts = append(parts, fmt.Sprintf("| %s", m.currentRepo))
	}

	if m.currentPR != "" {
		parts = append(parts, fmt.Sprintf("#%s", m.currentPR))
	}

	return " " + strings.Join(parts, " ") + " "
}

func (m *TopBarModel) renderRightSection() string {
	parts := []string{}

	if m.activePAT != "" {
		parts = append(parts, fmt.Sprintf("PAT: %s", m.activePAT))
	}

	if m.repoCount > 0 {
		parts = append(parts, fmt.Sprintf("Repos: %d", m.repoCount))
	}

	if m.totalPRs > 0 {
		parts = append(parts, fmt.Sprintf("PRs: %d", m.totalPRs))
	}

	if len(parts) == 0 {
		return ""
	}

	return " " + strings.Join(parts, " | ") + " "
}
