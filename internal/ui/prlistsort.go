package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

const sortStatusDelay = 4 * time.Second

func (m Model) setPRSortMode(mode domain.PRSortMode) (Model, tea.Cmd) {
	if m.state != ViewPRList {
		return m, nil
	}
	m.prListView.SetSortMode(mode)
	saved := m.saveSortMode(views.PRSortModeLabel(mode), func(settings *domain.Settings) {
		settings.PRSortMode = mode
	})
	return m, tea.Batch(m.loadPRListDetails(), saved)
}

// saveSortMode stores a list's new sort mode in the settings and reports it in the status bar.
func (m Model) saveSortMode(label string, apply func(*domain.Settings)) tea.Cmd {
	settings, err := m.repository.GetSettings()
	if err == nil {
		apply(&settings)
		err = m.repository.SaveSettings(settings)
	}
	if err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Sorted by %s (not saved: %v)", label, err), true)
	} else {
		m.statusBar.SetMessage("Sorted by "+label, false)
	}
	return clearStatusAfterDelay(sortStatusDelay)
}
