package views

import (
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type pipelineStatusAppearance struct {
	icon  string
	label string
	color lipgloss.Color
}

var pipelineStatusAppearances = map[domain.PipelineStatus]pipelineStatusAppearance{
	domain.PipelineStatusQueued:             {"○", "queued", lipgloss.Color("#9CA3AF")},
	domain.PipelineStatusRunning:            {"●", "running", lipgloss.Color("#3B82F6")},
	domain.PipelineStatusWaitingForApproval: {"⏸", "waiting for approval", lipgloss.Color("#F59E0B")},
	domain.PipelineStatusSucceeded:          {"✓", "succeeded", lipgloss.Color("#10B981")},
	domain.PipelineStatusPartiallySucceeded: {"!", "partially succeeded", lipgloss.Color("#FBBF24")},
	domain.PipelineStatusFailed:             {"✗", "failed", lipgloss.Color("#EF4444")},
	domain.PipelineStatusCanceled:           {"⊘", "canceled", lipgloss.Color("#6B7280")},
	domain.PipelineStatusSkipped:            {"»", "skipped", lipgloss.Color("#6B7280")},
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

var unknownPipelineStatusAppearance = pipelineStatusAppearance{"?", "unknown", lipgloss.Color("#6B7280")}

func appearanceFor(status domain.PipelineStatus) pipelineStatusAppearance {
	if appearance, ok := pipelineStatusAppearances[status]; ok {
		return appearance
	}
	return unknownPipelineStatusAppearance
}

func PipelineStatusLabel(status domain.PipelineStatus) string {
	return appearanceFor(status).label
}

func renderPipelineStatusIcon(status domain.PipelineStatus) string {
	appearance := appearanceFor(status)
	return renderListIcon(appearance.icon, appearance.color)
}

func renderPipelineStatusLabel(status domain.PipelineStatus, width int) string {
	appearance := appearanceFor(status)
	return lipgloss.NewStyle().Foreground(appearance.color).
		Render(padToWidth(truncateString(appearance.label, width), width))
}

func formatRunDuration(run domain.PipelineRun) string {
	if run.StartedAt.IsZero() {
		return emptyCellPlaceholder
	}
	end := run.FinishedAt
	if end.IsZero() {
		end = time.Now()
	}
	return end.Sub(run.StartedAt).Round(time.Second).String()
}
