package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type TopBarModel struct {
	width          int
	totalPRs       int
	authoredPRs    int
	assignedPRs    int
	otherPRs       int
	repoCount      int
	currentRepo    string
	currentPR      string
	prStatus       string
	prMergeable    bool
	prApproval     string
	activePAT      string
	patProvider    string
	selectedCount  int
	totalPATCount  int
	currentView    string
	shortcuts      []string
}

var (
	titleStyle       = lipgloss.NewStyle().Padding(1, 2)
	titleOrangeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	valueWhiteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	shortcutBlueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
	descGrayStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
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

func (m *TopBarModel) SetPRStatus(status string, mergeable bool) {
	m.prStatus = status
	m.prMergeable = mergeable
}

func (m *TopBarModel) SetPRApproval(approval string) {
	m.prApproval = approval
}

func (m *TopBarModel) SetActivePAT(pat, provider string) {
	m.activePAT = pat
	m.patProvider = provider
}

func (m *TopBarModel) SetSelectedPATCount(count int) {
	m.selectedCount = count
}

func (m *TopBarModel) SetPATCounts(selected, total int) {
	m.selectedCount = selected
	m.totalPATCount = total
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

	const fixedRows = 5

	const contextColWidth = 45
	const colMargin = 4

	for i := 0; i < fixedRows; i++ {
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
		if m.selectedCount > 1 {
			patName = fmt.Sprintf("%s + %d more", patName, m.selectedCount-1)
		}
		if len(patName) > 35 {
			patName = patName[:32] + "..."
		}
	}

	patLine := patEmoji + " " + titleOrangeStyle.Render("PAT: ") + valueWhiteStyle.Render(patName)
	if m.totalPATCount > 0 {
		countInfo := fmt.Sprintf(" [%d/%d]", m.selectedCount, m.totalPATCount)
		patLine += descGrayStyle.Render(countInfo)
	}
	lines = append(lines, patLine)

	isPRView := m.currentView == "PR Description" || m.currentView == "PR Diff" || m.currentView == "PR Inspect"

	if isPRView && m.currentRepo != "" {
		repoEmoji := "📦"
		lines = append(lines,
			repoEmoji+" "+
				titleOrangeStyle.Render("Repo: ")+
				valueWhiteStyle.Render(m.currentRepo))

		if m.currentPR != "" {
			prEmoji := "📋"
			prValue := fmt.Sprintf("#%s", m.currentPR)

			if m.prStatus != "" {
				var statusColor lipgloss.Color
				var statusText string
				var mergeIcon string

				switch m.prStatus {
				case "merged":
					statusColor = lipgloss.Color("10")
					statusText = "MERGED"
					mergeIcon = "✓"
				case "open":
					statusColor = lipgloss.Color("214")
					statusText = "OPEN"
					if m.prMergeable {
						mergeIcon = "✓"
					} else {
						mergeIcon = "✗"
					}
				case "closed":
					statusColor = lipgloss.Color("8")
					statusText = "CLOSED"
					mergeIcon = ""
				}

				statusStyle := lipgloss.NewStyle().Foreground(statusColor).Bold(true)
				statusBadge := statusStyle.Render(fmt.Sprintf("[%s %s]", statusText, mergeIcon))
				prValue = fmt.Sprintf("%s %s", prValue, statusBadge)
			}

			if m.prApproval != "" {
				var approvalColor lipgloss.Color
				var approvalText string

				switch m.prApproval {
				case "approved":
					approvalColor = lipgloss.Color("10")
					approvalText = "APPROVED ✓"
				case "changes_requested":
					approvalColor = lipgloss.Color("9")
					approvalText = "CHANGES ✗"
				case "pending":
					approvalColor = lipgloss.Color("214")
					approvalText = "PENDING ◯"
				}

				if approvalText != "" {
					approvalStyle := lipgloss.NewStyle().Foreground(approvalColor).Bold(true)
					approvalBadge := approvalStyle.Render(fmt.Sprintf("[%s]", approvalText))
					prValue = fmt.Sprintf("%s %s", prValue, approvalBadge)
				}
			}

			lines = append(lines,
				prEmoji+" "+
					titleOrangeStyle.Render("PR: ")+
					valueWhiteStyle.Render(prValue))
		}
	} else {
		lines = append(lines,
			"❤️ "+
				titleOrangeStyle.Render("your: ")+
				valueWhiteStyle.Render(fmt.Sprintf("%d", m.authoredPRs)))

		lines = append(lines,
			"👀 "+
				titleOrangeStyle.Render("assigned: ")+
				valueWhiteStyle.Render(fmt.Sprintf("%d", m.assignedPRs)))

		lines = append(lines,
			"⏳ "+
				titleOrangeStyle.Render("pending: ")+
				valueWhiteStyle.Render(fmt.Sprintf("%d", m.otherPRs)))
	}

	viewEmoji := "🎯"
	viewName := m.currentView
	if viewName == "" {
		viewName = "PATs"
	}
	lines = append(lines,
		viewEmoji+" "+
			titleOrangeStyle.Render("View: ")+
			valueWhiteStyle.Render(viewName))

	const minContextLines = 5
	for len(lines) < minContextLines {
		lines = append(lines, "")
	}

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

	minRows := 5
	if contextHeight > minRows {
		minRows = contextHeight
	}

	var col1, col2 []string

	if len(formattedShortcuts) <= minRows {
		col1 = formattedShortcuts
	} else {
		splitPoint := minRows
		col1 = formattedShortcuts[:splitPoint]
		col2 = formattedShortcuts[splitPoint:]
	}

	return col1, col2, maxWidth
}
