package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

func workItemTestModel(t *testing.T, provider *mockProvider) (Model, domain.PullRequest) {
	t.Helper()
	pr := domain.PullRequest{
		Number: 42, Title: "Add login", Description: "Adds the login page", PATID: testPATID,
		Repository: domain.Repo{FullName: "Proj/api"},
	}
	m := newPRTestModel(t, provider)
	m.state = ViewPRInspect
	m.prInspect.SetPR(&pr)
	return m, pr
}

func pressKey(t *testing.T, m Model, key string) (Model, tea.Cmd) {
	t.Helper()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	switch key {
	case keyTogglePane:
		msg = tea.KeyMsg{Type: tea.KeyTab}
	case keyEnter:
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case keyEsc:
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case keyFocusPaneIn:
		msg = tea.KeyMsg{Type: tea.KeyRight}
	case keyDown:
		msg = tea.KeyMsg{Type: tea.KeyDown}
	}
	updated, cmd := m.Update(msg)
	return updated.(Model), cmd
}

func runMsg(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a command")
	}
	updated, _ := m.Update(cmd())
	return updated.(Model)
}

func TestWorkItemsPane_TabShowsTheLinkedWorkItemsBesideTheDescription(t *testing.T) {
	provider := &mockProvider{linkedWorkItems: []domain.WorkItem{
		{ID: 12, Type: "Task", Title: "Build the login form", State: "Active", Description: "Email and password fields"},
	}}
	m, _ := workItemTestModel(t, provider)

	m, cmd := pressKey(t, m, keyTogglePane)
	if !strings.Contains(m.View(), "Loading work items...") {
		t.Error("expected the pane to show it is loading")
	}
	m = runMsg(t, m, cmd)

	view := m.View()
	for _, want := range []string{"Work items (1)", "#12 Task", "Active", "Build the login form", "Email and password fields", "Adds the login page"} {
		if !strings.Contains(view, want) {
			t.Errorf("expected %q beside the description", want)
		}
	}
	assertFillsTerminal(t, "work items pane", m, layoutTestWidth, layoutTestHeight)
}

func TestWorkItemsPane_RightFocusesThePaneAndEscGoesBack(t *testing.T) {
	m, _ := workItemTestModel(t, &mockProvider{})
	m, cmd := pressKey(t, m, keyTogglePane)
	m = runMsg(t, m, cmd)

	m, _ = pressKey(t, m, keyFocusPaneIn)
	if !m.isPaneFocused() {
		t.Fatal("expected → to focus the work items pane")
	}
	if !strings.Contains(m.View(), "No linked work items") {
		t.Error("expected the pane to say nothing is linked")
	}
	m, _ = pressKey(t, m, keyEsc)

	if m.isPaneFocused() || m.state != ViewPRInspect {
		t.Errorf("expected esc to leave the pane and stay in the PR, got state %v", m.state)
	}
}

func TestWorkItemsPane_IsNotShownBesideTheDiff(t *testing.T) {
	m, _ := workItemTestModel(t, &mockProvider{})
	m.prInspect.SwitchToDiff()

	m, cmd := pressKey(t, m, keyTogglePane)

	if cmd != nil || m.prInspect.IsPreviewOpen() {
		t.Error("expected tab to do nothing in the diff")
	}
}

func TestWorkItemPicker_LinksAFilteredWorkItemThatIsNotLinkedYet(t *testing.T) {
	linked := domain.WorkItem{ID: 12, Type: "Task", Title: "Build the login form"}
	provider := &mockProvider{
		linkedWorkItems: []domain.WorkItem{linked},
		assignedWorkItems: []domain.WorkItem{
			linked,
			{ID: 30, Type: "Bug", Title: "Crash on save", State: "New"},
			{ID: 31, Type: "User Story", Title: "Password reset", State: "Active"},
		},
	}
	m, _ := workItemTestModel(t, provider)
	m, cmd := pressKey(t, m, keyTogglePane)
	m = runMsg(t, m, cmd)

	m, cmd = pressKey(t, m, keyLinkWorkItem)
	if !m.workItemPickerView.IsActive() || !m.isInInputMode() {
		t.Fatal("expected l to open the work item picker")
	}
	m = runMsg(t, m, cmd)
	view := m.View()
	if strings.Contains(view, "Build the login form") || !strings.Contains(view, "Crash on save") || !strings.Contains(view, "Password reset") {
		t.Error("expected the assigned work items without the linked one")
	}

	for _, r := range "reset" {
		m, _ = pressKey(t, m, string(r))
	}
	if strings.Contains(m.View(), "Crash on save") {
		t.Error("expected typing to filter the work items")
	}
	m, cmd = pressKey(t, m, keyEnter)
	if m.workItemPickerView.IsActive() {
		t.Error("expected enter to close the picker")
	}
	m = runMsg(t, m, cmd)

	if provider.linkedWorkItemID != 31 {
		t.Errorf("expected work item 31 linked, got %d", provider.linkedWorkItemID)
	}
	if !strings.Contains(m.View(), "Linked #31 Password reset") {
		t.Error("expected the status bar to confirm the link")
	}
	if m.prInspect.PendingWorkItemsRequest() != nil {
		t.Error("expected the linked work items to be reloaded already")
	}
}

func TestWorkItemPicker_EscClosesWithoutLinking(t *testing.T) {
	provider := &mockProvider{assignedWorkItems: []domain.WorkItem{{ID: 30, Type: "Bug", Title: "Crash on save"}}}
	m, _ := workItemTestModel(t, provider)
	m, cmd := pressKey(t, m, keyLinkWorkItem)
	m = runMsg(t, m, cmd)

	m, _ = pressKey(t, m, keyEsc)

	if m.workItemPickerView.IsActive() || provider.linkedWorkItemID != 0 || m.prInspect.GetMode() != views.PRInspectModeDescription {
		t.Error("expected esc to close the picker without linking or leaving the description")
	}
}

func TestWorkItemsPane_IsClosedWhenAPullRequestIsOpened(t *testing.T) {
	m, pr := workItemTestModel(t, &mockProvider{})
	m, _ = pressKey(t, m, keyTogglePane)

	m, _ = m.inspectPR(pr)

	if m.prInspect.IsPreviewOpen() {
		t.Error("expected a newly opened pull request to hide the work items pane")
	}
}
