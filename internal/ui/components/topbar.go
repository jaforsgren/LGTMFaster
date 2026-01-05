package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type TopBarModel struct {
	width       int
	totalPRs    int
	authoredPRs int
	assignedPRs int
	otherPRs    int
	repoCount   int
	currentRepo string
	currentPR   string
	activePAT   string
	patProvider string
	currentView string
	shortcuts   []string
	test        bool
}

var (
	titleStyle        = lipgloss.NewStyle().Padding(1, 2)
	titleOrangeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	valueWhiteStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	shortcutBlueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
	descGrayStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
)

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

func (m *TopBarModel) SetPRBreakdown(authored, assigned, other int) {
	m.authoredPRs = authored
	m.assignedPRs = assigned
	m.otherPRs = other
}

func (m *TopBarModel) SetContext(repo, pr string) {
	m.currentRepo = repo
	m.currentPR = pr
}

func (m *TopBarModel) SetActivePAT(pat, provider string) {
	m.activePAT = pat
	m.patProvider = provider
}

func (m *TopBarModel) SetView(view string) {
	m.currentView = view
}

func (m *TopBarModel) SetShortcuts(shortcuts []string) {
	m.shortcuts = shortcuts
}

func (m *TopBarModel) View() string {
	titleLine := titleOrangeStyle.Render("LGTMFaster")

	contextLines := m.buildContextInfo()
	shortcutCol1, shortcutCol2, col1Width := m.buildShortcutsDisplay(len(contextLines))

	var topSection []string
	topSection = append(topSection, titleLine)
	topSection = append(topSection, "")

	maxLines := len(contextLines)
	if len(shortcutCol1) > maxLines {
		maxLines = len(shortcutCol1)
	}

	const contextColWidth = 45
	const colMargin = 4

	for i := 0; i < maxLines; i++ {
		var contextCol, sc1, sc2 string

		if i < len(contextLines) {
			contextCol = contextLines[i]
		}

		if i < len(shortcutCol1) {
			sc1 = shortcutCol1[i]
		}

		if i < len(shortcutCol2) {
			sc2 = shortcutCol2[i]
		}

		contextVisible := lipgloss.Width(contextCol)
		padding1 := contextColWidth - contextVisible
		if padding1 < 0 {
			padding1 = 1
		}

		line := contextCol + strings.Repeat(" ", padding1) + sc1

		if sc2 != "" {
			sc1Visible := lipgloss.Width(sc1)
			padding2 := col1Width - sc1Visible + colMargin
			if padding2 < colMargin {
				padding2 = colMargin
			}
			line += strings.Repeat(" ", padding2) + sc2
		}

		topSection = append(topSection, line)
	}

	content := strings.Join(topSection, "\n")
	return titleStyle.Width(m.width).Render(content)
}

func (m *TopBarModel) buildContextInfo() []string {
	var lines []string

	patName := "none"
	patEmoji := "🔑"
	if m.activePAT != "" {
		patName = m.activePAT
		if m.patProvider != "" {
			patName = fmt.Sprintf("%s (%s)", patName, m.patProvider)
		}
		if len(patName) > 25 {
			patName = patName[:22] + "..."
		}
	}
	lines = append(lines,
		patEmoji+" "+
			titleOrangeStyle.Render("PAT: ")+
			valueWhiteStyle.Render(patName))

	prEmoji := "📋"
	if m.totalPRs > 0 {
		prBreakdown := fmt.Sprintf("%d (✎%d →%d ○%d)",
			m.totalPRs, m.authoredPRs, m.assignedPRs, m.otherPRs)
		lines = append(lines,
			prEmoji+" "+
				titleOrangeStyle.Render("PRs: ")+
				valueWhiteStyle.Render(prBreakdown))
	} else {
		lines = append(lines,
			prEmoji+" "+
				titleOrangeStyle.Render("PRs: ")+
				valueWhiteStyle.Render("0"))
	}

	repoEmoji := "📦"
	lines = append(lines,
		repoEmoji+" "+
			titleOrangeStyle.Render("Repositories: ")+
			valueWhiteStyle.Render(fmt.Sprintf("%d", m.repoCount)))

	contextEmoji := "📍"
	contextValue := "none"
	if m.currentRepo != "" {
		contextValue = m.currentRepo
		if m.currentPR != "" {
			contextValue = fmt.Sprintf("%s #%s", m.currentRepo, m.currentPR)
		}
		if len(contextValue) > 25 {
			contextValue = contextValue[:22] + "..."
		}
	}
	lines = append(lines,
		contextEmoji+" "+
			titleOrangeStyle.Render("Context: ")+
			valueWhiteStyle.Render(contextValue))

	viewEmoji := "🎯"
	viewName := m.currentView
	if viewName == "" {
		viewName = "PATs"
	}
	lines = append(lines,
		viewEmoji+" "+
			titleOrangeStyle.Render("View: ")+
			valueWhiteStyle.Render(viewName))

	return lines
}

func (m *TopBarModel) buildShortcutsDisplay(contextHeight int) ([]string, []string, int) {
	var formattedShortcuts []string
	maxWidth := 0

	for _, shortcut := range m.shortcuts {
		parts := strings.SplitN(shortcut, ">", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimPrefix(parts[0], "<")
		desc := strings.TrimSpace(parts[1])

		formatted := shortcutBlueStyle.Render("<"+key+">") + " " + descGrayStyle.Render(desc)
		formattedShortcuts = append(formattedShortcuts, formatted)

		width := lipgloss.Width(formatted)
		if width > maxWidth {
			maxWidth = width
		}
	}

	var col1, col2 []string

	if len(formattedShortcuts) <= contextHeight {
		col1 = formattedShortcuts
	} else {
		splitPoint := contextHeight
		col1 = formattedShortcuts[:splitPoint]
		col2 = formattedShortcuts[splitPoint:]
	}

	return col1, col2, maxWidth
}
