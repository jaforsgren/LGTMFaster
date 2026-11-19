package ui

import (
	"fmt"

	"github.com/atotto/clipboard"
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

// handleYankPrefixKey starts a yank motion; the next key picks what is copied (yp: the PR link).
func handleYankPrefixKey(m Model) (Model, tea.Cmd) {
	m.yankPending = true
	return m, nil
}

// completeYank runs the yank motion key finishes and reports whether key was one.
func (m Model) completeYank(key string) (Model, tea.Cmd, bool) {
	m.yankPending = false
	if key != keyYankPRLink {
		return m, nil, false
	}
	pr := m.selectedPR()
	if pr == nil || pr.URL == "" {
		m.statusBar.SetMessage(noPRURLMessage, true)
		return m, nil, true
	}
	if err := clipboard.WriteAll(pr.URL); err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to copy: %v", err), true)
		return m, nil, true
	}
	m.statusBar.SetMessage(fmt.Sprintf("Copied link to #%d to clipboard", pr.Number), false)
	return m, nil, true
}
