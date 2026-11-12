package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

const paneScrollStep = 1

// splitList is a list view with a details pane beside it.
type splitList interface {
	TogglePreview()
	IsPreviewOpen() bool
	FocusNextPane()
	FocusPrevPane()
	IsPaneFocused() bool
	ScrollPane(delta int)
}

func (m Model) activeSplitList() (splitList, bool) {
	switch m.state {
	case ViewPRList:
		return m.prListView, true
	case ViewPipelineList:
		return m.pipelineListView, true
	case ViewPipelineRuns:
		return m.pipelineRunsView, true
	case ViewPipelineLog:
		return m.pipelineLogView, true
	case ViewPRInspect:
		if m.prInspect.GetMode() == views.PRInspectModeDescription {
			return m.prInspect, true
		}
	}
	return nil, false
}

func (m Model) isPaneFocused() bool {
	list, ok := m.activeSplitList()
	return ok && list.IsPaneFocused()
}

func handleTogglePreviewKey(m Model) (Model, tea.Cmd) {
	list, ok := m.activeSplitList()
	if !ok {
		return m, nil
	}
	list.TogglePreview()
	m.statusBar.ClearMessage()
	switch m.state {
	case ViewPRList:
		return m, m.loadPRListComments()
	case ViewPRInspect:
		return m, m.loadInspectWorkItems()
	}
	return m, m.loadPipelinePreview()
}

// handleFocusPaneInKey moves focus right, from the list into the details pane, opening it first
// when it is closed.
func handleFocusPaneInKey(m Model) (Model, tea.Cmd) {
	list, ok := m.activeSplitList()
	if !ok {
		return m, nil
	}
	if !list.IsPreviewOpen() {
		return handleTogglePreviewKey(m)
	}
	list.FocusNextPane()
	return m, nil
}

func handleFocusPaneOutKey(m Model) (Model, tea.Cmd) {
	list, ok := m.activeSplitList()
	if !ok {
		return m, nil
	}
	list.FocusPrevPane()
	return m, nil
}

// scrollFocusedPane scrolls the focused details pane and reports whether one was focused.
func (m Model) scrollFocusedPane(delta int) bool {
	list, ok := m.activeSplitList()
	if !ok || !list.IsPaneFocused() {
		return false
	}
	list.ScrollPane(delta)
	return true
}
