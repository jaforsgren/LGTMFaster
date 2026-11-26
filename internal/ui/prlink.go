package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const noPRURLMessage = "No PR URL available"

// selectedPR is the pull request the current view points at: the selected one in a list, or the
// inspected one.
func (m Model) selectedPR() *domain.PullRequest {
	switch m.state {
	case ViewPRList:
		return m.prListView.GetSelectedPR()
	case ViewPRInspect:
		return m.prInspect.GetPR()
	case ViewDashboard:
		return m.dashboardView.GetSelectedPR()
	}
	return nil
}

// handleYankPrefixKey starts a yank motion; the next key picks what is copied (yp: the PR link;
// in the pipeline logs yy: the step's log, yY: the run's logs).
func handleYankPrefixKey(m Model) (Model, tea.Cmd) {
	m.pendingPrefix = keyYankPrefix
	return m, nil
}

// completeYank runs the yank motion key finishes and reports whether key was one.
func (m Model) completeYank(key string) (Model, tea.Cmd, bool) {
	var cmd tea.Cmd
	switch {
	case m.state == ViewPipelineLog && key == keyYankStepLog:
		m = m.yankStepLog()
	case m.state == ViewPipelineLog && key == keyYankRunLogs:
		m, cmd = m.yankRunLogs()
	case m.state != ViewPipelineLog && key == keyYankPRLink:
		m = m.yankPRLink()
	default:
		return m, nil, false
	}
	return m, cmd, true
}

func (m Model) yankPRLink() Model {
	pr := m.selectedPR()
	if pr == nil || pr.URL == "" {
		m.statusBar.SetMessage(noPRURLMessage, true)
		return m
	}
	return m.copyToClipboard(pr.URL, fmt.Sprintf("Copied link to #%d to clipboard", pr.Number))
}

// copyToClipboard reports done once the text is on the clipboard, or why it is not.
func (m Model) copyToClipboard(text, done string) Model {
	if err := writeClipboard(text); err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to copy: %v", err), true)
		return m
	}
	m.statusBar.SetMessage(done, false)
	return m
}
