package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

const (
	splitTestWidth  = 140
	splitTestHeight = 30
)

func testDetailLines(count int) []string {
	lines := make([]string, count)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i)
	}
	return lines
}

func TestSplitView_RendersAtTheMainWindowSize(t *testing.T) {
	split := splitView{}
	panes := []detailPane{
		{title: "Info", lines: testDetailLines(4), fitContent: true},
		{title: "Description", lines: testDetailLines(200)},
	}
	closed := split.Render("list", panes, splitTestWidth, splitTestHeight)

	split.Toggle()
	open := split.Render("list", panes, splitTestWidth, splitTestHeight)

	if lipgloss.Height(open) != lipgloss.Height(closed) || lipgloss.Width(open) != lipgloss.Width(closed) {
		t.Errorf("expected the split to match the main window, got %dx%d vs %dx%d",
			lipgloss.Width(open), lipgloss.Height(open), lipgloss.Width(closed), lipgloss.Height(closed))
	}
}

func TestDetailHeights_FitContentPaneTakesOnlyWhatItNeeds(t *testing.T) {
	outerHeight := 40
	panes := []detailPane{
		{lines: testDetailLines(3), fitContent: true},
		{lines: testDetailLines(100)},
	}

	heights := detailHeights(outerHeight, panes)

	if heights[0] != len(panes[0].lines)+detailPaneTitleRows {
		t.Errorf("expected the fitted pane to hold exactly its lines and title, got %d", heights[0])
	}
	total := 0
	for _, height := range heights {
		total += outerSize(height, paneVerticalPadding)
	}
	if total != outerHeight {
		t.Errorf("expected the panes to fill %d rows, got %d", outerHeight, total)
	}
}

func TestDetailHeights_FitContentPaneIsCappedAtAnEvenShare(t *testing.T) {
	outerHeight := 30
	panes := []detailPane{
		{lines: testDetailLines(100), fitContent: true},
		{lines: testDetailLines(100)},
	}

	heights := detailHeights(outerHeight, panes)

	if heights[0] != innerSize(outerHeight/len(panes), paneVerticalPadding) {
		t.Errorf("expected the fitted pane to be capped at an even share, got %d", heights[0])
	}
}

func TestSplitView_FocusAndScroll(t *testing.T) {
	split := splitView{}
	panes := []detailPane{{title: "Description", lines: testDetailLines(200)}}

	split.FocusNext(len(panes))
	if split.IsDetailFocused() {
		t.Fatal("expected a closed split not to take focus")
	}

	split.Toggle()
	split.FocusNext(len(panes))
	split.FocusNext(len(panes))
	if split.focus != len(panes) {
		t.Fatalf("expected focus to stop at the last pane, got %d", split.focus)
	}

	split.ScrollFocused(panes, -1, splitTestWidth, splitTestHeight)
	if split.scroll[0] != 0 {
		t.Errorf("expected scrolling to stop at the top, got %d", split.scroll[0])
	}
	split.ScrollFocused(panes, 1000, splitTestWidth, splitTestHeight)
	view := split.Render("list", panes, splitTestWidth, splitTestHeight)
	if !strings.Contains(view, "line 199") {
		t.Error("expected scrolling to the end to show the last line")
	}

	split.FocusPrev()
	split.FocusPrev()
	if split.IsDetailFocused() {
		t.Error("expected focus to return to the list")
	}
}

func TestRenderDetailPaneContent_SummarisesHiddenLines(t *testing.T) {
	content := renderDetailPaneContent(detailPane{title: "T", lines: testDetailLines(10)}, 0, splitTestWidth, 6)

	if !strings.Contains(content, "… 7 more") {
		t.Errorf("expected a count of the hidden lines, got %q", content)
	}
}
