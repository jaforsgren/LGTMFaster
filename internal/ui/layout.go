package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/ui/components"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

const (
	minContentHeight = 1
	filterHelp       = "type to filter | enter: apply | esc: close"
	commandHelp      = "tab: complete | enter: run | esc: cancel"
	paneFocusHelp    = "j/k: scroll | ←/esc: back to the list"
)

type sizedView interface {
	SetSize(width, height int)
}

// contentHeight is the rows between the top bar and the command and status bars.
func (m Model) contentHeight() int {
	return max(m.height-components.TopBarHeight-components.CommandBarHeight-components.StatusBarHeight, minContentHeight)
}

func (m Model) mainWindowContentSize() (int, int) {
	return views.MainWindowContentSize(m.width, m.contentHeight())
}

// resizeViews gives the views framed in the main window its inner size, and the overlays that
// draw their own box the whole content area.
func (m Model) resizeViews() {
	innerWidth, innerHeight := m.mainWindowContentSize()
	framed := []sizedView{
		m.patsView, m.prListView, m.prInspect, m.commentDetailView,
		m.pipelineListView, m.pipelineRunsView, m.pipelineRunView,
	}
	for _, view := range framed {
		view.SetSize(innerWidth, innerHeight)
	}

	overlays := []sizedView{
		m.reviewView, m.mergeView, m.inlineCommentView, m.descriptionEditView,
		m.logsView, m.repoLinkView, m.reposView, m.stageApprovalView, m.newPRView, m.extensionFilterView,
	}
	for _, view := range overlays {
		view.SetSize(m.width, m.contentHeight())
	}
}

func (m Model) renderContent() string {
	if overlay, ok := m.activeOverlay(); ok {
		return lipgloss.NewStyle().
			MaxWidth(m.width).
			MaxHeight(m.contentHeight()).
			Height(m.contentHeight()).
			Render(overlay)
	}

	innerWidth, innerHeight := m.mainWindowContentSize()
	switch m.state {
	case ViewPRList:
		return m.prListView.View()
	case ViewPipelineList:
		return m.pipelineListView.View()
	case ViewPipelineRuns:
		return m.pipelineRunsView.View()
	case ViewPipelineRun:
		return m.pipelineRunView.View()
	case ViewPRInspect:
		return views.RenderMainWindow(m.prInspect.View(), innerWidth, innerHeight)
	default:
		return views.RenderMainWindow(m.patsView.View(), innerWidth, innerHeight)
	}
}

// activeOverlay is the view shown in place of the current state's view, if any. The comment detail
// view has no box of its own, so it is framed like the state views.
func (m Model) activeOverlay() (string, bool) {
	innerWidth, innerHeight := m.mainWindowContentSize()
	switch {
	case m.logsView.IsActive():
		return m.logsView.View(), true
	case m.reviewView.IsActive():
		return m.reviewView.View(), true
	case m.mergeView.IsActive():
		return m.mergeView.View(), true
	case m.inlineCommentView.IsActive():
		return m.inlineCommentView.View(), true
	case m.commentDetailView.IsActive():
		return views.RenderMainWindow(m.commentDetailView.View(), innerWidth, innerHeight), true
	case m.descriptionEditView.IsActive():
		return m.descriptionEditView.View(), true
	case m.repoLinkView.IsActive():
		return m.repoLinkView.View(), true
	case m.reposView.IsActive():
		return m.reposView.View(), true
	case m.stageApprovalView.IsActive():
		return m.stageApprovalView.View(), true
	case m.newPRView.IsActive():
		return m.newPRView.View(), true
	case m.extensionFilterView.IsActive():
		return m.extensionFilterView.View(), true
	}
	return "", false
}

// helpLine is the status bar's second row: what the keys do in whatever currently has input.
func (m Model) helpLine() string {
	if argument, isPRArgument := m.prCommandArgument(); isPRArgument && m.commandBar.IsActive() {
		return m.branchSuggestionHelp(argument)
	}
	switch {
	case m.commandBar.IsActive():
		return commandHelp
	case m.isFiltering():
		return filterHelp
	case m.isPaneFocused():
		return paneFocusHelp
	}
	return m.commandRegistry.NavigationHelp(m.state)
}

func (m Model) isFiltering() bool {
	switch m.state {
	case ViewPRList:
		return m.prListView.IsFiltering()
	case ViewPipelineList:
		return m.pipelineListView.IsFiltering()
	case ViewPipelineRuns:
		return m.pipelineRunsView.IsFiltering()
	}
	return false
}
