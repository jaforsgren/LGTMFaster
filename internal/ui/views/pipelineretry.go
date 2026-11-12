package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	pipelineRetryHelp = "↑/↓: choose | enter: retry | esc: cancel"
	// overlayBoxInset is the width an overlay box's border and padding take.
	overlayBoxInset = 4
)

var pipelineRetryBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(approveColor).
	Padding(1, 2)

// PipelineRetryViewModel asks how much of a run to run again.
type PipelineRetryViewModel struct {
	run      domain.PipelineRun
	stage    domain.PipelineStage
	scopes   []domain.PipelineRetryScope
	selected int
	active   bool
	width    int
	height   int
}

func NewPipelineRetryView() *PipelineRetryViewModel {
	return &PipelineRetryViewModel{}
}

func (m *PipelineRetryViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Activate offers the retries the run allows; selectedStage is the stage selected in the run, or nil.
func (m *PipelineRetryViewModel) Activate(run domain.PipelineRun, selectedStage *domain.PipelineStage) {
	m.run = run
	m.stage = domain.PipelineStage{}
	if selectedStage != nil {
		m.stage = *selectedStage
	}
	m.scopes = domain.RetryScopes(run, selectedStage)
	m.selected = 0
	m.active = true
}

func (m *PipelineRetryViewModel) Deactivate() {
	m.active = false
}

func (m *PipelineRetryViewModel) IsActive() bool {
	return m.active
}

func (m *PipelineRetryViewModel) MoveUp() {
	m.selected = max(m.selected-1, 0)
}

func (m *PipelineRetryViewModel) MoveDown() {
	m.selected = min(m.selected+1, len(m.scopes)-1)
}

func (m *PipelineRetryViewModel) GetRun() domain.PipelineRun {
	return m.run
}

func (m *PipelineRetryViewModel) GetRetry() domain.PipelineRetry {
	return domain.PipelineRetry{Run: m.run.Identifier(), Scope: m.scopes[m.selected], Stage: m.stage}
}

// RetryScopeLabel describes a retry of the given stage.
func RetryScopeLabel(scope domain.PipelineRetryScope, stage domain.PipelineStage) string {
	switch scope {
	case domain.PipelineRetryFailedJobs:
		return "Retry failed jobs"
	case domain.PipelineRetryStage:
		return fmt.Sprintf("Retry stage '%s'", stage.Name)
	case domain.PipelineRetryRerun:
		return "Rerun the whole pipeline (new run)"
	default:
		return string(scope)
	}
}

func (m *PipelineRetryViewModel) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder
	b.WriteString(listTitleStyle.Render(fmt.Sprintf("Retry run %s of %s", valueOrPlaceholder(m.run.Number), valueOrPlaceholder(m.run.PipelineName))))
	b.WriteString("\n\n")
	for i, scope := range m.scopes {
		marker, style := unselectedRowMarker, listRowStyle
		if i == m.selected {
			marker, style = selectedRowMarker, listSelectedRowStyle
		}
		b.WriteString(style.Render(marker + RetryScopeLabel(scope, m.stage)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(listMutedStyle.Render(pipelineRetryHelp))
	return pipelineRetryBoxStyle.Width(max(m.width-overlayBoxInset, minPaneSize)).Render(b.String())
}
