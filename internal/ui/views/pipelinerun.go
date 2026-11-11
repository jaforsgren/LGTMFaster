package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	runDetailRows         = 6
	stagesHeadingRows     = 2
	pipelineRunChromeRows = listTitleRows + runDetailRows + stagesHeadingRows + listScrollInfoRows
	stageStatusWidth      = 22
	runDetailLabelWidth   = 14
	approvalHint          = "approval required: a approve / x reject"
	autoRefreshingHint    = "  auto-refreshing"
	pipelinesTitle        = "Pipelines"
)

var approvalHintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true)

type PipelineRunViewModel struct {
	run          *domain.PipelineRun
	stages       []domain.PipelineStage
	stagesLoaded bool
	isFavorite   bool
	cursor       listCursor
	width        int
	height       int
}

func NewPipelineRunView() *PipelineRunViewModel {
	return &PipelineRunViewModel{cursor: newListCursor()}
}

func (m *PipelineRunViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.cursor.SetVisibleRows(height - pipelineRunChromeRows)
}

func (m *PipelineRunViewModel) SetRun(run domain.PipelineRun) {
	isNewRun := m.run == nil || m.run.ID != run.ID || m.run.Project != run.Project
	m.run = &run
	if isNewRun {
		m.stages = nil
		m.stagesLoaded = false
		m.cursor.Reset()
	}
}

func (m *PipelineRunViewModel) SetFavorite(isFavorite bool) {
	m.isFavorite = isFavorite
}

func (m *PipelineRunViewModel) GetRun() *domain.PipelineRun {
	return m.run
}

// SetStages keeps the cursor on the same stage across refreshes; on first load it jumps to the
// first stage awaiting approval, since that is what the user most likely came here to act on.
func (m *PipelineRunViewModel) SetStages(stages []domain.PipelineStage) {
	selected := m.GetSelectedStage()
	isFirstLoad := !m.stagesLoaded
	m.stages = append([]domain.PipelineStage(nil), stages...)
	m.stagesLoaded = true

	target := -1
	for i, stage := range m.stages {
		if !isFirstLoad && selected != nil && stage.ID == selected.ID {
			target = i
			break
		}
		if isFirstLoad && stage.AwaitsApproval() {
			target = i
			break
		}
	}
	if target >= 0 {
		m.cursor.SetIndex(target, len(m.stages))
		return
	}
	m.cursor.Clamp(len(m.stages))
}

func (m *PipelineRunViewModel) GetSelectedStage() *domain.PipelineStage {
	idx := m.cursor.Index()
	if idx < 0 || idx >= len(m.stages) {
		return nil
	}
	return &m.stages[idx]
}

func (m *PipelineRunViewModel) PendingApprovalCount() int {
	count := 0
	for _, stage := range m.stages {
		if stage.AwaitsApproval() {
			count++
		}
	}
	return count
}

func (m *PipelineRunViewModel) MoveUp() {
	m.cursor.MoveUp()
}

func (m *PipelineRunViewModel) MoveDown() {
	m.cursor.MoveDown(len(m.stages))
}

func (m *PipelineRunViewModel) View() string {
	if m.run == nil {
		return RenderMainWindow(listMutedStyle.Render("No run selected."), m.width, m.height)
	}

	var b strings.Builder
	run := m.run

	if m.isFavorite {
		b.WriteString(favoriteStyle.Render(favoriteMarker))
	}
	b.WriteString(renderListTitle(pipelinesTitle, valueOrPlaceholder(run.PipelineName), fmt.Sprintf("Run %s", valueOrPlaceholder(run.Number))))
	if !run.Status.IsFinished() {
		b.WriteString(listMutedStyle.Render(autoRefreshingHint))
	}
	b.WriteString("\n\n")

	b.WriteString(renderDetailLabel("Pipeline", runDetailLabelWidth))
	b.WriteString(detailValueStyle.Render(valueOrPlaceholder(run.PipelineName)))
	b.WriteString("\n")
	for _, line := range runDetailLines(*run) {
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(listTitleStyle.Render("Stages"))
	b.WriteString("\n")

	switch {
	case !m.stagesLoaded:
		b.WriteString(listMutedStyle.Render("Loading stages..."))
		b.WriteString("\n")
	case len(m.stages) == 0:
		b.WriteString(listMutedStyle.Render("This run has no stages."))
		b.WriteString("\n")
	default:
		start, end := m.cursor.VisibleRange(len(m.stages))
		for i := start; i < end; i++ {
			b.WriteString(m.renderStage(m.stages[i], i == m.cursor.Index()))
			b.WriteString("\n")
		}
		if len(m.stages) > end-start {
			b.WriteString(renderScrollInfo(start, end, len(m.stages)))
			b.WriteString("\n")
		}
	}

	return RenderMainWindow(b.String(), m.width, m.height)
}

func (m *PipelineRunViewModel) renderStage(stage domain.PipelineStage, selected bool) string {
	const minStageNameWidth = 20
	nameWidth := flexColumnWidth(m.width, stageStatusWidth+len(approvalHint)+2, minStageNameWidth)

	marker := unselectedRowMarker
	nameStyle := listRowStyle
	if selected {
		marker = selectedRowMarker
		nameStyle = listSelectedRowStyle
	}

	line := nameStyle.Render(marker) +
		renderPipelineStatusIcon(stage.Status) + " " +
		nameStyle.Render(padToWidth(truncateString(stage.Name, nameWidth), nameWidth)) + " " +
		renderPipelineStatusLabel(stage.Status, stageStatusWidth)

	if stage.AwaitsApproval() {
		line += " " + approvalHintStyle.Render(approvalHint)
	}
	return line
}
