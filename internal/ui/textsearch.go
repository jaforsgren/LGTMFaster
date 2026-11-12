package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

const noSearchMatchMessage = "No more matches"

// textSearch is the vim-style "/" search of the views that show text: the pipeline logs and the
// PR diff. Enter keeps the query for n/N, esc clears it.
type textSearch interface {
	ActivateSearch()
	DeactivateSearch()
	ClearSearch()
	IsSearching() bool
	SearchQuery() string
	UpdateSearch(msg tea.Msg) tea.Cmd
	NextMatch() bool
	PrevMatch() bool
}

func (m Model) activeTextSearch() (textSearch, bool) {
	switch m.state {
	case ViewPipelineLog:
		return m.pipelineLogView, true
	case ViewPRInspect:
		if m.prInspect.GetMode() == views.PRInspectModeDiff {
			return m.prInspect, true
		}
	}
	return nil, false
}

func (m Model) isTextSearching() bool {
	search, ok := m.activeTextSearch()
	return ok && search.IsSearching()
}

func (m Model) handleTextSearchInputKey(search textSearch, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case keyEnter:
		search.DeactivateSearch()
		return m, nil
	case keyEsc:
		search.ClearSearch()
		return m, nil
	default:
		return m, search.UpdateSearch(msg)
	}
}

func handleSearchKey(m Model) (Model, tea.Cmd) {
	if search, ok := m.activeTextSearch(); ok {
		search.ActivateSearch()
	}
	return m, nil
}

func handleNextMatchKey(m Model) (Model, tea.Cmd) {
	search, ok := m.activeTextSearch()
	if !ok {
		return m, nil
	}
	return m.jumpOrReport(search.NextMatch, noSearchMatchMessage)
}

func handlePrevMatchKey(m Model) (Model, tea.Cmd) {
	search, ok := m.activeTextSearch()
	if !ok {
		return m, nil
	}
	return m.jumpOrReport(search.PrevMatch, noSearchMatchMessage)
}

// jumpOrReport runs a search or error jump and says so when there was nothing to jump to.
func (m Model) jumpOrReport(jump func() bool, nothingFound string) (Model, tea.Cmd) {
	if jump() {
		m.statusBar.ClearMessage()
		return m, nil
	}
	m.statusBar.SetMessage(nothingFound, true)
	return m, clearStatusAfterDelay(4 * time.Second)
}
