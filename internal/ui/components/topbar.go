package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	ViewNamePATs          = "PATs"
	ViewNamePRList        = "PR List"
	ViewNamePRDescription = "PR Description"
	ViewNamePRDiff        = "PR Diff"
	ViewNamePRInspect     = "PR Inspect"
	ViewNamePipelines     = "Pipelines"
	ViewNamePipelineRuns  = "Pipeline Runs"
	ViewNamePipelineRun   = "Pipeline Run"
)

const (
	topBarSectionRows = 6 // rows under each section heading
	topBarTitleRows   = 2 // the title and the blank line under it
	topBarHeadingRows = 1
	topBarPaddingRows = 2 // titleStyle's vertical padding
	topBarPaddingCols = 2 // titleStyle's horizontal padding

	// TopBarHeight is the rendered height; View truncates lines so they never wrap and grow it.
	TopBarHeight = topBarTitleRows + topBarHeadingRows + topBarSectionRows + topBarPaddingRows

	topBarTitle           = "LGTMFaster"
	topBarNameMax         = 25
	topBarPathMax         = 35
	topBarSeparator       = " │ "
	topBarShortcutsGap    = 3
	accountActiveMarker   = "● "
	accountPassiveMarker  = "○ "
	truncationTail        = "..."
	lastRefreshTimeFormat = "15:04:05"
)

var (
	titleStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Background(lipgloss.Color("#000000")).Padding(1, 1)
	titleOrangeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	labelOrangeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	valueWhiteStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	dimStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	activeStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	autoRefreshOffStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	shortcutBlueStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
	descGrayStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	topBarHeadingStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	topBarSeparatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

// Account is a PAT as the top bar lists it.
type Account struct {
	Name     string
	Provider string
	Selected bool
	Primary  bool
}

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
	accounts       []Account
	currentView    string
	shortcuts      []string
	linkedRepoPath string
	linkedBranch   string
	pipelineRepo   string
	pipelineName   string
	pipelineRun    string
	pipelineCount  int
	autoRefresh    bool
	refreshEvery   time.Duration
	lastRefresh    time.Time
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

func (m *TopBarModel) SetAccounts(accounts []Account) {
	m.accounts = accounts
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

func (m *TopBarModel) SetAutoRefresh(enabled bool, interval time.Duration) {
	m.autoRefresh = enabled
	m.refreshEvery = interval
}

// SetLastRefresh records when pipeline data last arrived, which shows that auto-refresh is working.
func (m *TopBarModel) SetLastRefresh(at time.Time) {
	m.lastRefresh = at
}

func (m *TopBarModel) HasLinkedRepo() bool {
	return m.linkedRepoPath != ""
}

// View lays the context, the accounts and the shortcuts out as headed sections divided by
// vertical rules.
func (m *TopBarModel) View() string {
	sections := [][]string{
		topBarSection("CONTEXT", m.buildContextInfo()),
		topBarSection(m.accountsHeading(), m.buildAccountsSection()),
	}
	used := topBarPaddingCols
	for _, section := range sections {
		used += columnWidth(section) + lipgloss.Width(topBarSeparator)
	}
	sections = append(sections, topBarSection("SHORTCUTS", m.buildShortcutsDisplay(m.width-used)))

	lines := strings.Split(titleOrangeStyle.Render(topBarTitle)+"\n\n"+joinTopBarSections(sections), "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], max(m.width-topBarPaddingCols, 0), "")
	}
	return titleStyle.Width(m.width).Render(strings.Join(lines, "\n"))
}

func topBarSection(heading string, lines []string) []string {
	if len(lines) > topBarSectionRows {
		lines = lines[:topBarSectionRows]
	}
	return append([]string{topBarHeadingStyle.Render(heading)}, lines...)
}

func columnWidth(lines []string) int {
	widest := 0
	for _, line := range lines {
		widest = max(widest, lipgloss.Width(line))
	}
	return widest
}

// joinTopBarSections pads every section to its widest line and puts a vertical rule between them
// on every row.
func joinTopBarSections(sections [][]string) string {
	// Every row is emitted, even when all sections are short, to keep the height at TopBarHeight.
	rows := topBarHeadingRows + topBarSectionRows

	separator := topBarSeparatorStyle.Render(topBarSeparator)
	lines := make([]string, rows)
	for i := range lines {
		var cells []string
		for s, section := range sections {
			cell := ""
			if i < len(section) {
				cell = section[i]
			}
			if s < len(sections)-1 {
				cell += strings.Repeat(" ", columnWidth(section)-lipgloss.Width(cell))
			}
			cells = append(cells, cell)
		}
		lines[i] = strings.TrimRight(strings.Join(cells, separator), " ")
	}
	return strings.Join(lines, "\n")
}

func truncateName(name string, maxLength int) string {
	return ansi.Truncate(name, maxLength, truncationTail)
}

// truncatePathStart keeps the end of a path, which is the part that tells repositories apart.
func truncatePathStart(path string, maxLength int) string {
	if ansi.StringWidth(path) <= maxLength {
		return path
	}
	return truncationTail + ansi.TruncateLeft(path, ansi.StringWidth(path)-maxLength+len(truncationTail), "")
}

func contextLine(emoji, label, value string) string {
	return emoji + " " + labelOrangeStyle.Render(label+": ") + valueWhiteStyle.Render(value)
}

func (m *TopBarModel) isPRView() bool {
	return m.currentView == ViewNamePRDescription || m.currentView == ViewNamePRDiff || m.currentView == ViewNamePRInspect
}

func (m *TopBarModel) isPipelineView() bool {
	return m.currentView == ViewNamePipelines || m.currentView == ViewNamePipelineRuns || m.currentView == ViewNamePipelineRun
}

func (m *TopBarModel) buildContextInfo() []string {
	viewName := m.currentView
	if viewName == "" {
		viewName = ViewNamePATs
	}
	lines := []string{contextLine("📍", "Mode", viewName)}

	switch {
	case m.isPipelineView():
		lines = append(lines, m.buildPipelineContextInfo()...)
	case m.isPRView() && m.currentRepo != "":
		lines = append(lines, m.buildPRContextInfo()...)
	case m.currentView != ViewNamePATs:
		lines = append(lines,
			contextLine("❤️", "Yours", fmt.Sprintf("%d", m.authoredPRs)),
			contextLine("👀", "Assigned", fmt.Sprintf("%d", m.assignedPRs)),
			contextLine("⏳", "Pending", fmt.Sprintf("%d", m.otherPRs)),
			contextLine("📦", "Repos", fmt.Sprintf("%d", m.repoCount)),
		)
	}
	return lines
}

func (m *TopBarModel) buildPRContextInfo() []string {
	lines := []string{contextLine("📦", "Repo", m.currentRepo)}

	if m.currentPR != "" {
		prValue := valueWhiteStyle.Render("#" + m.currentPR)
		if badge := m.statusBadge(); badge != "" {
			prValue += " " + badge
		}
		if badge := m.approvalBadge(); badge != "" {
			prValue += " " + badge
		}
		lines = append(lines, "📋 "+labelOrangeStyle.Render("PR: ")+prValue)
	}

	if m.linkedRepoPath != "" {
		linkValue := valueWhiteStyle.Render(truncatePathStart(m.linkedRepoPath, topBarPathMax))
		if m.linkedBranch != "" {
			linkValue += " " + shortcutBlueStyle.Render("["+m.linkedBranch+"]")
		}
		lines = append(lines, "🔗 "+labelOrangeStyle.Render("Local: ")+linkValue)
	}
	return lines
}

type badgeAppearance struct {
	text  string
	color lipgloss.Color
}

var prStatusBadges = map[string]badgeAppearance{
	"merged": {"MERGED ✓", lipgloss.Color("10")},
	"closed": {"CLOSED", lipgloss.Color("8")},
}

var prApprovalBadges = map[string]badgeAppearance{
	"approved":          {"APPROVED ✓", lipgloss.Color("10")},
	"changes_requested": {"CHANGES ✗", lipgloss.Color("9")},
	"pending":           {"PENDING ◯", lipgloss.Color("214")},
}

const prStatusOpen = "open"

func (m *TopBarModel) statusBadge() string {
	appearance, ok := prStatusBadges[m.prStatus]
	if m.prStatus == prStatusOpen {
		mergeIcon := "✗"
		if m.prMergeable {
			mergeIcon = "✓"
		}
		appearance, ok = badgeAppearance{"OPEN " + mergeIcon, lipgloss.Color("214")}, true
	}
	if !ok {
		return ""
	}
	return renderBadge(appearance)
}

func (m *TopBarModel) approvalBadge() string {
	appearance, ok := prApprovalBadges[m.prApproval]
	if !ok {
		return ""
	}
	return renderBadge(appearance)
}

func renderBadge(appearance badgeAppearance) string {
	return lipgloss.NewStyle().Foreground(appearance.color).Bold(true).Render("[" + appearance.text + "]")
}

func (m *TopBarModel) buildPipelineContextInfo() []string {
	if m.pipelineName == "" {
		return []string{
			contextLine("🚀", "Pipelines", fmt.Sprintf("%d", m.pipelineCount)),
			m.autoRefreshLine(),
		}
	}

	lines := []string{
		contextLine("📦", "Repo", m.pipelineRepo),
		contextLine("🚀", "Pipeline", m.pipelineName),
	}
	if m.pipelineRun != "" {
		lines = append(lines, contextLine("▶️", "Run", m.pipelineRun))
	}
	return append(lines, m.autoRefreshLine())
}

func (m *TopBarModel) autoRefreshLine() string {
	if !m.autoRefresh {
		return "⏸️ " + labelOrangeStyle.Render("Auto-refresh: ") + autoRefreshOffStyle.Render("OFF")
	}
	line := "🔄 " + labelOrangeStyle.Render("Auto-refresh: ") + activeStyle.Render(fmt.Sprintf("ON every %s", m.refreshEvery))
	if !m.lastRefresh.IsZero() {
		line += dimStyle.Render(" · refreshed " + m.lastRefresh.Format(lastRefreshTimeFormat))
	}
	return line
}

func (m *TopBarModel) accountsHeading() string {
	if m.totalPATCount == 0 {
		return "ACCOUNTS"
	}
	return fmt.Sprintf("ACCOUNTS %d/%d", m.selectedCount, m.totalPATCount)
}

// buildAccountsSection lists every PAT on its own row, marking the selected ones, with a
// "+N more" row when they don't all fit.
func (m *TopBarModel) buildAccountsSection() []string {
	if len(m.accounts) == 0 {
		return []string{dimStyle.Render("none configured")}
	}

	shown := m.accounts
	if len(m.accounts) > topBarSectionRows {
		shown = m.accounts[:topBarSectionRows-1]
	}
	lines := make([]string, 0, len(shown)+1)
	for _, account := range shown {
		lines = append(lines, renderAccount(account))
	}
	if len(shown) < len(m.accounts) {
		lines = append(lines, dimStyle.Render(fmt.Sprintf("+%d more", len(m.accounts)-len(shown))))
	}
	return lines
}

func renderAccount(account Account) string {
	name := truncateName(account.Name, topBarNameMax)
	line := valueWhiteStyle.Render(accountPassiveMarker + name)
	if account.Selected {
		line = activeStyle.Render(accountActiveMarker + name)
	}
	if account.Provider != "" {
		line += dimStyle.Render(" " + account.Provider)
	}
	if account.Primary {
		line += dimStyle.Render(" (primary)")
	}
	return line
}

// buildShortcutsDisplay flows the view's shortcuts into as many columns of topBarSectionRows as
// fit in width, dropping the ones that don't fit.
func (m *TopBarModel) buildShortcutsDisplay(width int) []string {
	var formatted []string
	for _, shortcut := range m.shortcuts {
		parts := strings.SplitN(shortcut, ">", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimPrefix(parts[0], "<")
		formatted = append(formatted, shortcutBlueStyle.Render("<"+key+">")+" "+descGrayStyle.Render(strings.TrimSpace(parts[1])))
	}

	rows := make([]string, min(len(formatted), topBarSectionRows))
	used := 0
	for start := 0; start < len(formatted); start += topBarSectionRows {
		column := formatted[start:min(start+topBarSectionRows, len(formatted))]
		columnW := columnWidth(column)
		gap := 0
		if start > 0 {
			gap = topBarShortcutsGap
		}
		if used+gap+columnW > width {
			break
		}
		for i := range column {
			if start > 0 {
				rows[i] += strings.Repeat(" ", used+gap-lipgloss.Width(rows[i]))
			}
			rows[i] += column[i]
		}
		used += gap + columnW
	}
	return rows
}
