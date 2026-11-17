package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	closedPRsLoadingMessage = "Loading closed pull requests..."
	closedPRsStatusDelay    = 4 * time.Second
)

type ClosedPRsLoadedMsg struct {
	prs            []domain.PullRequest
	failedPATCount int
}

func handleClosedPRsCommand(m Model, args []string) (Model, tea.Cmd) {
	return m.showClosedPRs()
}

// handleToggleClosedPRsKey switches the PR list between the open and the closed pull requests.
func handleToggleClosedPRsKey(m Model) (Model, tea.Cmd) {
	if m.prListView.IsShowingClosed() {
		m.prListView.ShowClosed(false)
		return m, m.loadPRListDetails()
	}
	return m.showClosedPRs()
}

// showClosedPRs opens the PR list on the closed pull requests, loading them the first time.
func (m Model) showClosedPRs() (Model, tea.Cmd) {
	if len(m.providers) == 0 && m.provider == nil {
		m.statusBar.SetMessage("No active PAT. Please select a PAT first.", true)
		return m, nil
	}
	m.prListView.ShowClosed(true)
	m = m.showPRList()
	if m.prListView.HasClosedPRs() {
		return m, m.loadPRListDetails()
	}
	return m.reloadClosedPRs()
}

func (m Model) reloadClosedPRs() (Model, tea.Cmd) {
	m.statusBar.SetMessage(closedPRsLoadingMessage, false)
	return m, m.loadClosedPRs()
}

func (m Model) loadClosedPRs() tea.Cmd {
	return func() tea.Msg {
		if len(m.providers) == 0 {
			pat, err := m.repository.GetActivePAT()
			if err != nil {
				return ErrorMsg{err: err}
			}
			prs, err := listClosedPullRequests(m.ctx, m.provider, *pat)
			if err != nil {
				return ErrorMsg{err: err}
			}
			return ClosedPRsLoadedMsg{prs: prs}
		}

		listed, failedPATCount, err := m.listForSelectedPATs(listClosedPullRequests)
		if err != nil {
			return ErrorMsg{err: err}
		}
		var prs []domain.PullRequest
		for _, result := range listed {
			prs = append(prs, result.prs...)
		}
		return ClosedPRsLoadedMsg{prs: prs, failedPATCount: failedPATCount}
	}
}

func (m Model) handleClosedPRsLoaded(msg ClosedPRsLoadedMsg) (Model, tea.Cmd) {
	m.prListView.SetClosedPRs(msg.prs)
	if msg.failedPATCount > 0 {
		m.statusBar.SetMessage(fmt.Sprintf("Loaded %d closed PRs (%d PAT(s) failed)", len(msg.prs), msg.failedPATCount), true)
	} else {
		m.statusBar.SetMessage(fmt.Sprintf("Loaded %d closed pull requests", len(msg.prs)), false)
	}
	return m, tea.Batch(clearStatusAfterDelay(closedPRsStatusDelay), m.loadPRListDetails())
}
