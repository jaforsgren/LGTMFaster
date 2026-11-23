package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	mergeConflictRecheckInterval = 3 * time.Second
	maxMergeConflictRechecks     = 5
)

// mergeConflictCheck tracks the inspected PR while its provider is still computing mergeability.
type mergeConflictCheck struct {
	prKey    string
	rechecks int
}

// MergeConflictRecheckMsg asks for the PR again once the provider had time to finish computing.
type MergeConflictRecheckMsg struct {
	pr domain.PullRequest
}

// MergeConflictRecheckFailedMsg stops a check whose reload failed, so the badge does not stay on checking.
type MergeConflictRecheckFailedMsg struct {
	prKey string
}

// startMergeConflictCheck makes pr the one whose details are accepted and clears the previous PR's badge.
func (m Model) startMergeConflictCheck(pr domain.PullRequest) Model {
	m.mergeConflicts = mergeConflictCheck{prKey: pr.Key()}
	m.topBar.SetPRStatus(string(pr.Status), pr.MergeConflicts)
	return m
}

func (m Model) handlePRDetailLoaded(msg PRDetailLoadedMsg) (Model, tea.Cmd) {
	// A slow response for a previously opened PR must not overwrite the one on screen.
	if msg.pr.Key() != m.mergeConflicts.prKey {
		return m, nil
	}
	m.prInspect.SetPR(msg.pr)
	m.topBar.SetPRApproval(string(msg.pr.ApprovalStatus))
	m.updateLinkedRepoInfo(msg.pr)
	m.updateShortcuts()

	conflicts, recheck := m.nextMergeConflictStep(*msg.pr)
	m.topBar.SetPRStatus(string(msg.pr.Status), conflicts)
	if recheck == nil {
		m.mergeConflicts.rechecks = 0
		return m, m.loadInspectWorkItems()
	}
	m.mergeConflicts.rechecks++
	return m, tea.Batch(m.loadInspectWorkItems(), recheck)
}

// nextMergeConflictStep returns the status to show and, while still computing, the delayed recheck.
func (m Model) nextMergeConflictStep(pr domain.PullRequest) (domain.MergeConflictStatus, tea.Cmd) {
	if pr.Status != domain.PRStatusOpen || pr.MergeConflicts != domain.MergeConflictChecking {
		return pr.MergeConflicts, nil
	}
	if m.mergeConflicts.rechecks >= maxMergeConflictRechecks {
		return domain.MergeConflictUnknown, nil
	}
	return domain.MergeConflictChecking, tea.Tick(mergeConflictRecheckInterval, func(time.Time) tea.Msg {
		return MergeConflictRecheckMsg{pr: pr}
	})
}

func (m Model) handleMergeConflictRecheck(msg MergeConflictRecheckMsg) (Model, tea.Cmd) {
	if m.state != ViewPRInspect || msg.pr.Key() != m.mergeConflicts.prKey {
		return m, nil
	}
	load := m.loadPRDetail(msg.pr)
	prKey := msg.pr.Key()
	return m, func() tea.Msg {
		result := load()
		if _, failed := result.(ErrorMsg); failed {
			return MergeConflictRecheckFailedMsg{prKey: prKey}
		}
		return result
	}
}

func (m Model) handleMergeConflictRecheckFailed(msg MergeConflictRecheckFailedMsg) (Model, tea.Cmd) {
	if msg.prKey == m.mergeConflicts.prKey {
		m.topBar.SetMergeConflicts(domain.MergeConflictUnknown)
	}
	return m, nil
}
