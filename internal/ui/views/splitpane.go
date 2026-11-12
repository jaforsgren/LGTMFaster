package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	splitListWidthPercent = 40
	percentScale          = 100
	detailPaneTitleRows   = 2
	splitFocusList        = 0
	favoriteMarker        = "★ "
	notFavoriteMarker     = "  "
)

var favoriteStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))

// detailPane is one of the stacked right-hand panes of a split view. A fitContent pane is only as
// tall as its lines need, up to an even share, leaving the rest to the other panes.
type detailPane struct {
	title      string
	lines      []string
	fitContent bool
}

// splitView renders a list next to stacked detail panes. Focus moves right from the list into the
// panes so a long pane can be scrolled without moving the list cursor.
type splitView struct {
	open   bool
	focus  int
	scroll map[int]int
}

type splitLayout struct {
	listWidth     int
	listHeight    int
	detailWidth   int
	detailHeights []int
}

// newSplitLayout divides the main window so that the bordered panes together occupy exactly the
// same outer size. Widths and heights are inner sizes, like the ones passed to SetSize.
func newSplitLayout(innerWidth, innerHeight int, panes []detailPane) splitLayout {
	listWidth, detailWidth := splitWidths(innerWidth)
	return splitLayout{
		listWidth:     listWidth,
		listHeight:    innerHeight,
		detailWidth:   detailWidth,
		detailHeights: detailHeights(outerSize(innerHeight, paneVerticalPadding), panes),
	}
}

func splitWidths(innerWidth int) (int, int) {
	outerWidth := outerSize(innerWidth, paneHorizontalPadding)
	listOuterWidth := outerWidth * splitListWidthPercent / percentScale
	return innerSize(listOuterWidth, paneHorizontalPadding), innerSize(outerWidth-listOuterWidth, paneHorizontalPadding)
}

func detailHeights(outerHeight int, panes []detailPane) []int {
	if len(panes) == 0 {
		return nil
	}
	evenShare := outerHeight / len(panes)
	outerHeights := make([]int, len(panes))
	remaining := outerHeight
	flexible := 0
	for i, pane := range panes {
		if !pane.fitContent {
			flexible++
			continue
		}
		needed := outerSize(len(pane.lines)+detailPaneTitleRows, paneVerticalPadding)
		outerHeights[i] = min(needed, evenShare)
		remaining -= outerHeights[i]
	}

	// The last flexible pane takes the rounding remainder, so the column is exactly outerHeight.
	for i, pane := range panes {
		if pane.fitContent {
			continue
		}
		share := remaining / flexible
		if flexible == 1 {
			share = remaining
		}
		outerHeights[i] = share
		remaining -= share
		flexible--
	}
	if remaining > 0 {
		outerHeights[len(outerHeights)-1] += remaining
	}

	heights := make([]int, len(panes))
	for i, height := range outerHeights {
		heights[i] = innerSize(height, paneVerticalPadding)
	}
	return heights
}

func (s *splitView) Toggle() {
	s.open = !s.open
	s.focus = splitFocusList
	s.ResetScroll()
}

func (s *splitView) Close() {
	if s.open {
		s.Toggle()
	}
}

func (s *splitView) IsOpen() bool {
	return s.open
}

func (s *splitView) IsDetailFocused() bool {
	return s.open && s.focus != splitFocusList
}

// FocusNext moves focus one pane to the right, stopping at the last pane.
func (s *splitView) FocusNext(paneCount int) {
	if s.open && s.focus < paneCount {
		s.focus++
	}
}

// FocusPrev moves focus one pane to the left, stopping at the list.
func (s *splitView) FocusPrev() {
	if s.IsDetailFocused() {
		s.focus--
	}
}

func (s *splitView) ResetScroll() {
	s.scroll = make(map[int]int)
}

// ListWidth is the inner width available to the list, whether or not the panes are open.
func (s *splitView) ListWidth(innerWidth int) int {
	if !s.open {
		return innerWidth
	}
	listWidth, _ := splitWidths(innerWidth)
	return listWidth
}

func (s *splitView) ScrollFocused(panes []detailPane, delta, innerWidth, innerHeight int) {
	if !s.IsDetailFocused() || s.focus > len(panes) {
		return
	}
	layout := newSplitLayout(innerWidth, innerHeight, panes)
	index := s.focus - 1
	maxScroll := max(len(panes[index].lines)-paneBodyRows(layout.detailHeights[index]), 0)
	if s.scroll == nil {
		s.ResetScroll()
	}
	s.scroll[index] = min(max(s.scroll[index]+delta, 0), maxScroll)
}

// Render frames list alone in the main window while the panes are closed.
func (s *splitView) Render(list string, panes []detailPane, innerWidth, innerHeight int) string {
	if !s.open {
		return RenderMainWindow(list, innerWidth, innerHeight)
	}

	layout := newSplitLayout(innerWidth, innerHeight, panes)
	rendered := make([]string, len(panes))
	for i, pane := range panes {
		borderColor := detailPaneBorderColor
		if s.focus == i+1 {
			borderColor = focusedPaneBorderColor
		}
		content := renderDetailPaneContent(pane, s.scroll[i], layout.detailWidth, layout.detailHeights[i])
		rendered[i] = renderPane(content, layout.detailWidth, layout.detailHeights[i], borderColor)
	}

	listPane := renderPane(list, layout.listWidth, layout.listHeight, mainWindowBorderColor)
	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, lipgloss.JoinVertical(lipgloss.Left, rendered...))
}

func paneBodyRows(innerHeight int) int {
	return max(innerHeight-detailPaneTitleRows, minPaneSize)
}

// renderDetailPaneContent shows the pane from scroll on, replacing the last row with a count of the
// rows below it so a cut-off pane is never mistaken for a complete one.
func renderDetailPaneContent(pane detailPane, scroll, width, innerHeight int) string {
	rows := paneBodyRows(innerHeight)
	scroll = min(scroll, max(len(pane.lines)-rows, 0))
	visible := pane.lines[scroll:]
	if len(visible) > rows {
		hidden := len(visible) - rows + 1
		visible = append(visible[:rows-1:rows-1], listMutedStyle.Render(fmt.Sprintf("… %d more", hidden)))
	}

	lines := make([]string, 0, len(visible)+detailPaneTitleRows)
	lines = append(lines, listTitleStyle.Render(truncateString(pane.title, width)), "")
	lines = append(lines, visible...)
	return strings.Join(lines, "\n")
}

func favoritePrefix(isFavorite bool) string {
	if isFavorite {
		return favoriteMarker
	}
	return notFavoriteMarker
}

func renderDetailRow(status domain.PipelineStatus, text string) string {
	return renderPipelineStatusIcon(status) + " " + listRowStyle.Render(text)
}
