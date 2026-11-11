package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	pipelineRunChromeRows = 18
	stageStatusWidth      = 22
	runDetailLabelWidth   = 14
	approvalHint          = "approval required: a approve / x reject"
)

var (
	pipelineRunLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	pipelineRunValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	approvalHintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true)
)

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
		return pipelineMutedStyle.Render("No run selected.")
	}

	var b strings.Builder
	run := m.run

	if m.isFavorite {
		b.WriteString(favoriteStyle.Render(favoriteMarker))
	}
	b.WriteString(pipelineTitleStyle.Render(fmt.Sprintf("Run %s", valueOrPlaceholder(run.Number))))
	b.WriteString("\n\n")

	details := []struct {
		label string
		value string
	}{
		{"Pipeline", valueOrPlaceholder(run.PipelineName)},
		{"Status", renderPipelineStatusLabel(run.Status, stageStatusWidth)},
		{"Branch", valueOrPlaceholder(run.SourceBranch)},
		{"Requested by", valueOrPlaceholder(run.RequestedBy.Username)},
		{"Queued", formatOptionalAge(run.QueuedAt)},
		{"Duration", formatRunDuration(*run)},
	}
	for _, detail := range details {
		b.WriteString(pipelineRunLabelStyle.Render(padToWidth(detail.label+":", runDetailLabelWidth)))
		b.WriteString(pipelineRunValueStyle.Render(detail.value))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(pipelineTitleStyle.Render("Stages"))
	b.WriteString("\n")

	switch {
	case !m.stagesLoaded:
		b.WriteString(pipelineMutedStyle.Render("Loading stages..."))
		b.WriteString("\n")
	case len(m.stages) == 0:
		b.WriteString(pipelineMutedStyle.Render("This run has no stages."))
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

	b.WriteString("\n")
	b.WriteString(pipelineMutedStyle.Render(m.helpText()))
	return b.String()
}

func (m *PipelineRunViewModel) renderStage(stage domain.PipelineStage, selected bool) string {
	const minStageNameWidth = 20
	nameWidth := flexColumnWidth(m.width, stageStatusWidth+len(approvalHint)+2, minStageNameWidth)

	marker := unselectedRowMarker
	nameStyle := pipelineRowStyle
	if selected {
		marker = selectedRowMarker
		nameStyle = pipelineSelectedRowStyle
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

func (m *PipelineRunViewModel) helpText() string {
	help := "j/k: Select stage | a: Approve | x: Reject | f: Favorite | ctrl+o: Open run | r: Refresh | q: Back"
	if m.run != nil && !m.run.Status.IsFinished() {
		help += " | auto-refreshing"
	}
	return help
}
