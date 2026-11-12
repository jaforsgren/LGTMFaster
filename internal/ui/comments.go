package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

// PRCommentThreadsLoadedMsg carries a PR's comment thread count for the PR list's Info pane.
type PRCommentThreadsLoadedMsg struct {
	prKey   string
	threads domain.CommentThreads
	err     error
}

// handleCommentsLoaded shows the inspected PR's comments and how many of their threads are resolved.
func (m Model) handleCommentsLoaded(msg CommentsLoadedMsg) (Model, tea.Cmd) {
	m.prInspect.SetComments(msg.comments)
	threads := domain.SummarizeThreads(msg.comments)
	m.prListView.SetCommentThreads(msg.prKey, threads)
	if pr := m.prInspect.GetPR(); pr != nil && pr.Key() == msg.prKey {
		m.topBar.SetCommentThreads(&threads)
	}
	return m, nil
}

// loadPRListComments counts the comment threads of the PR selected in the list while its Info pane
// is open.
func (m Model) loadPRListComments() tea.Cmd {
	pr := m.prListView.PendingCommentRequest()
	if pr == nil {
		return nil
	}
	provider := m.getProviderForPR(*pr)
	ctx, selected := m.ctx, *pr
	return func() tea.Msg {
		comments, err := provider.GetComments(ctx, prIdentifier(selected))
		return PRCommentThreadsLoadedMsg{prKey: selected.Key(), threads: domain.SummarizeThreads(comments), err: err}
	}
}

func (m Model) handlePRCommentThreadsLoaded(msg PRCommentThreadsLoadedMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		m.prListView.CommentThreadsFailed(msg.prKey)
		m.statusBar.SetMessage(fmt.Sprintf("Failed to count comments: %v", msg.err), true)
		return m, nil
	}
	m.prListView.SetCommentThreads(msg.prKey, msg.threads)
	return m, nil
}
