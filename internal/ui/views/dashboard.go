package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	dashboardChecksLoading   = " · checking builds (%d left)..."
	dashboardActivityLoading = " · reading activity (%d left)..."
	dashboardFailingFormat   = "%s  (failing: %s)"
	dashboardNewsFormat      = "%s  (%s)"
	dashboardNewsMoreFormat  = "%s, +%d more"
	checksLoadingIcon        = "…"
	// The pull request grid takes dashboardGridParts of dashboardHeightParts of the height; the
	// active runs pane spans the rest below it.
	dashboardGridParts   = 2
	dashboardHeightParts = dashboardGridParts + 1
	// prTitleValue is the title's index in a compact pull request row.
	prTitleValue         = 1
	dashboardRunsTitle   = "Your active pipeline runs"
	dashboardRunsLoading = "Loading pipeline runs..."
	noCursor             = -1
	// dashboardEmptyPaneRows is the height of a pane with nothing in it: only its title, without
	// padding, so the panes with something to show get the room.
	dashboardEmptyPaneRows    = 1
	dashboardEmptyPanePadding = 0
)

// dashboardRunsPane is the focus index of the active runs pane, after the pull request panes.
var dashboardRunsPane = len(dashboardSections)

// dashboardDetail is what a section needs fetched per pull request before it can sort it in.
type dashboardDetail int

const (
	dashboardDetailNone dashboardDetail = iota
	dashboardDetailChecks
	dashboardDetailActivity
)

// dashboardPRState is what the dashboard fetched for a pull request.
type dashboardPRState struct {
	checks       domain.Checks
	checksLoaded bool
	// news is what others did since the user last looked, newest first.
	news []domain.Activity
}

type dashboardSection struct {
	title    string
	column   int
	needs    dashboardDetail
	includes func(pr domain.PullRequest, state dashboardPRState) bool
	// orderedAt sorts the section, newest first.
	orderedAt func(pr domain.PullRequest, state dashboardPRState) time.Time
}

func isOwnOpenPR(pr domain.PullRequest) bool {
	return pr.Category == domain.PRCategoryAuthored && pr.Status == domain.PRStatusOpen
}

// followsPR reports whether the user authored or reviews the open pull request, whose activity the
// News pane follows.
func followsPR(pr domain.PullRequest) bool {
	return pr.Status == domain.PRStatusOpen && (pr.Category == domain.PRCategoryAuthored || pr.Category == domain.PRCategoryAssigned)
}

func byUpdate(pr domain.PullRequest, _ dashboardPRState) time.Time {
	return pr.UpdatedAt
}

func byLatestNews(_ domain.PullRequest, state dashboardPRState) time.Time {
	return state.news[0].At
}

// dashboardSections are listed column by column, which is also the tab order: the left column holds
// what waits on the user, the right one what changed and the user's pull requests that need action.
var dashboardSections = []dashboardSection{
	{"Waiting for your review", 0, dashboardDetailNone, func(pr domain.PullRequest, _ dashboardPRState) bool {
		return pr.Category == domain.PRCategoryAssigned && !pr.ReviewedByMe && !pr.IsDraft && pr.Status == domain.PRStatusOpen
	}, byUpdate},
	{"Created by you", 0, dashboardDetailNone, func(pr domain.PullRequest, _ dashboardPRState) bool {
		return isOwnOpenPR(pr)
	}, byUpdate},
	{"News", 1, dashboardDetailActivity, func(pr domain.PullRequest, state dashboardPRState) bool {
		return followsPR(pr) && len(state.news) > 0
	}, byLatestNews},
	{"Failing checks", 1, dashboardDetailChecks, func(pr domain.PullRequest, state dashboardPRState) bool {
		return isOwnOpenPR(pr) && state.checksLoaded && state.checks.State == domain.CheckStateFailing
	}, byUpdate},
	{"Ready to merge", 1, dashboardDetailChecks, func(pr domain.PullRequest, state dashboardPRState) bool {
		return isOwnOpenPR(pr) && !pr.IsDraft && pr.ApprovalStatus == domain.ApprovalStatusApproved &&
			pr.MergeConflicts == domain.MergeConflictNone &&
			state.checksLoaded && (state.checks.State == domain.CheckStatePassing || state.checks.State == domain.CheckStateNone)
	}, byUpdate},
}

// dashboardColumnPanes holds each column's sections, top to bottom.
var dashboardColumnPanes = sectionsByColumn()

func sectionsByColumn() [][]int {
	var columns [][]int
	for section, definition := range dashboardSections {
		for len(columns) <= definition.column {
			columns = append(columns, nil)
		}
		columns[definition.column] = append(columns[definition.column], section)
	}
	return columns
}

// dashboardPanePosition is the column of a section and its row in that column.
func dashboardPanePosition(section int) (column, row int) {
	for column, sections := range dashboardColumnPanes {
		for row, candidate := range sections {
			if candidate == section {
				return column, row
			}
		}
	}
	return 0, 0
}

var checkAppearances = map[domain.CheckState]prCategoryAppearance{
	domain.CheckStatePassing: {icon: "✓", color: lipgloss.Color("#10B981")},
	domain.CheckStateFailing: {icon: "✗", color: lipgloss.Color("#EF4444")},
	domain.CheckStatePending: {icon: "●", color: lipgloss.Color("#3B82F6")},
	domain.CheckStateNone:    {icon: "○", color: lipgloss.Color("#6B7280")},
}

var activityAppearances = map[domain.ActivityKind]prCategoryAppearance{
	domain.ActivityKindOpened:  {icon: "●", color: lipgloss.Color("#A855F7")},
	domain.ActivityKindPush:    {icon: "↑", color: lipgloss.Color("#3B82F6")},
	domain.ActivityKindComment: {icon: "✎", color: lipgloss.Color("#F59E0B")},
	domain.ActivityKindReview:  {icon: "★", color: lipgloss.Color("#10B981")},
}

// DashboardViewModel shows the loaded pull requests in a fixed grid of panes, one per section, each
// with its own cursor. Checks are only fetched for the user's own pull requests, activity for the
// ones the user authored or reviews.
type DashboardViewModel struct {
	prs             []domain.PullRequest
	runs            []domain.PipelineRun
	runsLoaded      bool
	runsCursor      listCursor
	checks          map[string]domain.Checks
	checksPending   map[string]bool
	activity        map[string][]domain.Activity
	activityPending map[string]bool
	// lastViewed maps PullRequest.Key to when the user last opened it; news is what happened since,
	// at most newsWindow back.
	lastViewed map[string]time.Time
	newsWindow time.Duration
	sections   [][]domain.PullRequest
	cursors    []listCursor
	focus      int
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
		checks:          make(map[string]domain.Checks),
		checksPending:   make(map[string]bool),
		activity:        make(map[string][]domain.Activity),
		activityPending: make(map[string]bool),
		lastViewed:      make(map[string]time.Time),
		newsWindow:      domain.Settings{}.NewsWindow(),
		sections:        make([][]domain.PullRequest, len(dashboardSections)),
		cursors:         cursors,
		runsCursor:      newListCursor(),
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
		m.cursors[section].SetVisibleRows(visibleListRows(layout.sectionHeight(section), len(m.sections[section])))
	}
	m.runsCursor.SetVisibleRows(visibleListRows(layout.runs.inner, len(m.runs)))
}

// visibleListRows is how many of a list's count rows fit a pane of the inner height under its title
// and column header; the scroll position takes a row only when they do not all fit.
func visibleListRows(inner, count int) int {
	rows := inner - listTitleRows - listHeaderRows
	if count > rows {
		rows -= listScrollInfoRows
	}
	return rows
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

// PendingActivity returns the followed pull requests whose activity is not fetched yet, and marks
// them in flight.
func (m *DashboardViewModel) PendingActivity() []domain.PullRequest {
	var pending []domain.PullRequest
	for _, pr := range m.prs {
		key := pr.Key()
		if _, loaded := m.activity[key]; loaded || m.activityPending[key] || !followsPR(pr) {
			continue
		}
		m.activityPending[key] = true
		pending = append(pending, pr)
	}
	return pending
}

func (m *DashboardViewModel) SetActivity(prKey string, activity []domain.Activity) {
	delete(m.activityPending, prKey)
	m.activity[prKey] = activity
	m.rebuild()
}

// ActivityFailed forgets the request; the pull request stays out of the News pane.
func (m *DashboardViewModel) ActivityFailed(prKey string) {
	delete(m.activityPending, prKey)
}

// SetLastViewed takes when each pull request was last opened, keeping its own copy since
// MarkViewed adds to it.
func (m *DashboardViewModel) SetLastViewed(lastViewed map[string]time.Time) {
	m.lastViewed = make(map[string]time.Time, len(lastViewed))
	for key, viewedAt := range lastViewed {
		m.lastViewed[key] = viewedAt
	}
	m.rebuild()
}

// MarkViewed takes the pull request out of the News pane until something new happens on it.
func (m *DashboardViewModel) MarkViewed(prKey string, viewedAt time.Time) {
	m.lastViewed[prKey] = viewedAt
	m.rebuild()
}

func (m *DashboardViewModel) SetNewsWindow(window time.Duration) {
	m.newsWindow = window
	m.rebuild()
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

// ResetDetails makes every check and activity be fetched again, for a refresh.
func (m *DashboardViewModel) ResetDetails() {
	m.checks = make(map[string]domain.Checks)
	m.checksPending = make(map[string]bool)
	m.activity = make(map[string][]domain.Activity)
	m.activityPending = make(map[string]bool)
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

// FocusRight and FocusLeft move to the pull request pane beside the focused one, or the column's
// last pane when it has fewer; the runs pane spans both columns, so they leave it alone.
func (m *DashboardViewModel) FocusRight() {
	m.focusColumn(1)
}

func (m *DashboardViewModel) FocusLeft() {
	m.focusColumn(-1)
}

func (m *DashboardViewModel) focusColumn(step int) {
	if m.focus == dashboardRunsPane {
		return
	}
	column, row := dashboardPanePosition(m.focus)
	target := column + step
	if target < 0 || target >= len(dashboardColumnPanes) {
		return
	}
	panes := dashboardColumnPanes[target]
	m.setFocus(panes[min(row, len(panes)-1)])
}

func (m *DashboardViewModel) setFocus(section int) {
	m.focus = section
	m.focusChosen = true
}

func (m *DashboardViewModel) prState(pr domain.PullRequest) dashboardPRState {
	checks, loaded := m.checks[pr.Key()]
	return dashboardPRState{checks: checks, checksLoaded: loaded, news: m.newsOf(pr)}
}

// newsOf is what others did on the pull request since the user last opened it, but at most the
// news window back, so a pull request opened long ago does not stay in the News pane for good. Its
// opening counts before the rest of its activity has loaded.
func (m *DashboardViewModel) newsOf(pr domain.PullRequest) []domain.Activity {
	activity := append([]domain.Activity{pr.OpenedActivity()}, m.activity[pr.Key()]...)
	since := time.Now().Add(-m.newsWindow)
	if viewedAt := m.lastViewed[pr.Key()]; viewedAt.After(since) {
		since = viewedAt
	}
	return domain.ActivitySince(activity, since)
}

// rebuild sorts the pull requests into the sections, newest first, keeping each pane's cursor on
// the same pull request.
func (m *DashboardViewModel) rebuild() {
	states := make(map[string]dashboardPRState, len(m.prs))
	for _, pr := range m.prs {
		states[pr.Key()] = m.prState(pr)
	}

	for section, definition := range dashboardSections {
		selectedKey := ""
		if index := m.cursors[section].Index(); index < len(m.sections[section]) {
			selectedKey = m.sections[section][index].Key()
		}

		var members []domain.PullRequest
		for _, pr := range m.prs {
			if definition.includes(pr, states[pr.Key()]) {
				members = append(members, pr)
			}
		}
		sort.SliceStable(members, func(i, j int) bool {
			return definition.orderedAt(members[i], states[members[i].Key()]).After(definition.orderedAt(members[j], states[members[j].Key()]))
		})
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

// paneHeight is a pane's inner height and the vertical padding around it.
type paneHeight struct {
	inner   int
	padding int
}

var collapsedPaneHeight = paneHeight{inner: dashboardEmptyPaneRows, padding: dashboardEmptyPanePadding}

func (h paneHeight) outer() int {
	return outerSize(h.inner, h.padding)
}

// dashboardLayout holds the inner sizes of the panes, which together with the blank rows no list
// needs occupy exactly the main window. Each column stacks its pull request panes on its own, so a
// short list only gives its room to the panes under or over it; the runs pane below spans the window.
type dashboardLayout struct {
	widths []int
	// heights holds the pull request panes' heights by section.
	heights []paneHeight
	// gridHeight is the outer height of the columns; a column whose panes need less is padded.
	gridHeight int
	runs       paneHeight
	// spareRows are left blank between the grid and the runs pane when every list fits, which keeps
	// the runs pane at the bottom.
	spareRows int
}

func (l dashboardLayout) sectionHeight(section int) int {
	return l.heights[section].inner
}

func (m *DashboardViewModel) layout() dashboardLayout {
	outer := outerSize(m.height, paneVerticalPadding)
	// A short window gives the grid the runs pane's rows rather than squeezing a column's panes
	// below their title.
	gridOuter := max(outer/dashboardHeightParts*dashboardGridParts, minDashboardGridOuter())
	runs := paneHeight{inner: innerSize(outer-gridOuter, paneVerticalPadding), padding: paneVerticalPadding}
	gridNeeded := m.gridNeededOuter()
	switch {
	case m.runsLoaded && len(m.runs) == 0:
		runs = collapsedPaneHeight
	case m.runsLoaded:
		// The runs pane takes only the rows its runs need, and more of the grid's when the grid's
		// lists need less than their share.
		runs.inner = min(fittedListRows(len(m.runs)), runs.inner+max(gridOuter-gridNeeded, 0))
	}
	gridOuter = outer - runs.outer()
	spareRows := 0
	if m.runsLoaded && gridNeeded < gridOuter {
		spareRows = gridOuter - gridNeeded
		gridOuter = gridNeeded
	}

	heights := make([]paneHeight, len(dashboardSections))
	for _, sections := range dashboardColumnPanes {
		for row, height := range stackPanes(gridOuter, m.wantedRows(sections)) {
			heights[sections[row]] = height
		}
	}
	return dashboardLayout{
		widths:     splitEvenly(outerSize(m.width, paneHorizontalPadding), len(dashboardColumnPanes), paneHorizontalPadding),
		heights:    heights,
		gridHeight: gridOuter,
		runs:       runs,
		spareRows:  spareRows,
	}
}

// fittedListRows is the inner height of a pane that shows all of its count rows: the title, the
// column header and the rows, with no scroll position.
func fittedListRows(count int) int {
	return listTitleRows + listHeaderRows + count
}

// wantedRows is the inner height each section's list needs, 0 for an empty one.
func (m *DashboardViewModel) wantedRows(sections []int) []int {
	wants := make([]int, len(sections))
	for row, section := range sections {
		if count := len(m.sections[section]); count > 0 {
			wants[row] = fittedListRows(count)
		}
	}
	return wants
}

// gridNeededOuter is the height of the column whose lists need the most rows.
func (m *DashboardViewModel) gridNeededOuter() int {
	tallest := minDashboardGridOuter()
	for _, sections := range dashboardColumnPanes {
		needed := 0
		for _, want := range m.wantedRows(sections) {
			if want == 0 {
				needed += collapsedPaneHeight.outer()
				continue
			}
			needed += outerSize(want, paneVerticalPadding)
		}
		tallest = max(tallest, needed)
	}
	return tallest
}

// minDashboardGridOuter is the height of the tallest column with every pane but one shrunk to its
// title, the one left showing a single row, so a long list is never squeezed out by short ones.
func minDashboardGridOuter() int {
	tallest := 0
	for _, sections := range dashboardColumnPanes {
		tallest = max(tallest, len(sections))
	}
	return (tallest-1)*collapsedPaneHeight.outer() + outerSize(minPaneSize, paneVerticalPadding)
}

// stackPanes divides an outer height between panes stacked on each other, given the inner rows
// each one's list wants (0 when empty). Empty panes collapse to their title; the others get what
// their list needs, shorter lists first, and share what is left evenly when it is not enough. Rows
// no list needs are left over, for the caller to leave blank.
func stackPanes(outer int, wants []int) []paneHeight {
	heights := make([]paneHeight, len(wants))
	var filled []int
	for i, want := range wants {
		if want == 0 {
			heights[i] = collapsedPaneHeight
			continue
		}
		filled = append(filled, i)
	}
	room := outer - (len(wants)-len(filled))*collapsedPaneHeight.outer()

	sort.SliceStable(filled, func(i, j int) bool { return wants[filled[i]] < wants[filled[j]] })
	for n, pane := range filled {
		share := innerSize(room/(len(filled)-n), paneVerticalPadding)
		heights[pane] = paneHeight{inner: min(wants[pane], share), padding: paneVerticalPadding}
		room -= heights[pane].outer()
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
	columns := make([]string, len(dashboardColumnPanes))
	for column, sections := range dashboardColumnPanes {
		width := layout.widths[column]
		panes := make([]string, 0, len(sections)+1)
		used := 0
		for _, section := range sections {
			height := layout.heights[section]
			panes = append(panes, renderPaddedPane(m.renderSection(section, width), width, height.inner, height.padding, m.paneBorderColor(section)))
			used += height.outer()
		}
		if spare := layout.gridHeight - used; spare > 0 {
			panes = append(panes, blankRows(outerSize(width, paneHorizontalPadding), spare))
		}
		columns[column] = lipgloss.JoinVertical(lipgloss.Left, panes...)
	}
	grid := lipgloss.JoinHorizontal(lipgloss.Top, columns...)
	runs := renderPaddedPane(m.renderRuns(), m.width, layout.runs.inner, layout.runs.padding, m.paneBorderColor(dashboardRunsPane))
	if layout.spareRows == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, grid, runs)
	}
	return lipgloss.JoinVertical(lipgloss.Left, grid, blankRows(outerSize(m.width, paneHorizontalPadding), layout.spareRows), runs)
}

func blankRows(width, rows int) string {
	return lipgloss.NewStyle().Width(width).Height(rows).Render("")
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
	b.WriteString(renderListRows(pipelineRunColumns(m.width), len(m.runs), cursor, func(i int) listRow {
		return pipelineRunRow(m.runs[i])
	}))
	return b.String()
}

func pipelineRunColumns(width int) []listColumn {
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

func pipelineRunRow(run domain.PipelineRun) listRow {
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
	definition := dashboardSections[section]
	title := fmt.Sprintf("%s (%d)", definition.title, len(members)) + m.loadingSuffix(definition.needs)

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
	b.WriteString(renderListRows(prColumns(width, true, prAgeColumnTitle), len(members), cursor, func(i int) listRow {
		if definition.needs == dashboardDetailActivity {
			return m.newsRow(members[i])
		}
		return m.prRow(members[i])
	}))
	return b.String()
}

func (m *DashboardViewModel) loadingSuffix(needs dashboardDetail) string {
	switch {
	case needs == dashboardDetailChecks && len(m.checksPending) > 0:
		return fmt.Sprintf(dashboardChecksLoading, len(m.checksPending))
	case needs == dashboardDetailActivity && len(m.activityPending) > 0:
		return fmt.Sprintf(dashboardActivityLoading, len(m.activityPending))
	}
	return ""
}

// newsRow badges the latest news by its kind, dates the row by it and says who did what.
func (m *DashboardViewModel) newsRow(pr domain.PullRequest) listRow {
	news := m.newsOf(pr)
	latest := news[0]
	row := prRowWith(pr, true, prApprovalBadges[pr.ApprovalStatus], FormatAge(latest.At))
	appearance := activityAppearances[latest.Kind]
	row.icon = renderListIcon(appearance.icon, appearance.color)

	description := describeActivity(latest)
	if more := len(news) - 1; more > 0 {
		description = fmt.Sprintf(dashboardNewsMoreFormat, description, more)
	}
	row.values[prTitleValue] = fmt.Sprintf(dashboardNewsFormat, prTitle(pr), description)
	return row
}

func describeActivity(activity domain.Activity) string {
	if activity.Author.Username == "" {
		return activity.Summary
	}
	return activity.Author.Username + " " + activity.Summary
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
		row.values[prTitleValue] = fmt.Sprintf(dashboardFailingFormat, row.values[prTitleValue], strings.Join(failing, ", "))
	}
	return row
}
