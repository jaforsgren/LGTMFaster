package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	ViewNamePipelines    = "Pipelines"
	ViewNamePipelineRuns = "Pipeline Runs"
	ViewNamePipelineRun  = "Pipeline Run"
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
	linkedRepoPath string
	linkedBranch   string
	pipelineRepo   string
	pipelineName   string
	pipelineRun    string
	pipelineCount  int
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

func (m *TopBarModel) SetLinkedRepo(path, branch string) {
	m.linkedRepoPath = path
	m.linkedBranch = branch
}

func (m *TopBarModel) ClearLinkedRepo() {
	m.linkedRepoPath = ""
	m.linkedBranch = ""
}

func (m *TopBarModel) SetPipelineContext(repo, pipeline, run string) {
	m.pipelineRepo = repo
	m.pipelineName = pipeline
	m.pipelineRun = run
}

func (m *TopBarModel) SetPipelineCount(count int) {
	m.pipelineCount = count
}

func (m *TopBarModel) HasLinkedRepo() bool {
	return m.linkedRepoPath != ""
}

const (
	topBarRows            = 5
	topBarContextColWidth = 45
	topBarColumnMargin    = 4
	topBarHorizontalPad   = 4
)

func (m *TopBarModel) View() string {
	titleLine := titleOrangeStyle.Render("LGTMFaster")

	contextLines := m.buildContextInfo()
	availableWidth := m.width - topBarHorizontalPad - topBarContextColWidth
	columns := layoutShortcutColumns(m.formattedShortcuts(), topBarRows, availableWidth)

	topSection := []string{titleLine, ""}

	for i := 0; i < topBarRows; i++ {
		var contextCol string
		if i < len(contextLines) {
			contextCol = contextLines[i]
		}

		line := contextCol + strings.Repeat(" ", max(1, topBarContextColWidth-lipgloss.Width(contextCol)))
		for c, column := range columns {
			if i >= len(column.cells) {
				continue
			}
			cell := column.cells[i]
			line += cell
			if c < len(columns)-1 {
				line += strings.Repeat(" ", column.width-lipgloss.Width(cell)+topBarColumnMargin)
			}
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
	isPipelineView := m.currentView == ViewNamePipelines || m.currentView == ViewNamePipelineRuns || m.currentView == ViewNamePipelineRun

	if isPipelineView {
		lines = append(lines, m.buildPipelineContextInfo()...)
	} else if isPRView && m.currentRepo != "" {
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

		if m.linkedRepoPath != "" {
			linkEmoji := "🔗"
			linkValue := m.linkedRepoPath
			if len(linkValue) > 35 {
				linkValue = "..." + linkValue[len(linkValue)-32:]
			}
			if m.linkedBranch != "" {
				branchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
				linkValue = linkValue + " " + branchStyle.Render("["+m.linkedBranch+"]")
			}
			lines = append(lines,
				linkEmoji+" "+
					titleOrangeStyle.Render("Local: ")+
					valueWhiteStyle.Render(linkValue))
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

func (m *TopBarModel) buildPipelineContextInfo() []string {
	if m.pipelineName == "" {
		return []string{
			"🚀 " + titleOrangeStyle.Render("Pipelines: ") + valueWhiteStyle.Render(fmt.Sprintf("%d", m.pipelineCount)),
		}
	}

	lines := []string{
		"📦 " + titleOrangeStyle.Render("Repo: ") + valueWhiteStyle.Render(m.pipelineRepo),
		"🚀 " + titleOrangeStyle.Render("Pipeline: ") + valueWhiteStyle.Render(m.pipelineName),
	}
	if m.pipelineRun != "" {
		lines = append(lines, "▶️ "+titleOrangeStyle.Render("Run: ")+valueWhiteStyle.Render(m.pipelineRun))
	}
	return lines
}

func (m *TopBarModel) formattedShortcuts() []string {
	var formatted []string
	for _, shortcut := range m.shortcuts {
		parts := strings.SplitN(shortcut, ">", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimPrefix(parts[0], "<")
		desc := strings.TrimSpace(parts[1])
		formatted = append(formatted, shortcutBlueStyle.Render("<"+key+">")+" "+descGrayStyle.Render(desc))
	}
	return formatted
}

type shortcutColumn struct {
	cells []string
	width int
}

// layoutShortcutColumns fills columns top to bottom, adding columns until the shortcuts run out
// or the next column would not fit, so no hint is wrapped into the content below the top bar.
func layoutShortcutColumns(shortcuts []string, rows, availableWidth int) []shortcutColumn {
	var columns []shortcutColumn
	usedWidth := 0

	for start := 0; start < len(shortcuts); start += rows {
		end := min(start+rows, len(shortcuts))
		column := shortcutColumn{cells: shortcuts[start:end]}
		for _, cell := range column.cells {
			column.width = max(column.width, lipgloss.Width(cell))
		}

		required := column.width
		if len(columns) > 0 {
			required += topBarColumnMargin
		}
		if len(columns) > 0 && usedWidth+required > availableWidth {
			break
		}

		usedWidth += required
		columns = append(columns, column)
	}
	return columns
}
