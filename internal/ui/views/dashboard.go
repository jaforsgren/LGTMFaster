package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	dashboardChecksLoading = " · checking builds (%d left)..."
	dashboardFailingFormat = "%s  (failing: %s)"
	checksLoadingIcon      = "…"
	// dashboardPaneChromeRows are the rows of a pane that are not pull requests: its title, the blank
	// line under it, the column header and the scroll position.
	dashboardPaneChromeRows = listTitleRows + listHeaderRows + listScrollInfoRows
	dashboardColumns        = 2
	// dashboardRows are the rows of pull request panes; the active runs pane spans a row below them.
	dashboardRows        = 2
	dashboardGridRows    = dashboardRows + 1
	dashboardRunsTitle   = "Your active pipeline runs"
	dashboardRunsLoading = "Loading pipeline runs..."
	noCursor             = -1
	// dashboardEmptyPaneRows is the height of a pane with nothing in it: only its title, so the
	// panes with something to show get the room.
	dashboardEmptyPaneRows = 1
)

// dashboardRunsPane is the focus index of the active runs pane, after the pull request panes.
var dashboardRunsPane = len(dashboardSections)

type dashboardSection struct {
	title    string
	includes func(pr domain.PullRequest, checks domain.Checks, checksLoaded bool) bool
}

func isOwnOpenPR(pr domain.PullRequest) bool {
	return pr.Category == domain.PRCategoryAuthored && pr.Status == domain.PRStatusOpen
}

// dashboardSections fill the panes column by column: the left column holds what waits on someone,
// the right one the user's pull requests that need action.
var dashboardSections = []dashboardSection{
	{"Waiting for your review", func(pr domain.PullRequest, _ domain.Checks, _ bool) bool {
		return pr.Category == domain.PRCategoryAssigned && !pr.ReviewedByMe && !pr.IsDraft && pr.Status == domain.PRStatusOpen
	}},
	{"Created by you", func(pr domain.PullRequest, _ domain.Checks, _ bool) bool {
		return isOwnOpenPR(pr)
	}},
	{"Failing checks", func(pr domain.PullRequest, checks domain.Checks, loaded bool) bool {
		return isOwnOpenPR(pr) && loaded && checks.State == domain.CheckStateFailing
	}},
	{"Ready to merge", func(pr domain.PullRequest, checks domain.Checks, loaded bool) bool {
		return isOwnOpenPR(pr) && !pr.IsDraft && pr.ApprovalStatus == domain.ApprovalStatusApproved && pr.Mergeable &&
			loaded && (checks.State == domain.CheckStatePassing || checks.State == domain.CheckStateNone)
	}},
}

var checkAppearances = map[domain.CheckState]prCategoryAppearance{
	domain.CheckStatePassing: {icon: "✓", color: lipgloss.Color("#10B981")},
	domain.CheckStateFailing: {icon: "✗", color: lipgloss.Color("#EF4444")},
	domain.CheckStatePending: {icon: "●", color: lipgloss.Color("#3B82F6")},
	domain.CheckStateNone:    {icon: "○", color: lipgloss.Color("#6B7280")},
}

// DashboardViewModel shows the loaded pull requests in a fixed grid of panes, one per section, each
// with its own cursor. Checks are only fetched for the user's own pull requests.
type DashboardViewModel struct {
	prs           []domain.PullRequest
	runs          []domain.PipelineRun
	runsLoaded    bool
	runsCursor    listCursor
	checks        map[string]domain.Checks
	checksPending map[string]bool
	sections      [][]domain.PullRequest
	cursors       []listCursor
	focus         int
	// focusChosen stops the focus from following the first filled pane once the user moved it.
	focusChosen bool
	width       int
	height      int
}

func NewDashboardView() *DashboardViewModel {
	cursors := make([]listCursor, len(dashboardSections))
	for i := range cursors {
		cursors[i] = newListCursor()
	}
	return &DashboardViewModel{
		checks:        make(map[string]domain.Checks),
		checksPending: make(map[string]bool),
		sections:      make([][]domain.PullRequest, len(dashboardSections)),
		cursors:       cursors,
		runsCursor:    newListCursor(),
	}
}

func (m *DashboardViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.fitCursors()
}

// fitCursors sizes every pane's cursor to its pane, which changes with the size and whenever a pane
// becomes empty or filled.
func (m *DashboardViewModel) fitCursors() {
	layout := m.layout()
	for section := range m.cursors {
		m.cursors[section].SetVisibleRows(layout.sectionHeight(section) - dashboardPaneChromeRows)
	}
	m.runsCursor.SetVisibleRows(layout.runsHeight - dashboardPaneChromeRows)
}

func (m *DashboardViewModel) SetPRs(prs []domain.PullRequest) {
	m.prs = prs
	m.rebuild()
}

// PendingChecks returns the user's open pull requests whose checks are not fetched yet, and marks
// them in flight.
func (m *DashboardViewModel) PendingChecks() []domain.PullRequest {
	var pending []domain.PullRequest
	for _, pr := range m.prs {
		key := pr.Key()
		if _, loaded := m.checks[key]; loaded || m.checksPending[key] || !isOwnOpenPR(pr) {
			continue
		}
		m.checksPending[key] = true
		pending = append(pending, pr)
	}
	return pending
}

// SetActiveRuns shows the runs newest first, keeping the cursor on the same run.
func (m *DashboardViewModel) SetActiveRuns(runs []domain.PipelineRun) {
	selectedKey := ""
	if run := m.selectedRun(); run != nil {
		selectedKey = run.FavoriteKey()
	}

	sorted := append([]domain.PipelineRun(nil), runs...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].QueuedAt.After(sorted[j].QueuedAt) })
	m.runs = sorted
	m.runsLoaded = true
	m.fitCursors()

	m.runsCursor.Clamp(len(sorted))
	for i, run := range sorted {
		if run.FavoriteKey() == selectedKey {
			m.runsCursor.SetIndex(i, len(sorted))
		}
	}
}

func (m *DashboardViewModel) ActiveRuns() []domain.PipelineRun {
	return m.runs
}

func (m *DashboardViewModel) SetChecks(prKey string, checks domain.Checks) {
	delete(m.checksPending, prKey)
	m.checks[prKey] = checks
	m.rebuild()
}

// ChecksFailed forgets the request; the pull request stays out of the sections that need checks.
func (m *DashboardViewModel) ChecksFailed(prKey string) {
	delete(m.checksPending, prKey)
}

// ResetChecks makes every check be fetched again, for a refresh.
func (m *DashboardViewModel) ResetChecks() {
	m.checks = make(map[string]domain.Checks)
	m.checksPending = make(map[string]bool)
	m.rebuild()
}

func (m *DashboardViewModel) SectionCounts() []int {
	counts := make([]int, len(m.sections))
	for i, section := range m.sections {
		counts[i] = len(section)
	}
	return counts
}

// GetSelectedPR is the pull request under the cursor of the focused pane, nil while the runs pane
// is focused.
func (m *DashboardViewModel) GetSelectedPR() *domain.PullRequest {
	if m.focus == dashboardRunsPane {
		return nil
	}
	section := m.sections[m.focus]
	index := m.cursors[m.focus].Index()
	if index < 0 || index >= len(section) {
		return nil
	}
	return &section[index]
}

// GetSelectedRun is the run under the cursor while the runs pane is focused.
func (m *DashboardViewModel) GetSelectedRun() *domain.PipelineRun {
	if m.focus != dashboardRunsPane {
		return nil
	}
	return m.selectedRun()
}

func (m *DashboardViewModel) selectedRun() *domain.PipelineRun {
	index := m.runsCursor.Index()
	if index < 0 || index >= len(m.runs) {
		return nil
	}
	return &m.runs[index]
}

func (m *DashboardViewModel) MoveUp() {
	m.focusedCursor().MoveUp()
}

func (m *DashboardViewModel) MoveDown() {
	if m.focus == dashboardRunsPane {
		m.runsCursor.MoveDown(len(m.runs))
		return
	}
	m.cursors[m.focus].MoveDown(len(m.sections[m.focus]))
}

func (m *DashboardViewModel) focusedCursor() *listCursor {
	if m.focus == dashboardRunsPane {
		return &m.runsCursor
	}
	return &m.cursors[m.focus]
}

// FocusNext and FocusPrev cycle through the pull request panes column by column, then the runs pane.
func (m *DashboardViewModel) FocusNext() {
	m.setFocus((m.focus + 1) % dashboardPaneCount())
}

func (m *DashboardViewModel) FocusPrev() {
	m.setFocus((m.focus + dashboardPaneCount() - 1) % dashboardPaneCount())
}

func dashboardPaneCount() int {
	return dashboardRunsPane + 1
}

// FocusRight and FocusLeft move to the pull request pane beside the focused one; the runs pane
// spans both columns, so they leave it alone.
func (m *DashboardViewModel) FocusRight() {
	if m.focus != dashboardRunsPane && m.focus+dashboardRows < len(dashboardSections) {
		m.setFocus(m.focus + dashboardRows)
	}
}

func (m *DashboardViewModel) FocusLeft() {
	if m.focus != dashboardRunsPane && m.focus >= dashboardRows {
		m.setFocus(m.focus - dashboardRows)
	}
}

func (m *DashboardViewModel) setFocus(section int) {
	m.focus = section
	m.focusChosen = true
}

// rebuild sorts the pull requests into the sections, newest update first, keeping each pane's
// cursor on the same pull request.
func (m *DashboardViewModel) rebuild() {
	for section, definition := range dashboardSections {
		selectedKey := ""
		if index := m.cursors[section].Index(); index < len(m.sections[section]) {
			selectedKey = m.sections[section][index].Key()
		}

		var members []domain.PullRequest
		for _, pr := range m.prs {
			checks, loaded := m.checks[pr.Key()]
			if definition.includes(pr, checks, loaded) {
				members = append(members, pr)
			}
		}
		sort.SliceStable(members, func(i, j int) bool { return members[i].UpdatedAt.After(members[j].UpdatedAt) })
		m.sections[section] = members

		m.cursors[section].Clamp(len(members))
		for i, pr := range members {
			if pr.Key() == selectedKey {
				m.cursors[section].SetIndex(i, len(members))
			}
		}
	}

	if !m.focusChosen {
		for section, members := range m.sections {
			if len(members) > 0 {
				m.focus = section
				break
			}
		}
	}
	m.fitCursors()
}

// dashboardLayout holds the inner sizes of the panes, which together occupy exactly the main
// window. Each column stacks its pull request panes on its own, so an empty pane only gives its
// room to the pane under or over it; the runs pane below spans the window.
type dashboardLayout struct {
	widths []int
	// heights holds the pull request panes' heights per column, top to bottom.
	heights    [][]int
	runsHeight int
}

func (l dashboardLayout) sectionHeight(section int) int {
	return l.heights[section/dashboardRows][section%dashboardRows]
}

func (m *DashboardViewModel) layout() dashboardLayout {
	outer := outerSize(m.height, paneVerticalPadding)
	gridOuter := outer / dashboardGridRows * dashboardRows
	runsOuter := outer - gridOuter
	if m.runsLoaded && len(m.runs) == 0 {
		runsOuter = outerSize(dashboardEmptyPaneRows, paneVerticalPadding)
		gridOuter = outer - runsOuter
	}

	heights := make([][]int, dashboardColumns)
	for column := range heights {
		empty := make([]bool, dashboardRows)
		for row := range empty {
			empty[row] = len(m.sections[column*dashboardRows+row]) == 0
		}
		heights[column] = stackPanes(gridOuter, empty)
	}
	return dashboardLayout{
		widths:     splitEvenly(outerSize(m.width, paneHorizontalPadding), dashboardColumns, paneHorizontalPadding),
		heights:    heights,
		runsHeight: innerSize(runsOuter, paneVerticalPadding),
	}
}

// stackPanes divides an outer height between panes stacked on each other and returns their inner
// heights: empty panes only show their title and the others share the rest evenly. When all are
// empty, the last one takes the rest so the stack still fills the height.
func stackPanes(outer int, empty []bool) []int {
	emptyOuter := outerSize(dashboardEmptyPaneRows, paneVerticalPadding)
	filled := make([]bool, len(empty))
	filledCount := 0
	for i, isEmpty := range empty {
		filled[i] = !isEmpty
		if filled[i] {
			filledCount++
		}
	}
	if filledCount == 0 {
		filled[len(filled)-1] = true
		filledCount = 1
	}

	shared := splitEvenly(outer-(len(empty)-filledCount)*emptyOuter, filledCount, paneVerticalPadding)
	heights := make([]int, len(empty))
	for i := range heights {
		if !filled[i] {
			heights[i] = dashboardEmptyPaneRows
			continue
		}
		heights[i], shared = shared[0], shared[1:]
	}
	return heights
}

// splitEvenly divides an outer size into parts, the last one taking the rounding remainder, and
// returns their inner sizes.
func splitEvenly(outer, parts, padding int) []int {
	sizes := make([]int, parts)
	for i := range sizes {
		part := outer / parts
		if i == parts-1 {
			part = outer - part*(parts-1)
		}
		sizes[i] = innerSize(part, padding)
	}
	return sizes
}

func (m *DashboardViewModel) View() string {
	layout := m.layout()
	columns := make([]string, dashboardColumns)
	for column := range columns {
		panes := make([]string, dashboardRows)
		for row := range panes {
			section := column*dashboardRows + row
			width, height := layout.widths[column], layout.heights[column][row]
			panes[row] = renderPane(m.renderSection(section, width), width, height, m.paneBorderColor(section))
		}
		columns[column] = lipgloss.JoinVertical(lipgloss.Left, panes...)
	}
	grid := lipgloss.JoinHorizontal(lipgloss.Top, columns...)
	runs := renderPane(m.renderRuns(), m.width, layout.runsHeight, m.paneBorderColor(dashboardRunsPane))
	return lipgloss.JoinVertical(lipgloss.Left, grid, runs)
}

func (m *DashboardViewModel) paneBorderColor(pane int) lipgloss.Color {
	if pane == m.focus {
		return focusedPaneBorderColor
	}
	return mainWindowBorderColor
}

func (m *DashboardViewModel) renderRuns() string {
	title := listTitleStyle.Render(fmt.Sprintf("%s (%d)", dashboardRunsTitle, len(m.runs)))
	if m.runsLoaded && len(m.runs) == 0 {
		return title
	}

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	if !m.runsLoaded {
		b.WriteString(listMutedStyle.Render(dashboardRunsLoading))
		return b.String()
	}

	cursor := m.runsCursor
	if m.focus != dashboardRunsPane {
		cursor.index = noCursor
	}
	b.WriteString(renderListRows(dashboardRunColumns(m.width), len(m.runs), cursor, func(i int) listRow {
		return dashboardRunRow(m.runs[i])
	}))
	return b.String()
}

func dashboardRunColumns(width int) []listColumn {
	const (
		runWidth         = 16
		statusWidth      = 22
		branchWidth      = 30
		queuedWidth      = 14
		durationWidth    = 10
		minPipelineWidth = 16
		columnSpacing    = 7
	)
	fixed := runWidth + statusWidth + runStageColumnWidth + branchWidth + queuedWidth + durationWidth +
		runCommitColumnWidth + columnSpacing
	return []listColumn{
		{"Pipeline", flexColumnWidth(width, fixed, minPipelineWidth)},
		{"Run", runWidth},
		{"Status", statusWidth},
		runStageColumn,
		{"Branch", branchWidth},
		{"Queued", queuedWidth},
		{"Duration", durationWidth},
		runCommitColumn,
	}
}

func dashboardRunRow(run domain.PipelineRun) listRow {
	return listRow{
		icon: renderPipelineStatusIcon(run.Status),
		values: []string{
			valueOrPlaceholder(run.PipelineName),
			valueOrPlaceholder(run.Number),
			PipelineStatusLabel(run.Status),
			runStageLabel(run),
			valueOrPlaceholder(run.SourceBranch),
			formatOptionalAge(run.QueuedAt),
			formatElapsed(run.StartedAt, run.FinishedAt),
			runCommitLabel(run),
		},
		style: listRowStyle,
	}
}

func (m *DashboardViewModel) renderSection(section, width int) string {
	members := m.sections[section]
	title := fmt.Sprintf("%s (%d)", dashboardSections[section].title, len(members))
	if loading := len(m.checksPending); loading > 0 && section >= dashboardRows {
		title += fmt.Sprintf(dashboardChecksLoading, loading)
	}

	if len(members) == 0 {
		return listTitleStyle.Render(title)
	}

	var b strings.Builder
	b.WriteString(listTitleStyle.Render(title))
	b.WriteString("\n\n")

	cursor := m.cursors[section]
	if section != m.focus {
		// Only the focused pane shows its cursor; the copy keeps the pane's scroll position.
		cursor.index = noCursor
	}
	b.WriteString(renderListRows(prColumns(width, true), len(members), cursor, func(i int) listRow {
		return m.prRow(members[i])
	}))
	return b.String()
}

// prRow shows the checks in the icon column once they are known, and names the failing ones.
func (m *DashboardViewModel) prRow(pr domain.PullRequest) listRow {
	row := prRow(pr, true)
	checks, loaded := m.checks[pr.Key()]
	switch {
	case loaded:
		appearance := checkAppearances[checks.State]
		row.icon = renderListIcon(appearance.icon, appearance.color)
	case m.checksPending[pr.Key()]:
		row.icon = renderListIcon(checksLoadingIcon, lipgloss.Color("#6B7280"))
	}
	if failing := checks.Failing(); len(failing) > 0 {
		row.values[1] = fmt.Sprintf(dashboardFailingFormat, row.values[1], strings.Join(failing, ", "))
	}
	return row
}
