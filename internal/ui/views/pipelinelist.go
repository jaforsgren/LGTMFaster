package views

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	pipelineFolderSeparator = `\`
	pipelineListChromeRows  = 12
)

type PipelineListViewModel struct {
	sourcePipelines  []domain.Pipeline
	visiblePipelines []domain.Pipeline
	cursor           listCursor
	filter           listFilter
	width            int
	height           int
}

func NewPipelineListView() *PipelineListViewModel {
	return &PipelineListViewModel{
		cursor: newListCursor(),
		filter: newListFilter("Filter by project, repository, or pipeline name..."),
	}
}

func (m *PipelineListViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.cursor.SetVisibleRows(height - pipelineListChromeRows)
}

func (m *PipelineListViewModel) SetPipelines(pipelines []domain.Pipeline) {
	selected := m.GetSelectedPipeline()
	m.sourcePipelines = append([]domain.Pipeline(nil), pipelines...)
	m.rebuild()
	m.restoreSelection(selected)
}

func (m *PipelineListViewModel) HasPipelines() bool {
	return len(m.sourcePipelines) > 0
}

func (m *PipelineListViewModel) PipelineCount() int {
	return len(m.sourcePipelines)
}

func (m *PipelineListViewModel) GetSelectedPipeline() *domain.Pipeline {
	idx := m.cursor.Index()
	if idx < 0 || idx >= len(m.visiblePipelines) {
		return nil
	}
	return &m.visiblePipelines[idx]
}

func (m *PipelineListViewModel) MoveUp() {
	m.cursor.MoveUp()
}

func (m *PipelineListViewModel) MoveDown() {
	m.cursor.MoveDown(len(m.visiblePipelines))
}

func (m *PipelineListViewModel) ActivateFilter() {
	m.filter.Activate()
}

func (m *PipelineListViewModel) DeactivateFilter() {
	m.filter.Deactivate()
}

func (m *PipelineListViewModel) ClearFilter() {
	m.filter.Clear()
	m.rebuild()
}

func (m *PipelineListViewModel) IsFiltering() bool {
	return m.filter.IsActive()
}

func (m *PipelineListViewModel) GetFilterText() string {
	return m.filter.Text()
}

func (m *PipelineListViewModel) UpdateFilter(msg tea.Msg) tea.Cmd {
	cmd := m.filter.Update(msg)
	m.rebuild()
	m.cursor.Reset()
	return cmd
}

func (m *PipelineListViewModel) rebuild() {
	var filtered []domain.Pipeline
	for _, pipeline := range m.sourcePipelines {
		if m.filter.Matches(pipeline.Project, pipeline.Repository, PipelineDisplayName(pipeline)) {
			filtered = append(filtered, pipeline)
		}
	}
	m.visiblePipelines = sortPipelines(filtered)
	m.cursor.Clamp(len(m.visiblePipelines))
}

func (m *PipelineListViewModel) restoreSelection(previous *domain.Pipeline) {
	if previous == nil {
		return
	}
	for i, pipeline := range m.visiblePipelines {
		if pipeline.ID == previous.ID && pipeline.Project == previous.Project && pipeline.PATID == previous.PATID {
			m.cursor.SetIndex(i, len(m.visiblePipelines))
			return
		}
	}
}

func sortPipelines(pipelines []domain.Pipeline) []domain.Pipeline {
	out := append([]domain.Pipeline(nil), pipelines...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if !strings.EqualFold(a.Project, b.Project) {
			return strings.ToLower(a.Project) < strings.ToLower(b.Project)
		}
		if !strings.EqualFold(a.Repository, b.Repository) {
			return strings.ToLower(a.Repository) < strings.ToLower(b.Repository)
		}
		return strings.ToLower(PipelineDisplayName(a)) < strings.ToLower(PipelineDisplayName(b))
	})
	return out
}

// PipelineDisplayName prefixes the pipeline name with its folder; the root folder is omitted.
func PipelineDisplayName(pipeline domain.Pipeline) string {
	folder := strings.Trim(pipeline.Folder, pipelineFolderSeparator)
	if folder == "" {
		return pipeline.Name
	}
	return folder + pipelineFolderSeparator + pipeline.Name
}

func (m *PipelineListViewModel) columns() []listColumn {
	const (
		projectWidth  = 20
		repoWidth     = 25
		lastRunWidth  = 18
		branchWidth   = 20
		ageWidth      = 14
		minNameWidth  = 20
		columnSpacing = 5
	)
	fixed := projectWidth + repoWidth + lastRunWidth + branchWidth + ageWidth + columnSpacing
	return []listColumn{
		{"Project", projectWidth},
		{"Repository", repoWidth},
		{"Pipeline", flexColumnWidth(m.width, fixed, minNameWidth)},
		{"Last run", lastRunWidth},
		{"Branch", branchWidth},
		{"Age", ageWidth},
	}
}

func (m *PipelineListViewModel) View() string {
	var b strings.Builder

	b.WriteString(pipelineTitleStyle.Render("Pipelines"))
	b.WriteString("\n\n")

	if len(m.visiblePipelines) == 0 {
		if m.filter.Text() != "" {
			b.WriteString(pipelineMutedStyle.Render("No pipelines match the filter."))
		} else {
			b.WriteString(pipelineMutedStyle.Render("No pipelines loaded."))
		}
		b.WriteString("\n")
	} else {
		columns := m.columns()
		b.WriteString(renderListHeader(columns))
		b.WriteString("\n")

		start, end := m.cursor.VisibleRange(len(m.visiblePipelines))
		for i := start; i < end; i++ {
			b.WriteString(m.renderPipeline(m.visiblePipelines[i], columns, i == m.cursor.Index()))
			b.WriteString("\n")
		}
		if len(m.visiblePipelines) > end-start {
			b.WriteString(renderScrollInfo(start, end, len(m.visiblePipelines)))
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

func (m *PipelineListViewModel) renderPipeline(pipeline domain.Pipeline, columns []listColumn, selected bool) string {
	status := domain.PipelineStatusUnknown
	lastRun, branch, age := emptyCellPlaceholder, emptyCellPlaceholder, emptyCellPlaceholder
	if run := pipeline.LatestRun; run != nil {
		status = run.Status
		lastRun = valueOrPlaceholder(run.Number)
		branch = valueOrPlaceholder(run.SourceBranch)
		age = formatOptionalAge(run.QueuedAt)
	}

	values := []string{
		pipeline.Project,
		pipeline.Repository,
		PipelineDisplayName(pipeline),
		lastRun,
		branch,
		age,
	}
	return renderListRow(status, columns, values, selected)
}

func (m *PipelineListViewModel) helpText() string {
	if m.filter.IsActive() {
		return "Type to filter | Enter/Esc: Close"
	}
	if m.filter.Text() != "" {
		return "Enter: Runs | ctrl+o: Open pipeline | r: Refresh | /: Filter | Esc: Clear filter | q: Back"
	}
	return "Enter: Runs | ctrl+o: Open pipeline | r: Refresh | /: Filter | q: Back"
}
