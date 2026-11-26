package views

import (
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type pipelineStatusAppearance struct {
	icon  string
	color lipgloss.Color
}

var pipelineStatusAppearances = map[domain.PipelineStatus]pipelineStatusAppearance{
	domain.PipelineStatusQueued:             {"○", lipgloss.Color("#9CA3AF")},
	domain.PipelineStatusRunning:            {"●", lipgloss.Color("#3B82F6")},
	domain.PipelineStatusWaitingForApproval: {"⏸", lipgloss.Color("#F59E0B")},
	domain.PipelineStatusSucceeded:          {"✓", lipgloss.Color("#10B981")},
	domain.PipelineStatusPartiallySucceeded: {"!", lipgloss.Color("#FBBF24")},
	domain.PipelineStatusFailed:             {"✗", lipgloss.Color("#EF4444")},
	domain.PipelineStatusCanceled:           {"⊘", lipgloss.Color("#6B7280")},
	domain.PipelineStatusSkipped:            {"»", lipgloss.Color("#6B7280")},
}

const (
	runStageColumnWidth = 20
	// runCommitColumnWidth caps the commit message, which goes last so it never pushes other
	// columns out; runCommitDetailWidth caps it where it has a line of its own.
	runCommitColumnWidth = 50
	runCommitDetailWidth = 72
)

var (
	runStageColumn  = listColumn{"Stage", runStageColumnWidth}
	runCommitColumn = listColumn{"Commit", runCommitColumnWidth}
)

func runStageLabel(run domain.PipelineRun) string {
	return valueOrPlaceholder(run.CurrentStage)
}

func runCommitLabel(run domain.PipelineRun) string {
	return valueOrPlaceholder(run.CommitMessage)
}

var unknownPipelineStatusAppearance = pipelineStatusAppearance{"?", lipgloss.Color("#6B7280")}

func appearanceFor(status domain.PipelineStatus) pipelineStatusAppearance {
	if appearance, ok := pipelineStatusAppearances[status]; ok {
		return appearance
	}
	return unknownPipelineStatusAppearance
}

func renderPipelineStatusIcon(status domain.PipelineStatus) string {
	appearance := appearanceFor(status)
	return renderListIcon(appearance.icon, appearance.color)
}

func renderPipelineStatusLabel(status domain.PipelineStatus, width int) string {
	appearance := appearanceFor(status)
	return lipgloss.NewStyle().Foreground(appearance.color).
		Render(padToWidth(truncateString(status.Label(), width), width))
}

// formatElapsed is how long something that started has run: until it finished, else until now.
func formatElapsed(startedAt, finishedAt time.Time) string {
	if startedAt.IsZero() {
		return emptyCellPlaceholder
	}
	end := finishedAt
	if end.IsZero() {
		end = time.Now()
	}
	return end.Sub(startedAt).Round(time.Second).String()
}
