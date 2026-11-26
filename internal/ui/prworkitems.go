package ui

import (
	"context"
	"errors"
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
)

// workItemIDLookupLimit bounds the lookups in flight, since a long PR list needs one per pull request.
const workItemIDLookupLimit = 8

// PRWorkItemIDsLoadedMsg carries the linked work items of the listed PRs, keyed by PR key; failed
// holds the PRs whose lookup failed.
type PRWorkItemIDsLoadedMsg struct {
	ids    map[string][]int
	failed []string
}

// loadPRListWorkItemIDs looks up the linked work items the PR list sorts or filters by.
func (m Model) loadPRListWorkItemIDs() tea.Cmd {
	prs := m.prListView.PendingWorkItemIDRequests()
	if len(prs) == 0 {
		return nil
	}
	providers := make([]domain.Provider, len(prs))
	for i, pr := range prs {
		providers[i] = m.getProviderForPR(pr)
	}
	ctx := m.ctx
	return func() tea.Msg {
		var (
			mu  sync.Mutex
			wg  sync.WaitGroup
			msg = PRWorkItemIDsLoadedMsg{ids: make(map[string][]int, len(prs))}
		)
		slots := make(chan struct{}, workItemIDLookupLimit)
		for i, pr := range prs {
			wg.Add(1)
			go func(pr domain.PullRequest, provider domain.Provider) {
				defer wg.Done()
				slots <- struct{}{}
				defer func() { <-slots }()

				ids, err := lookupWorkItemIDs(ctx, provider, pr)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					logger.LogError("LIST_PR_WORK_ITEM_IDS", fmt.Sprintf("%s#%d", pr.Repository.FullName, pr.Number), err)
					msg.failed = append(msg.failed, pr.Key())
					return
				}
				msg.ids[pr.Key()] = ids
			}(pr, providers[i])
		}
		wg.Wait()
		return msg
	}
}

func lookupWorkItemIDs(ctx context.Context, provider domain.Provider, pr domain.PullRequest) ([]int, error) {
	if provider == nil {
		return nil, errors.New(noProviderMessage)
	}
	return provider.ListLinkedWorkItemIDs(ctx, pr.Identifier())
}

func (m Model) handlePRWorkItemIDsLoaded(msg PRWorkItemIDsLoadedMsg) (Model, tea.Cmd) {
	m.prListView.SetWorkItemIDs(msg.ids, msg.failed)
	if len(msg.failed) > 0 {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to look up the work items of %d PR(s)", len(msg.failed)), true)
	}
	return m, nil
}
