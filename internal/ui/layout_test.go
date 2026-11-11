package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	layoutTestWidth  = 160
	layoutTestHeight = 45
)

func sizedTestModel(t *testing.T, width, height int) Model {
	t.Helper()
	m := NewModel(&mockRepository{pats: map[string]*domain.PAT{}})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated.(Model)
}

func layoutTestPRs() []domain.PullRequest {
	prs := make([]domain.PullRequest, 80)
	for i := range prs {
		prs[i] = domain.PullRequest{
			Number:      i + 1,
			Title:       strings.Repeat("a very long pull request title ", 5),
			Description: strings.Repeat("description line\n", 100),
			Repository:  domain.Repo{FullName: "owner/repo"},
			Category:    domain.PRCategoryAuthored,
			CreatedAt:   time.Now(),
		}
	}
	return prs
}

func assertFillsTerminal(t *testing.T, name string, m Model, width, height int) {
	t.Helper()
	view := m.View()
	if got := lipgloss.Height(view); got != height {
		t.Errorf("%s: expected the view to be %d rows, got %d", name, height, got)
	}
	for i, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Errorf("%s: line %d is %d wide, terminal is %d", name, i, got, width)
			return
		}
	}
}

func TestView_FillsTerminalExactlyInEveryState(t *testing.T) {
	for _, size := range []struct{ width, height int }{{layoutTestWidth, layoutTestHeight}, {100, 30}} {
		m := sizedTestModel(t, size.width, size.height)
		m.prListView.SetPRs(layoutTestPRs())
		m.pipelineListView.SetPipelines([]domain.Pipeline{testPipeline()})

		for _, state := range allViewStates {
			m.state = state
			assertFillsTerminal(t, "closed panes", m, size.width, size.height)
		}

		m.prListView.TogglePreview()
		m.pipelineListView.TogglePreview()
		m.pipelineRunsView.TogglePreview()
		for _, state := range splitViewStates {
			m.state = state
			assertFillsTerminal(t, "open panes", m, size.width, size.height)
		}
	}
}

func TestView_CommandBarShowsSuggestionAndCompletesWithTab(t *testing.T) {
	m := sizedTestModel(t, layoutTestWidth, layoutTestHeight)
	m.state = ViewPRList
	m.commandBar.Activate()

	for _, r := range "pipe" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	if !strings.Contains(m.View(), "lines") {
		t.Fatal("expected the rest of \"pipelines\" as a suggestion")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if got := m.commandBar.Value(); got != ":pipelines " {
		t.Errorf("expected tab to complete the command, got %q", got)
	}
}

func TestPaneFocus_ScrollsPaneInsteadOfMovingCursor(t *testing.T) {
	m := sizedTestModel(t, layoutTestWidth, layoutTestHeight)
	m.state = ViewPRList
	m.prListView.SetPRs(layoutTestPRs())
	selected := m.prListView.GetSelectedPR().Number

	m, _ = handleFocusPaneInKey(m)
	if !m.prListView.IsPreviewOpen() {
		t.Fatal("expected → to open the details pane")
	}
	m, _ = handleFocusPaneInKey(m)
	if !m.isPaneFocused() {
		t.Fatal("expected → to focus the details pane")
	}

	m, _ = handleDownKey(m)
	if got := m.prListView.GetSelectedPR().Number; got != selected {
		t.Errorf("expected the cursor to stay on #%d while a pane is focused, got #%d", selected, got)
	}

	m, _ = handleEscKey(m)
	if m.isPaneFocused() {
		t.Error("expected esc to return focus to the list")
	}
	if m.state != ViewPRList {
		t.Errorf("expected esc to stay in the list while leaving the pane, got %v", m.state)
	}
}
