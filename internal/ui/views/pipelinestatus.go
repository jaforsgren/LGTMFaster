package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	pipelineStatusColumnWidth = 3
	selectedRowMarker         = "▸ "
	unselectedRowMarker       = "  "
	emptyCellPlaceholder      = "-"
)

var (
	pipelineHeaderStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	pipelineSelectedRowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Background(lipgloss.Color("#1F2937")).Bold(true)
	pipelineRowStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB"))
	pipelineMutedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Italic(true)
	pipelineTitleStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true)
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
	return lipgloss.NewStyle().Foreground(appearance.color).Bold(true).
		Render(padToWidth(appearance.icon, pipelineStatusColumnWidth))
}

func renderPipelineStatusLabel(status domain.PipelineStatus, width int) string {
	appearance := appearanceFor(status)
	return lipgloss.NewStyle().Foreground(appearance.color).
		Render(padToWidth(truncateString(appearance.label, width), width))
}

type listColumn struct {
	title string
	width int
}

func renderListHeader(columns []listColumn) string {
	cells := make([]string, len(columns))
	for i, column := range columns {
		cells[i] = padToWidth(truncateString(column.title, column.width), column.width)
	}
	return unselectedRowMarker + padToWidth("", pipelineStatusColumnWidth) + " " +
		pipelineHeaderStyle.Render(strings.Join(cells, " "))
}

func renderListRow(status domain.PipelineStatus, columns []listColumn, values []string, selected bool) string {
	cells := make([]string, len(columns))
	for i, column := range columns {
		cells[i] = padToWidth(truncateString(values[i], column.width), column.width)
	}

	marker := unselectedRowMarker
	style := pipelineRowStyle
	if selected {
		marker = selectedRowMarker
		style = pipelineSelectedRowStyle
	}
	return style.Render(marker) + renderPipelineStatusIcon(status) + " " + style.Render(strings.Join(cells, " "))
}

func renderScrollInfo(start, end, count int) string {
	return pipelineMutedStyle.Render(fmt.Sprintf("[%d-%d of %d]", start+1, end, count))
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

func formatOptionalAge(t time.Time) string {
	if t.IsZero() {
		return emptyCellPlaceholder
	}
	return formatAge(t)
}

func valueOrPlaceholder(value string) string {
	if value == "" {
		return emptyCellPlaceholder
	}
	return value
}

// flexColumnWidth sizes the single stretchy column so the row fills the terminal width.
func flexColumnWidth(totalWidth, fixedWidth, minWidth int) int {
	const rowChrome = len(unselectedRowMarker) + pipelineStatusColumnWidth + 1
	return max(minWidth, totalWidth-fixedWidth-rowChrome)
}
