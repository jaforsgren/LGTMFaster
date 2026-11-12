package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	pipelineLogStepsTitle   = "Steps"
	pipelineLogSearchRows   = 1
	logLineNumberWidth      = 6
	logStepDurationWidth    = 8
	followingLabel          = " [following]"
	noLogsText              = "This run has no step logs yet."
	loadingLogsText         = "Loading logs..."
	loadingLogLinesText     = "Loading log..."
	emptyLogText            = "No output yet."
	noStepSelectedText      = "No step selected."
	logStepMatchCountFormat = " (%d)"
)

var (
	logErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	logWarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	logMarkedStyle  = lipgloss.NewStyle().Background(lipgloss.Color("#F59E0B")).Foreground(lipgloss.Color("#000000"))
	logFollowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Bold(true)
)

// LogLinesRequest is a read of a step's log after the lines already shown. Finished tells that
// the step had finished when it was asked for, so the lines read are all there will be.
type LogLinesRequest struct {
	Log      domain.PipelineLog
	Offset   int
	Finished bool
}

// PipelineLogViewModel shows a run's step logs: the steps on the left, the selected step's log on
// the right. Following keeps the newest running step selected and its log scrolled to the end.
type PipelineLogViewModel struct {
	run          *domain.PipelineRun
	initialStage string
	logs         []domain.PipelineLog
	logsLoaded   bool
	steps        listCursor
	lines        map[int][]string
	complete     map[int]bool
	inFlight     map[int]bool
	scroll       int
	// marked is the line the last search or error jump landed on, -1 when none.
	marked      int
	stepsHidden bool
	logFocused  bool
	following   bool
	search      searchField
	width       int
	height      int
}

func NewPipelineLogView() *PipelineLogViewModel {
	return &PipelineLogViewModel{steps: newListCursor(), search: newSearchField("text in this step's log"), marked: -1}
}

func (m *PipelineLogViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.steps.SetVisibleRows(height - listTitleRows)
	m.clampScroll()
}

// Open shows the logs of run, opening on the given stage's step that most needs attention once the
// logs are loaded. An unfinished run is followed.
func (m *PipelineLogViewModel) Open(run domain.PipelineRun, stage string) {
	m.run = &run
	m.initialStage = stage
	m.logs = nil
	m.logsLoaded = false
	m.steps.Reset()
	m.lines = make(map[int][]string)
	m.complete = make(map[int]bool)
	m.inFlight = make(map[int]bool)
	m.scroll = 0
	m.marked = -1
	m.stepsHidden = false
	m.logFocused = false
	m.following = !run.Status.IsFinished()
	m.ClearSearch()
}

func (m *PipelineLogViewModel) GetRun() *domain.PipelineRun {
	return m.run
}

// SetRun refreshes the run's status; following stops once it has finished and every step is read.
func (m *PipelineLogViewModel) SetRun(run domain.PipelineRun) {
	if m.run != nil && m.run.Identifier() == run.Identifier() {
		m.run = &run
	}
}

func (m *PipelineLogViewModel) IsFollowing() bool {
	return m.following
}

func (m *PipelineLogViewModel) ToggleFollow() {
	m.following = !m.following
	if m.following {
		m.followNewestStep()
	}
}

// SetLogs keeps the selected step across refreshes, unless following moves it on.
func (m *PipelineLogViewModel) SetLogs(logs []domain.PipelineLog) {
	selected := m.SelectedLog()
	isFirstLoad := !m.logsLoaded
	m.logs = append([]domain.PipelineLog(nil), logs...)
	m.logsLoaded = true

	switch {
	case isFirstLoad:
		m.selectStep(domain.DefaultLogIndex(m.logs, m.initialStage))
	case m.following:
		m.followNewestStep()
	case selected != nil:
		for i, log := range m.logs {
			if log.ID == selected.ID {
				m.steps.SetIndex(i, len(m.logs))
				return
			}
		}
		m.steps.Clamp(len(m.logs))
	}
}

func (m *PipelineLogViewModel) followNewestStep() {
	if index := domain.FollowedLogIndex(m.logs); index != m.steps.Index() {
		m.selectStep(index)
	}
	m.scrollToEnd()
}

func (m *PipelineLogViewModel) selectStep(index int) {
	if index < 0 {
		return
	}
	m.steps.SetIndex(index, len(m.logs))
	m.scroll = 0
	m.marked = -1
	if m.following {
		m.scrollToEnd()
	}
}

func (m *PipelineLogViewModel) SelectedLog() *domain.PipelineLog {
	index := m.steps.Index()
	if index < 0 || index >= len(m.logs) {
		return nil
	}
	return &m.logs[index]
}

// PendingLinesRequest is the read the selected step needs, if any: the rest of its log, unless it
// was read completely or a read is already in flight. It marks the read as in flight.
func (m *PipelineLogViewModel) PendingLinesRequest() (LogLinesRequest, bool) {
	log := m.SelectedLog()
	if log == nil || m.complete[log.ID] || m.inFlight[log.ID] {
		return LogLinesRequest{}, false
	}
	m.inFlight[log.ID] = true
	return LogLinesRequest{Log: *log, Offset: len(m.lines[log.ID]), Finished: log.Status.IsFinished()}, true
}

// AppendLines adds the lines a request read. A failed step's log opens on its first error.
func (m *PipelineLogViewModel) AppendLines(request LogLinesRequest, lines []string) {
	logID := request.Log.ID
	delete(m.inFlight, logID)
	if request.Offset != len(m.lines[logID]) {
		return
	}
	m.lines[logID] = append(m.lines[logID], lines...)
	if request.Finished {
		m.complete[logID] = true
	}

	selected := m.SelectedLog()
	if selected == nil || selected.ID != logID {
		return
	}
	switch {
	case m.following:
		m.scrollToEnd()
	case request.Offset == 0 && selected.Status == domain.PipelineStatusFailed:
		m.JumpToLevel(domain.LogLineError, true)
	}
}

// LinesFailed forgets a read that failed, so the next refresh asks again.
func (m *PipelineLogViewModel) LinesFailed(logID int) {
	delete(m.inFlight, logID)
}

// ReloadLines drops what was read, so every step is read again from the start.
func (m *PipelineLogViewModel) ReloadLines() {
	m.lines = make(map[int][]string)
	m.complete = make(map[int]bool)
	m.inFlight = make(map[int]bool)
	m.scroll = 0
	m.marked = -1
}

func (m *PipelineLogViewModel) currentLines() []string {
	if log := m.SelectedLog(); log != nil {
		return m.lines[log.ID]
	}
	return nil
}

// MoveUp and MoveDown change the step, which stops following.
func (m *PipelineLogViewModel) MoveUp() {
	m.following = false
	m.steps.MoveUp()
	m.scroll, m.marked = 0, -1
}

func (m *PipelineLogViewModel) MoveDown() {
	m.following = false
	m.steps.MoveDown(len(m.logs))
	m.scroll, m.marked = 0, -1
}

// The steps list and the log pane form a split view: tab hides the steps, → focuses the log.

func (m *PipelineLogViewModel) TogglePreview() {
	m.stepsHidden = !m.stepsHidden
	m.logFocused = m.stepsHidden
}

func (m *PipelineLogViewModel) IsPreviewOpen() bool {
	return true
}

func (m *PipelineLogViewModel) FocusNextPane() {
	m.logFocused = true
}

func (m *PipelineLogViewModel) FocusPrevPane() {
	m.logFocused = false
	m.stepsHidden = false
}

func (m *PipelineLogViewModel) IsPaneFocused() bool {
	return m.logFocused
}

// ScrollPane scrolls the log; scrolling up stops following, like leaving the end in `less +F`.
func (m *PipelineLogViewModel) ScrollPane(delta int) {
	if delta < 0 {
		m.following = false
	}
	m.scroll += delta
	m.clampScroll()
}

func (m *PipelineLogViewModel) PageUp() {
	m.ScrollPane(-m.logBodyRows())
}

func (m *PipelineLogViewModel) PageDown() {
	m.ScrollPane(m.logBodyRows())
}

func (m *PipelineLogViewModel) ScrollToTop() {
	m.following = false
	m.scroll = 0
}

func (m *PipelineLogViewModel) ScrollToBottom() {
	m.scrollToEnd()
}

func (m *PipelineLogViewModel) scrollToEnd() {
	m.scroll = m.maxScroll()
}

func (m *PipelineLogViewModel) maxScroll() int {
	return max(len(m.currentLines())-m.logBodyRows(), 0)
}

func (m *PipelineLogViewModel) clampScroll() {
	m.scroll = min(max(m.scroll, 0), m.maxScroll())
}

// logBodyRows are the log lines the pane shows below its title and above the search field.
func (m *PipelineLogViewModel) logBodyRows() int {
	return max(m.height-listTitleRows-pipelineLogSearchRows, minPaneSize)
}

func (m *PipelineLogViewModel) ActivateSearch() {
	m.search.Activate()
}

func (m *PipelineLogViewModel) DeactivateSearch() {
	m.search.Deactivate()
}

func (m *PipelineLogViewModel) IsSearching() bool {
	return m.search.IsActive()
}

func (m *PipelineLogViewModel) SearchQuery() string {
	return m.search.Query()
}

func (m *PipelineLogViewModel) ClearSearch() {
	m.search.Clear()
	m.marked = -1
}

// UpdateSearch edits the query and moves to its first match from where the log is scrolled.
func (m *PipelineLogViewModel) UpdateSearch(msg tea.Msg) tea.Cmd {
	cmd := m.search.Update(msg)
	m.marked = -1
	if m.SearchQuery() != "" {
		m.jumpTo(m.scroll-1, true, m.matchesQuery)
	}
	return cmd
}

// NextMatch and PrevMatch move to the next or previous line containing the query, and report
// whether there was one.
func (m *PipelineLogViewModel) NextMatch() bool {
	return m.SearchQuery() != "" && m.jumpTo(m.searchAnchor(), true, m.matchesQuery)
}

func (m *PipelineLogViewModel) PrevMatch() bool {
	return m.SearchQuery() != "" && m.jumpTo(m.searchAnchor(), false, m.matchesQuery)
}

// JumpToLevel moves to the next or previous error or warning line, and reports whether there was one.
func (m *PipelineLogViewModel) JumpToLevel(level domain.LogLineLevel, forward bool) bool {
	return m.jumpTo(m.searchAnchor(), forward, func(line string) bool {
		return domain.ClassifyLogLine(line) == level
	})
}

// searchAnchor is where the next jump starts from: the marked line, else just before the top of
// the log pane.
func (m *PipelineLogViewModel) searchAnchor() int {
	if m.marked >= 0 {
		return m.marked
	}
	return m.scroll - 1
}

func (m *PipelineLogViewModel) matchesQuery(line string) bool {
	return m.search.Matches(line)
}

// jumpTo marks the first line after (or before) from that matches, centring it in the pane.
func (m *PipelineLogViewModel) jumpTo(from int, forward bool, matches func(string) bool) bool {
	lines := m.currentLines()
	step := 1
	if !forward {
		step = -1
		if from < 0 {
			from = len(lines)
		}
	}
	for i := from + step; i >= 0 && i < len(lines); i += step {
		if matches(lines[i]) {
			m.following = false
			m.marked = i
			m.scroll = i - m.logBodyRows()/2
			m.clampScroll()
			return true
		}
	}
	return false
}

// matchPosition is the marked match's position among all matches of the query, and their count.
func (m *PipelineLogViewModel) matchPosition() (int, int) {
	position, count := 0, 0
	for i, line := range m.currentLines() {
		if m.matchesQuery(line) {
			count++
			if i == m.marked {
				position = count
			}
		}
	}
	return position, count
}

func (m *PipelineLogViewModel) View() string {
	if m.run == nil {
		return RenderMainWindow(listMutedStyle.Render("No run selected."), m.width, m.height)
	}

	if m.stepsHidden {
		return m.renderLogPane(m.width)
	}
	stepsWidth, logWidth := splitWidths(m.width)
	return lipgloss.JoinHorizontal(lipgloss.Top, m.renderStepsPane(stepsWidth), m.renderLogPane(logWidth))
}

func (m *PipelineLogViewModel) paneBorder(focused bool, unfocused lipgloss.Color) lipgloss.Color {
	if focused {
		return focusedPaneBorderColor
	}
	return unfocused
}

func (m *PipelineLogViewModel) renderStepsPane(width int) string {
	var b strings.Builder
	b.WriteString(renderListTitle(pipelinesTitle, valueOrPlaceholder(m.run.PipelineName), fmt.Sprintf("Run %s", valueOrPlaceholder(m.run.Number)), pipelineLogStepsTitle))
	b.WriteString("\n\n")

	switch {
	case !m.logsLoaded:
		b.WriteString(listMutedStyle.Render(loadingLogsText))
	case len(m.logs) == 0:
		b.WriteString(listMutedStyle.Render(noLogsText))
	default:
		start, end := m.steps.VisibleRange(len(m.logs))
		rows := make([]string, 0, end-start)
		for i := start; i < end; i++ {
			rows = append(rows, m.renderStep(m.logs[i], i == m.steps.Index(), width))
		}
		b.WriteString(strings.Join(rows, "\n"))
	}
	return renderPane(b.String(), width, m.height, m.paneBorder(!m.logFocused, mainWindowBorderColor))
}

func (m *PipelineLogViewModel) renderStep(log domain.PipelineLog, selected bool, width int) string {
	marker, style := unselectedRowMarker, listRowStyle
	if selected {
		marker, style = selectedRowMarker, listSelectedRowStyle
	}
	label := log.Path()
	if query := m.SearchQuery(); query != "" {
		if lines, read := m.lines[log.ID]; read {
			label += fmt.Sprintf(logStepMatchCountFormat, countMatches(lines, query))
		}
	}
	const columnSpacing = 2
	nameWidth := max(width-len(marker)-listIconColumnWidth-logStepDurationWidth-columnSpacing, minPaneSize)
	return style.Render(marker) + renderPipelineStatusIcon(log.Status) + " " +
		style.Render(padToWidth(truncateString(label, nameWidth), nameWidth)) + " " +
		listMutedStyle.Render(formatElapsed(log.StartedAt, log.FinishedAt))
}

func countMatches(lines []string, query string) int {
	count := 0
	for _, line := range lines {
		if containsFold(line, query) {
			count++
		}
	}
	return count
}

func (m *PipelineLogViewModel) renderLogPane(width int) string {
	var b strings.Builder
	log := m.SelectedLog()
	title := noStepSelectedText
	if log != nil {
		title = log.Path()
	}
	b.WriteString(listTitleStyle.Render(truncateString(title, width)))
	if m.following {
		b.WriteString(logFollowStyle.Render(followingLabel))
	}
	b.WriteString("\n\n")
	b.WriteString(m.renderLogBody(log, width))
	b.WriteString("\n")
	b.WriteString(m.renderSearchLine())
	return renderPane(b.String(), width, m.height, m.paneBorder(m.logFocused, detailPaneBorderColor))
}

func (m *PipelineLogViewModel) renderLogBody(log *domain.PipelineLog, width int) string {
	rows := m.logBodyRows()
	body := make([]string, rows)
	lines := m.currentLines()
	switch {
	case log == nil:
	case len(lines) == 0 && !m.complete[log.ID] && m.inFlight[log.ID]:
		body[0] = listMutedStyle.Render(loadingLogLinesText)
	case len(lines) == 0:
		body[0] = listMutedStyle.Render(emptyLogText)
	default:
		start := min(m.scroll, m.maxScroll())
		for row := 0; row < rows && start+row < len(lines); row++ {
			body[row] = m.renderLogLine(start+row, lines[start+row], width)
		}
	}
	return strings.Join(body, "\n")
}

func (m *PipelineLogViewModel) renderLogLine(index int, line string, width int) string {
	number := listMutedStyle.Render(fmt.Sprintf("%*d ", logLineNumberWidth-1, index+1))
	text := truncateString(line, max(width-logLineNumberWidth, minPaneSize))
	switch {
	case index == m.marked:
		return number + logMarkedStyle.Render(text)
	case m.matchesQuery(line):
		return number + searchMatchStyle.Render(text)
	}
	switch domain.ClassifyLogLine(line) {
	case domain.LogLineError:
		return number + logErrorStyle.Render(text)
	case domain.LogLineWarning:
		return number + logWarningStyle.Render(text)
	default:
		return number + listRowStyle.Render(text)
	}
}

func (m *PipelineLogViewModel) renderSearchLine() string {
	position, count := m.matchPosition()
	return m.search.View(position, count)
}
