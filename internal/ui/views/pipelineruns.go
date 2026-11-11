package views

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type PipelineRunsViewModel struct {
	pipeline       *domain.Pipeline
	sourceRuns     []domain.PipelineRun
	visibleRuns    []domain.PipelineRun
	loaded         bool
	favorites      map[string]bool
	previewStages  map[string][]domain.PipelineStage
	previewPending map[string]bool
	cursor         listCursor
	filter         listFilter
	split          splitView
	width          int
	height         int
}

func NewPipelineRunsView() *PipelineRunsViewModel {
	return &PipelineRunsViewModel{
		favorites:      make(map[string]bool),
		previewStages:  make(map[string][]domain.PipelineStage),
		previewPending: make(map[string]bool),
		cursor:         newListCursor(),
		filter:         newListFilter("Filter by run, branch, or requester..."),
	}
}

func (m *PipelineRunsViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.cursor.SetVisibleRows(height - listChromeRows)
}

func (m *PipelineRunsViewModel) SetPipeline(pipeline domain.Pipeline) {
	m.pipeline = &pipeline
	m.sourceRuns = nil
	m.visibleRuns = nil
	m.loaded = false
	m.filter.Clear()
	m.cursor.Reset()
	m.ResetPreview()
}

func (m *PipelineRunsViewModel) SetFavorites(favorites map[string]bool) {
	selected := m.GetSelectedRun()
	m.favorites = favorites
	m.rebuild()
	m.selectRun(selected)
}

func (m *PipelineRunsViewModel) TogglePreview() {
	m.split.Toggle()
}

func (m *PipelineRunsViewModel) IsPreviewOpen() bool {
	return m.split.IsOpen()
}

func (m *PipelineRunsViewModel) FocusNextPane() {
	m.split.FocusNext(len(m.detailPanes()))
}

func (m *PipelineRunsViewModel) FocusPrevPane() {
	m.split.FocusPrev()
}

func (m *PipelineRunsViewModel) IsPaneFocused() bool {
	return m.split.IsDetailFocused()
}

func (m *PipelineRunsViewModel) ScrollPane(delta int) {
	m.split.ScrollFocused(m.detailPanes(), delta, m.width, m.height)
}

// NextPreviewRequest returns the selected run when the open pane needs its stages fetched,
// and marks the request in flight so cursor movement does not refetch it.
func (m *PipelineRunsViewModel) NextPreviewRequest() *domain.PipelineRun {
	run := m.GetSelectedRun()
	if !m.split.IsOpen() || run == nil {
		return nil
	}
	key := run.FavoriteKey()
	if _, loaded := m.previewStages[key]; loaded || m.previewPending[key] {
		return nil
	}
	m.previewPending[key] = true
	return run
}

func (m *PipelineRunsViewModel) SetPreviewStages(runKey string, stages []domain.PipelineStage) {
	delete(m.previewPending, runKey)
	m.previewStages[runKey] = stages
}

func (m *PipelineRunsViewModel) ResetPreview() {
	m.previewStages = make(map[string][]domain.PipelineStage)
	m.previewPending = make(map[string]bool)
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
	m.selectRun(selected)
}

func (m *PipelineRunsViewModel) selectRun(run *domain.PipelineRun) {
	if run == nil {
		return
	}
	for i, visible := range m.visibleRuns {
		if visible.ID == run.ID {
			m.cursor.SetIndex(i, len(m.visibleRuns))
			return
		}
	}
}

// SelectRun moves the cursor to the run with key and reports whether it is listed.
func (m *PipelineRunsViewModel) SelectRun(key string) bool {
	for i, run := range m.visibleRuns {
		if run.FavoriteKey() == key {
			m.cursor.SetIndex(i, len(m.visibleRuns))
			return true
		}
	}
	return false
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
	m.split.ResetScroll()
}

func (m *PipelineRunsViewModel) MoveDown() {
	m.cursor.MoveDown(len(m.visibleRuns))
	m.split.ResetScroll()
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
	sort.SliceStable(filtered, func(i, j int) bool {
		return m.favorites[filtered[i].FavoriteKey()] && !m.favorites[filtered[j].FavoriteKey()]
	})
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
	listWidth := m.split.ListWidth(m.width)
	if m.split.IsOpen() {
		return []listColumn{
			{"Run", flexColumnWidth(listWidth, statusWidth+1, minRunWidth)},
			{"Status", statusWidth},
		}
	}
	fixed := statusWidth + branchWidth + requesterWidth + queuedWidth + durationWidth + columnSpacing
	return []listColumn{
		{"Run", flexColumnWidth(listWidth, fixed, minRunWidth)},
		{"Status", statusWidth},
		{"Branch", branchWidth},
		{"Requested by", requesterWidth},
		{"Queued", queuedWidth},
		{"Duration", durationWidth},
	}
}

func (m *PipelineRunsViewModel) View() string {
	var b strings.Builder

	pipelineName := "Runs"
	if m.pipeline != nil {
		pipelineName = PipelineDisplayName(*m.pipeline)
	}
	b.WriteString(renderListTitle(pipelinesTitle, pipelineName))
	b.WriteString("\n\n")

	switch {
	case !m.loaded:
		b.WriteString(listMutedStyle.Render("Loading runs..."))
		b.WriteString("\n")
	case len(m.visibleRuns) == 0 && m.filter.Text() != "":
		b.WriteString(listMutedStyle.Render("No runs match the filter."))
		b.WriteString("\n")
	case len(m.visibleRuns) == 0:
		b.WriteString(listMutedStyle.Render("This pipeline has no runs yet."))
		b.WriteString("\n")
	default:
		b.WriteString(renderListRows(m.columns(), len(m.visibleRuns), m.cursor, func(i int) listRow {
			run := m.visibleRuns[i]
			return runRow(run, m.favorites[run.FavoriteKey()])
		}))
	}

	if m.filter.IsActive() {
		b.WriteString("\n")
		b.WriteString(m.filter.View())
	}

	return m.split.Render(b.String(), m.detailPanes(), m.width, m.height)
}

func (m *PipelineRunsViewModel) detailPanes() []detailPane {
	return []detailPane{m.stagesPane()}
}

func (m *PipelineRunsViewModel) stagesPane() detailPane {
	run := m.GetSelectedRun()
	if run == nil {
		return detailPane{title: "Stages", lines: []string{listMutedStyle.Render("No run selected.")}}
	}
	title := fmt.Sprintf("Stages of %s", valueOrPlaceholder(run.Number))
	stages, loaded := m.previewStages[run.FavoriteKey()]
	if !loaded {
		return detailPane{title: title, lines: []string{listMutedStyle.Render("Loading stages...")}}
	}
	if len(stages) == 0 {
		return detailPane{title: title, lines: []string{listMutedStyle.Render("This run has no stages.")}}
	}
	lines := make([]string, 0, len(stages))
	for _, stage := range stages {
		line := renderDetailRow(stage.Status, stage.Name+"  "+PipelineStatusLabel(stage.Status))
		if stage.AwaitsApproval() {
			line += " " + approvalHintStyle.Render("needs approval")
		}
		lines = append(lines, line)
	}
	return detailPane{title: title, lines: lines}
}

func runRow(run domain.PipelineRun, isFavorite bool) listRow {
	return listRow{
		icon: renderPipelineStatusIcon(run.Status),
		values: []string{
			favoritePrefix(isFavorite) + valueOrPlaceholder(run.Number),
			PipelineStatusLabel(run.Status),
			valueOrPlaceholder(run.SourceBranch),
			valueOrPlaceholder(run.RequestedBy.Username),
			formatOptionalAge(run.QueuedAt),
			formatRunDuration(run),
		},
		style: listRowStyle,
	}
}
