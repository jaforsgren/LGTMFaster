package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func filterTestDiff() *domain.Diff {
	return &domain.Diff{Files: []domain.FileDiff{
		{NewPath: "src/app.go", Hunks: []domain.DiffHunk{{Header: "@@ -1,4 +1,5 @@", Lines: []domain.DiffLine{
			{Type: "context", Content: " package app", OldLine: 1, NewLine: 1},
			{Type: "delete", Content: "-func Run() {", OldLine: 2},
			{Type: "delete", Content: "-  return", OldLine: 3},
			{Type: "add", Content: "+func Run()  {", NewLine: 2},
			{Type: "add", Content: "+\treturn nil", NewLine: 3},
			{Type: "add", Content: "+   ", NewLine: 4},
			{Type: "context", Content: " }", OldLine: 4, NewLine: 5},
		}}}},
		{NewPath: "README.md", Hunks: []domain.DiffHunk{{Header: "@@ -1 +1 @@", Lines: []domain.DiffLine{
			{Type: "add", Content: "+# Title", NewLine: 1},
		}}}},
		{NewPath: "Makefile", Hunks: []domain.DiffHunk{{Header: "@@ -1 +1 @@", Lines: []domain.DiffLine{
			{Type: "add", Content: "+build:", NewLine: 1},
		}}}},
	}}
}

func TestCollapseWhitespaceChanges(t *testing.T) {
	lines := collapseWhitespaceChanges(filterTestDiff().Files[0].Hunks[0].Lines)

	var contents []string
	for _, line := range lines {
		contents = append(contents, line.Content)
	}
	expected := []string{" package app", " func Run()  {", "-  return", "+\treturn nil", " }"}
	if strings.Join(contents, "|") != strings.Join(expected, "|") {
		t.Fatalf("expected %q, got %q", expected, contents)
	}
	if lines[1].OldLine != 2 || lines[1].NewLine != 2 || lines[1].Type != diffLineTypeCtx {
		t.Errorf("expected the whitespace-only change to become context on old 2 / new 2, got %+v", lines[1])
	}
}

func TestFileExtensions(t *testing.T) {
	if FileExtension("src/App.GO") != ".go" || FileExtension("Makefile") != noExtension {
		t.Error("expected lower-cased extensions and a name for files without one")
	}
	extensions := DiffExtensions(filterTestDiff())
	if len(extensions) != 3 || extensions[0].Extension != noExtension || extensions[1].Extension != ".go" || extensions[2].Files != 1 {
		t.Errorf("unexpected extensions %+v", extensions)
	}
}

func filteredInspectView() *PRInspectViewModel {
	m := NewPRInspectView()
	m.SetSize(120, 40)
	m.SetDiff(filterTestDiff())
	m.SwitchToDiff()
	return m
}

func TestExtensionFilter_HidesFilesFromTheDiffAndTheYank(t *testing.T) {
	m := filteredInspectView()

	m.SetHiddenExtensions(map[string]bool{".md": true, noExtension: true})

	all := m.GetAllFilesDiffText()
	if strings.Contains(all, "README.md") || strings.Contains(all, "Makefile") || !strings.Contains(all, "src/app.go") {
		t.Errorf("expected Y to copy only the shown files, got %q", all)
	}
	if len(m.GetDiff().Files) != 3 {
		t.Error("expected the comment view to keep the whole diff")
	}
	if !strings.Contains(m.View(), "1/3 files, hiding (none), .md") {
		t.Errorf("expected the status line to say what is hidden, got %q", m.statusLine())
	}

	m.SetHiddenExtensions(map[string]bool{".md": true, noExtension: true, ".go": true})
	if !strings.Contains(m.View(), allFilesFilteredText) {
		t.Error("expected a notice when every file is hidden")
	}
}

func TestIgnoreWhitespace_ChangesTheDiffAndTheYank(t *testing.T) {
	m := filteredInspectView()

	m.ToggleIgnoreWhitespace()

	text := m.GetCurrentFileDiffText()
	if strings.Contains(text, "-func Run() {") || strings.Contains(text, "+   ") {
		t.Errorf("expected whitespace-only changes left out of y, got %q", text)
	}
	if !strings.Contains(text, "+\treturn nil") {
		t.Errorf("expected real changes kept, got %q", text)
	}

	m.ToggleIgnoreWhitespace()
	if !strings.Contains(m.GetCurrentFileDiffText(), "-func Run() {") {
		t.Error("expected toggling back to show whitespace changes again")
	}
}

func TestExtensionFilterView_TogglesAndAppliesExtensions(t *testing.T) {
	view := NewExtensionFilterView()
	view.SetSize(80, 30)
	view.Activate(DiffExtensions(filterTestDiff()), map[string]bool{})

	view.HandleKey(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")})
	if !view.Hidden()[noExtension] {
		t.Error("expected space to hide the selected extension")
	}
	view.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if view.Hidden()[noExtension] {
		t.Error("expected a to show everything when something is hidden")
	}
	view.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if !view.Hidden()[".go"] || !view.Hidden()[".md"] {
		t.Error("expected a to hide everything when everything is shown")
	}
	if action := view.HandleKey(tea.KeyMsg{Type: tea.KeyEnter}); action != ExtensionFilterActionApply {
		t.Errorf("expected enter to apply, got %v", action)
	}
	if !strings.Contains(view.View(), extensionHiddenMarker+".go") {
		t.Error("expected unticked extensions in the view")
	}
}
