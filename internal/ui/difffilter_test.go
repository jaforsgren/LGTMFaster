package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

func TestFileFilterKey_PicksExtensionsForTheDiff(t *testing.T) {
	m := createTestModel()
	m.prInspect.SetDiff(&domain.Diff{Files: []domain.FileDiff{{NewPath: "a.go"}, {NewPath: "b.md"}}})

	m, _ = handleFileFilterKey(m)
	if m.extensionFilterView.IsActive() {
		t.Fatal("expected the filter only in diff mode")
	}

	m.prInspect.SwitchToDiff()
	m, _ = handleFileFilterKey(m)
	if !m.extensionFilterView.IsActive() {
		t.Fatal("expected F to open the extension filter")
	}
	m, _ = m.handleExtensionFilterKey(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.handleExtensionFilterKey(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")})
	m, _ = m.handleExtensionFilterKey(tea.KeyMsg{Type: tea.KeyEnter})

	if m.extensionFilterView.IsActive() || !m.prInspect.HiddenExtensions()[".md"] {
		t.Errorf("expected .md hidden after applying, got %v", m.prInspect.HiddenExtensions())
	}
	if views.FileExtension(m.prInspect.GetCurrentFileDiffText()) == ".md" {
		t.Error("expected the hidden file out of the yank")
	}
}

func TestWhitespaceKey_TogglesOnlyInDiffMode(t *testing.T) {
	m := createTestModel()

	m, _ = handleWhitespaceKey(m)
	if m.prInspect.IgnoresWhitespace() {
		t.Error("expected w to do nothing outside the diff")
	}

	m.prInspect.SwitchToDiff()
	m, _ = handleWhitespaceKey(m)
	if !m.prInspect.IgnoresWhitespace() {
		t.Error("expected w to hide whitespace changes in the diff")
	}
}
