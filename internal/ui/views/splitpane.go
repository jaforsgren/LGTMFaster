package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	splitPaneListRatio   = 0.55
	splitPaneGap         = 1
	splitPaneBorderWidth = 4
	minSplitPaneWidth    = 20
	favoriteMarker       = "★ "
	notFavoriteMarker    = "  "
)

var (
	detailPaneBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#374151")).
				Padding(0, 1)
	favoriteStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))
)

type splitPane struct {
	open bool
}

func (p *splitPane) Toggle() {
	p.open = !p.open
}

func (p *splitPane) IsOpen() bool {
	return p.open
}

// ListWidth is the width available to the list column, whether or not the pane is open.
func (p *splitPane) ListWidth(totalWidth int) int {
	if !p.open {
		return totalWidth
	}
	return int(float64(totalWidth) * splitPaneListRatio)
}

func (p *splitPane) detailWidth(totalWidth int) int {
	return max(minSplitPaneWidth, totalWidth-p.ListWidth(totalWidth)-splitPaneGap-splitPaneBorderWidth)
}

func (p *splitPane) Render(list, detailTitle string, detailLines []string, totalWidth, maxLines int) string {
	if !p.open {
		return list
	}

	if len(detailLines) > maxLines {
		hidden := len(detailLines) - maxLines + 1
		detailLines = append(detailLines[:maxLines-1:maxLines-1], pipelineMutedStyle.Render(fmt.Sprintf("… %d more", hidden)))
	}

	width := p.detailWidth(totalWidth)
	lines := make([]string, 0, len(detailLines)+2)
	lines = append(lines, pipelineTitleStyle.Render(truncateString(detailTitle, width)), "")
	for _, line := range detailLines {
		lines = append(lines, lipgloss.NewStyle().MaxWidth(width).Render(line))
	}
	detail := detailPaneBorderStyle.Width(width).Render(strings.Join(lines, "\n"))

	listColumn := lipgloss.NewStyle().Width(p.ListWidth(totalWidth)).Render(list)
	return lipgloss.JoinHorizontal(lipgloss.Top, listColumn, strings.Repeat(" ", splitPaneGap), detail)
}

func favoritePrefix(isFavorite bool) string {
	if isFavorite {
		return favoriteMarker
	}
	return notFavoriteMarker
}

func renderDetailRow(status domain.PipelineStatus, text string) string {
	return renderPipelineStatusIcon(status) + " " + pipelineRowStyle.Render(text)
}
