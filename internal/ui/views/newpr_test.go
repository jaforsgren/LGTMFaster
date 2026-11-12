package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func testPushedBranch() domain.PushedBranch {
	return domain.PushedBranch{Repository: domain.Repo{FullName: "Proj/api"}, Branch: "feature/add-login_page"}
}

func newPRTestView() *NewPRViewModel {
	m := NewNewPRView()
	m.SetSize(viewTestWidth, viewTestHeight)
	m.Activate(testPushedBranch())
	m.SetBranches(domain.RepositoryBranches{Default: "main", Names: []string{"develop", "feature/add-login_page", "main", "release/1.0"}})
	return m
}

func pressKeys(m *NewPRViewModel, keys ...tea.KeyMsg) NewPRAction {
	action := NewPRActionNone
	for _, key := range keys {
		action, _ = m.HandleKey(key)
	}
	return action
}

func typed(text string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)}
}

var (
	tabKey      = tea.KeyMsg{Type: tea.KeyTab}
	shiftTabKey = tea.KeyMsg{Type: tea.KeyShiftTab}
	spaceKey    = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
)

func TestTitleFromBranch(t *testing.T) {
	tests := map[string]string{
		"feature/add-login_page": "Add login page",
		"fix-typo":               "Fix typo",
		"main":                   "Main",
		"feature/":               "",
	}
	for branch, expected := range tests {
		if got := TitleFromBranch(branch); got != expected {
			t.Errorf("TitleFromBranch(%q) = %q, want %q", branch, got, expected)
		}
	}
}

func TestNewPR_DefaultsToTheDefaultBranchAndATitleFromTheSource(t *testing.T) {
	request, err := newPRTestView().Request()
	if err != nil {
		t.Fatal(err)
	}

	if request.TargetBranch != "main" || request.SourceBranch != "feature/add-login_page" || request.Title != "Add login page" {
		t.Errorf("unexpected request %+v", request)
	}
	if request.IsDraft || request.RepositoryFullName != "Proj/api" {
		t.Errorf("expected a ready pull request in Proj/api, got %+v", request)
	}
}

func TestNewPR_TargetDropdownCompletesTheTypedBranch(t *testing.T) {
	m := newPRTestView()
	pressKeys(m, shiftTabKey)
	m.target.SetValue("")

	pressKeys(m, typed("rel"), tabKey)

	if m.target.Value() != "release/1.0" {
		t.Errorf("expected the dropdown to complete to release/1.0, got %q", m.target.Value())
	}
	if !strings.Contains(m.View(), "Title:") {
		t.Error("expected the form to render")
	}
}

func TestNewPR_TypingIntoThePreselectedTargetReplacesIt(t *testing.T) {
	m := newPRTestView()
	pressKeys(m, shiftTabKey)

	pressKeys(m, typed("dev"))

	if m.target.Value() != "dev" {
		t.Errorf("expected typing to replace the preselected main, got %q", m.target.Value())
	}
	pressKeys(m, tabKey)
	if m.target.Value() != "develop" {
		t.Errorf("expected tab to complete to develop, got %q", m.target.Value())
	}
}

func TestNewPR_EditingKeysKeepThePreselectedTarget(t *testing.T) {
	m := newPRTestView()
	pressKeys(m, shiftTabKey)

	pressKeys(m, tea.KeyMsg{Type: tea.KeyBackspace})

	if m.target.Value() != "mai" {
		t.Errorf("expected backspace to edit the preselected branch, got %q", m.target.Value())
	}
}

func TestNewPR_DropdownLeavesOutTheSourceBranch(t *testing.T) {
	m := newPRTestView()
	m.target.SetValue("feature")

	if matches := m.target.Matches(); len(matches) != 0 {
		t.Errorf("expected the source branch not to be offered, got %v", matches)
	}
}

func TestNewPR_SpaceOnStatusTogglesDraft(t *testing.T) {
	m := newPRTestView()

	pressKeys(m, tabKey, spaceKey)

	request, _ := m.Request()
	if !request.IsDraft {
		t.Error("expected space on the status field to make the pull request a draft")
	}
}

func TestNewPR_RejectsInvalidForms(t *testing.T) {
	tests := map[string]func(m *NewPRViewModel){
		"no title":       func(m *NewPRViewModel) { m.title.SetValue("  ") },
		"same branch":    func(m *NewPRViewModel) { m.target.SetValue("feature/add-login_page") },
		"unknown target": func(m *NewPRViewModel) { m.target.SetValue("nope") },
		"no target":      func(m *NewPRViewModel) { m.target.SetValue("") },
	}
	for name, breakForm := range tests {
		m := newPRTestView()
		breakForm(m)
		if _, err := m.Request(); err == nil {
			t.Errorf("%s: expected the form to be rejected", name)
		}
	}
}

func TestNewPR_KeysForSubmitCancelAndEditor(t *testing.T) {
	m := newPRTestView()
	keys := map[string]NewPRAction{"ctrl+s": NewPRActionSubmit, "esc": NewPRActionCancel, "ctrl+g": NewPRActionOpenEditor}
	keyTypes := map[string]tea.KeyType{"ctrl+s": tea.KeyCtrlS, "esc": tea.KeyEsc, "ctrl+g": tea.KeyCtrlG}
	for name, expected := range keys {
		if action := pressKeys(m, tea.KeyMsg{Type: keyTypes[name]}); action != expected {
			t.Errorf("%s: expected action %v, got %v", name, expected, action)
		}
	}
}
