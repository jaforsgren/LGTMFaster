package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/ui/components"
)

type DashboardChecksLoadedMsg struct {
	prKey  string
	checks domain.Checks
	err    error
}

// openDashboard shows the dashboard over the loaded pull requests, loading them first if needed.
func (m Model) openDashboard() (Model, tea.Cmd) {
	if len(m.providers) == 0 && m.provider == nil {
		m.statusBar.SetMessage("No active PAT. Please select a PAT first.", true)
		return m, nil
	}
	m.state = ViewDashboard
	m.topBar.SetView(components.ViewNameDashboard)
	m.updateShortcuts()
	m.dashboardView.SetPRs(m.prListView.AllPRs())
	if len(m.prListView.AllPRs()) == 0 && !m.loadingState.IsLoading {
		return m, m.loadPRsWithCache()
	}
	return m, m.loadDashboardChecks()
}

// loadDashboardChecks fetches the checks of the user's pull requests the dashboard has none for.
func (m Model) loadDashboardChecks() tea.Cmd {
	var cmds []tea.Cmd
	for _, pr := range m.dashboardView.PendingChecks() {
		provider := m.getProviderForPR(pr)
		ctx, target := m.ctx, pr
		cmds = append(cmds, func() tea.Msg {
			checks, err := provider.GetChecks(ctx, prIdentifier(target))
			return DashboardChecksLoadedMsg{prKey: target.Key(), checks: checks, err: err}
		})
	}
	return tea.Batch(cmds...)
}

func (m Model) handleDashboardChecksLoaded(msg DashboardChecksLoadedMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		m.dashboardView.ChecksFailed(msg.prKey)
		m.statusBar.SetMessage(fmt.Sprintf("Failed to load checks: %v", msg.err), true)
		return m, nil
	}
	m.dashboardView.SetChecks(msg.prKey, msg.checks)
	return m, nil
}

// showPRResults puts freshly loaded pull requests on screen: into the dashboard while it is open,
// else into the PR list.
func (m Model) showPRResults() (Model, tea.Cmd) {
	m.dashboardView.SetPRs(m.prListView.AllPRs())
	if m.state == ViewDashboard {
		return m, m.loadDashboardChecks()
	}
	return m.showPRList(), nil
}

func (m Model) showPRList() Model {
	m.state = ViewPRList
	m.topBar.SetView(components.ViewNamePRList)
	m.updateShortcuts()
	return m
}

func handleDashboardNextPaneKey(m Model) (Model, tea.Cmd) {
	m.dashboardView.FocusNext()
	return m, nil
}

func handleDashboardPrevPaneKey(m Model) (Model, tea.Cmd) {
	m.dashboardView.FocusPrev()
	return m, nil
}

func handleDashboardRightKey(m Model) (Model, tea.Cmd) {
	m.dashboardView.FocusRight()
	return m, nil
}

func handleDashboardLeftKey(m Model) (Model, tea.Cmd) {
	m.dashboardView.FocusLeft()
	return m, nil
}

func (m Model) refreshDashboard() (Model, tea.Cmd) {
	m.dashboardView.ResetChecks()
	m.loadingState = LoadingState{}
	m.prCache = nil
	return m, m.loadPRsStreaming()
}

func handleDashboardKey(m Model) (Model, tea.Cmd) {
	return m.openDashboard()
}

func handleDashboardCommand(m Model, args []string) (Model, tea.Cmd) {
	return m.openDashboard()
}
