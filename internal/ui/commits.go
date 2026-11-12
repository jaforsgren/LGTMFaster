package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/ui/components"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

const commitCommentsBlockedMessage = "Inline comments go on the pull request's diff; press q to leave the commit first"

type CommitsLoadedMsg struct {
	prKey   string
	commits []domain.Commit
	err     error
}

type CommitDiffLoadedMsg struct {
	prKey  string
	commit domain.Commit
	diff   *domain.Diff
	err    error
}

func prIdentifier(pr domain.PullRequest) domain.PRIdentifier {
	return domain.PRIdentifier{Repository: pr.Repository.FullName, Number: pr.Number}
}

// handleCommitsKey lists the inspected pull request's commits, fetching them the first time.
func handleCommitsKey(m Model) (Model, tea.Cmd) {
	pr := m.prInspect.GetPR()
	if pr == nil {
		return m, nil
	}
	m.prInspect.ShowCommits()
	m.topBar.SetView(components.ViewNamePRCommits)
	m.updateShortcuts()
	if m.prInspect.HasCommits() {
		return m, nil
	}

	provider := m.getProviderForPR(*pr)
	ctx, inspected := m.ctx, *pr
	return m, func() tea.Msg {
		commits, err := provider.ListCommits(ctx, prIdentifier(inspected))
		return CommitsLoadedMsg{prKey: inspected.Key(), commits: commits, err: err}
	}
}

func (m Model) handleCommitsLoaded(msg CommitsLoadedMsg) (Model, tea.Cmd) {
	if pr := m.prInspect.GetPR(); pr == nil || pr.Key() != msg.prKey {
		return m, nil
	}
	if msg.err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to load commits: %v", msg.err), true)
		return m, nil
	}
	m.prInspect.SetCommits(msg.commits)
	return m, nil
}

func (m Model) openSelectedCommit() (Model, tea.Cmd) {
	pr, commit := m.prInspect.GetPR(), m.prInspect.SelectedCommit()
	if pr == nil || commit == nil {
		return m, nil
	}

	provider := m.getProviderForPR(*pr)
	ctx, inspected, selected := m.ctx, *pr, *commit
	m.statusBar.SetMessage(fmt.Sprintf("Loading commit %s...", selected.ShortID()), false)
	return m, func() tea.Msg {
		diff, err := provider.GetCommitDiff(ctx, prIdentifier(inspected), selected.ID)
		return CommitDiffLoadedMsg{prKey: inspected.Key(), commit: selected, diff: diff, err: err}
	}
}

func (m Model) handleCommitDiffLoaded(msg CommitDiffLoadedMsg) (Model, tea.Cmd) {
	if pr := m.prInspect.GetPR(); pr == nil || pr.Key() != msg.prKey || m.prInspect.GetMode() != views.PRInspectModeCommits {
		return m, nil
	}
	if msg.err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to load commit %s: %v", msg.commit.ShortID(), msg.err), true)
		return m, nil
	}
	m.statusBar.ClearMessage()
	m.prInspect.ShowCommitDiff(msg.commit, msg.diff)
	m.topBar.SetView(components.ViewNamePRCommit)
	m.updateShortcuts()
	return m, nil
}

// stepBackInPRInspect goes one level up inside a pull request: from a commit's diff to the commits,
// from the commits to the diff, from the diff to the description. It reports false at the top.
func (m Model) stepBackInPRInspect() (Model, bool) {
	view := ""
	switch {
	case m.prInspect.ViewingCommit() != nil:
		m.prInspect.LeaveCommit()
		view = components.ViewNamePRCommits
	case m.prInspect.GetMode() == views.PRInspectModeCommits:
		m.prInspect.SwitchToDiff()
		view = components.ViewNamePRDiff
	case m.prInspect.GetMode() == views.PRInspectModeDiff:
		m.prInspect.SwitchToDescription()
		view = components.ViewNamePRDescription
	default:
		return m, false
	}
	m.topBar.SetView(view)
	m.updateShortcuts()
	return m, true
}
