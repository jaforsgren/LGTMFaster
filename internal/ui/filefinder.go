package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

// handleDiffLeaderKey is space in PR inspect: a second space in the diff opens the file finder.
func handleDiffLeaderKey(m Model) (Model, tea.Cmd) {
	if m.prInspect.GetMode() != views.PRInspectModeDiff || !m.prInspect.PressSpace() {
		return m, nil
	}
	files, current := m.prInspect.DiffFiles()
	if len(files) == 0 {
		m.statusBar.SetMessage("No files in the diff", true)
		return m, nil
	}
	m.fileFinderView.Activate(files, current)
	return m, nil
}

func (m Model) handleFileFinderKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	action, cmd := m.fileFinderView.HandleKey(msg)
	switch action {
	case views.FileFinderActionOpen:
		if index, ok := m.fileFinderView.SelectedFile(); ok {
			m.prInspect.GoToFile(index)
		}
		m.fileFinderView.Deactivate()
	case views.FileFinderActionCancel:
		m.fileFinderView.Deactivate()
	}
	return m, cmd
}
