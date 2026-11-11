package views

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type PipelineRunsViewModel struct {
	pipeline    *domain.Pipeline
	sourceRuns  []domain.PipelineRun
	visibleRuns []domain.PipelineRun
	loaded      bool
	cursor      listCursor
	filter      listFilter
	width       int
	height      int
}

func NewPipelineRunsView() *PipelineRunsViewModel {
	return &PipelineRunsViewModel{
		cursor: newListCursor(),
		filter: newListFilter("Filter by run, branch, or requester..."),
	}
}

func (m *PipelineRunsViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.cursor.SetVisibleRows(height - pipelineListChromeRows)
}

func (m *PipelineRunsViewModel) SetPipeline(pipeline domain.Pipeline) {
	m.pipeline = &pipeline
	m.sourceRuns = nil
	m.visibleRuns = nil
	m.loaded = false
	m.filter.Clear()
	m.cursor.Reset()
}

func (m *PipelineRunsViewModel) GetPipeline() *domain.Pipeline {
	return m.pipeline
}

// SetRuns keeps the selection on the same run ID across watch refreshes, since new runs shift indices.
func (m *PipelineRunsViewModel) SetRuns(runs []domain.PipelineRun) {
	selected := m.GetSelectedRun()
	m.sourceRuns = append([]domain.PipelineRun(nil), runs...)
	m.loaded = true
	m.rebuild()
	if selected == nil {
		return
	}
	for i, run := range m.visibleRuns {
		if run.ID == selected.ID {
			m.cursor.SetIndex(i, len(m.visibleRuns))
			return
		}
	}
}

func (m *PipelineRunsViewModel) GetSelectedRun() *domain.PipelineRun {
	idx := m.cursor.Index()
	if idx < 0 || idx >= len(m.visibleRuns) {
		return nil
	}
	return &m.visibleRuns[idx]
}

func (m *PipelineRunsViewModel) MoveUp() {
	m.cursor.MoveUp()
}

func (m *PipelineRunsViewModel) MoveDown() {
	m.cursor.MoveDown(len(m.visibleRuns))
}

func (m *PipelineRunsViewModel) ActivateFilter() {
	m.filter.Activate()
}

func (m *PipelineRunsViewModel) DeactivateFilter() {
	m.filter.Deactivate()
}

func (m *PipelineRunsViewModel) ClearFilter() {
	m.filter.Clear()
	m.rebuild()
}

func (m *PipelineRunsViewModel) IsFiltering() bool {
	return m.filter.IsActive()
}

func (m *PipelineRunsViewModel) GetFilterText() string {
	return m.filter.Text()
}

func (m *PipelineRunsViewModel) UpdateFilter(msg tea.Msg) tea.Cmd {
	cmd := m.filter.Update(msg)
	m.rebuild()
	m.cursor.Reset()
	return cmd
}

func (m *PipelineRunsViewModel) rebuild() {
	var filtered []domain.PipelineRun
	for _, run := range m.sourceRuns {
		if m.filter.Matches(run.Number, strconv.Itoa(run.ID), run.SourceBranch, run.RequestedBy.Username) {
			filtered = append(filtered, run)
		}
	}
	m.visibleRuns = filtered
	m.cursor.Clamp(len(m.visibleRuns))
}

func (m *PipelineRunsViewModel) columns() []listColumn {
	const (
		statusWidth    = 22
		branchWidth    = 30
		requesterWidth = 22
		queuedWidth    = 14
		durationWidth  = 10
		minRunWidth    = 16
		columnSpacing  = 5
	)
	fixed := statusWidth + branchWidth + requesterWidth + queuedWidth + durationWidth + columnSpacing
	return []listColumn{
		{"Run", flexColumnWidth(m.width, fixed, minRunWidth)},
		{"Status", statusWidth},
		{"Branch", branchWidth},
		{"Requested by", requesterWidth},
		{"Queued", queuedWidth},
		{"Duration", durationWidth},
	}
}

func (m *PipelineRunsViewModel) View() string {
	var b strings.Builder

	title := "Pipeline Runs"
	if m.pipeline != nil {
		title = "Runs: " + PipelineDisplayName(*m.pipeline)
	}
	b.WriteString(pipelineTitleStyle.Render(title))
	b.WriteString("\n\n")

	switch {
	case !m.loaded:
		b.WriteString(pipelineMutedStyle.Render("Loading runs..."))
		b.WriteString("\n")
	case len(m.visibleRuns) == 0 && m.filter.Text() != "":
		b.WriteString(pipelineMutedStyle.Render("No runs match the filter."))
		b.WriteString("\n")
	case len(m.visibleRuns) == 0:
		b.WriteString(pipelineMutedStyle.Render("This pipeline has no runs yet."))
		b.WriteString("\n")
	default:
		columns := m.columns()
		b.WriteString(renderListHeader(columns))
		b.WriteString("\n")

		start, end := m.cursor.VisibleRange(len(m.visibleRuns))
		for i := start; i < end; i++ {
			b.WriteString(renderRun(m.visibleRuns[i], columns, i == m.cursor.Index()))
			b.WriteString("\n")
		}
		if len(m.visibleRuns) > end-start {
			b.WriteString(renderScrollInfo(start, end, len(m.visibleRuns)))
			b.WriteString("\n")
		}
	}

	if m.filter.IsActive() {
		b.WriteString("\n")
		b.WriteString(m.filter.View())
	}

	b.WriteString("\n")
	b.WriteString(pipelineMutedStyle.Render(m.helpText()))
	return b.String()
}

func renderRun(run domain.PipelineRun, columns []listColumn, selected bool) string {
	values := []string{
		valueOrPlaceholder(run.Number),
		PipelineStatusLabel(run.Status),
		valueOrPlaceholder(run.SourceBranch),
		valueOrPlaceholder(run.RequestedBy.Username),
		formatOptionalAge(run.QueuedAt),
		formatRunDuration(run),
	}
	return renderListRow(run.Status, columns, values, selected)
}

func (m *PipelineRunsViewModel) helpText() string {
	if m.filter.IsActive() {
		return "Type to filter | Enter/Esc: Close"
	}
	help := "Enter: Inspect run | ctrl+o: Open run | r: Refresh | /: Filter | q: Back | auto-refreshing"
	if m.filter.Text() != "" {
		help = strings.Replace(help, "q: Back", "Esc: Clear filter | q: Back", 1)
	}
	return help
}
