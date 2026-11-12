package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func diffNavigationTestModel(t *testing.T) Model {
	t.Helper()
	lines := make([]domain.DiffLine, 100)
	for i := range lines {
		lines[i] = domain.DiffLine{Type: "context", Content: fmt.Sprintf(" line %d", i+1), OldLine: i + 1, NewLine: i + 1}
	}
	m := sizedTestModel(t, layoutTestWidth, layoutTestHeight)
	m.state = ViewPRInspect
	m.prInspect.SetPR(&domain.PullRequest{Number: 1, Repository: domain.Repo{FullName: "owner/repo"}})
	m.prInspect.SetDiff(&domain.Diff{Files: []domain.FileDiff{{NewPath: "a.go", Hunks: []domain.DiffHunk{{Lines: lines}}}}})
	m.prInspect.SwitchToDiff()
	return m
}

func pressKeys(m Model, keys ...string) Model {
	for _, key := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		if key == keyDown {
			msg = tea.KeyMsg{Type: tea.KeyDown}
		}
		updated, _ := m.Update(msg)
		m = updated.(Model)
	}
	return m
}

func cursorLine(m Model) string {
	return m.prInspect.GetCurrentLineInfo().Content
}

func TestDiffKeys_CountGGAndG(t *testing.T) {
	m := pressKeys(diffNavigationTestModel(t), "4", "2", "g", "g")
	if cursorLine(m) != " line 42" {
		t.Errorf("expected 42gg to go to line 42, got %q", cursorLine(m))
	}
	m = pressKeys(m, "G")
	if cursorLine(m) != " line 100" {
		t.Errorf("expected G to go to the last line, got %q", cursorLine(m))
	}
	m = pressKeys(m, "g", "g")
	if cursorLine(m) != " line 1" {
		t.Errorf("expected gg to go to the first line, got %q", cursorLine(m))
	}
	assertFillsTerminal(t, "diff with line numbers", m, layoutTestWidth, layoutTestHeight)
}

func TestDiffKeys_OtherKeyDropsTheCount(t *testing.T) {
	m := pressKeys(diffNavigationTestModel(t), "5", "0", keyDown, "g", "g")
	if cursorLine(m) != " line 1" {
		t.Errorf("expected the count dropped by ↓ and gg to go to the top, got %q", cursorLine(m))
	}
}

func TestGKey_ChecksOutOutsideTheDiff(t *testing.T) {
	m := diffNavigationTestModel(t)
	m.prInspect.SwitchToDescription()

	m = pressKeys(m, "g")
	if m.prInspect.PendingMotion() != "" || !strings.Contains(m.View(), "No local repository linked") {
		t.Error("expected g to try a checkout in the description")
	}
}

func sendKey(m Model, msg tea.KeyMsg) Model {
	updated, _ := m.Update(msg)
	return updated.(Model)
}

func twoFileDiffModel(t *testing.T) Model {
	t.Helper()
	m := diffNavigationTestModel(t)
	m.prInspect.SetDiff(&domain.Diff{Files: []domain.FileDiff{
		{NewPath: "internal/ui/app.go", Hunks: []domain.DiffHunk{{Lines: []domain.DiffLine{{Type: "context", Content: " start", NewLine: 1}}}}},
		{NewPath: "internal/provider/client.go", Hunks: []domain.DiffHunk{{Lines: []domain.DiffLine{{Type: "add", Content: "+needle", NewLine: 1}}}}},
	}})
	m.prInspect.SwitchToDiff()
	return m
}

func TestDiffKeys_SpaceSpaceFindsAndOpensAFile(t *testing.T) {
	m := twoFileDiffModel(t)
	m = sendKey(m, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")})
	m = sendKey(m, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")})
	if !m.fileFinderView.IsActive() || !m.isInInputMode() {
		t.Fatal("expected space space to open the file finder")
	}
	assertFillsTerminal(t, "file finder", m, layoutTestWidth, layoutTestHeight)

	m = pressKeys(m, "c", "l", "i")
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.fileFinderView.IsActive() || cursorLine(m) != "+needle" {
		t.Errorf("expected enter to open client.go, got %q", cursorLine(m))
	}
}

func TestDiffKeys_SlashSearchesAndNJumpsBetweenMatches(t *testing.T) {
	m := twoFileDiffModel(t)
	m = pressKeys(m, "/", "n", "e", "e", "d")
	if !m.isInInputMode() || cursorLine(m) != "+needle" {
		t.Fatalf("expected typing to jump to the match, got %q", cursorLine(m))
	}
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyEnter})

	m = pressKeys(m, "n")
	if cursorLine(m) != "+needle" || !strings.Contains(m.View(), "Search: need 1/1") {
		t.Error("expected n to stay on the only match while searching, not go to the next file")
	}
	m = sendKey(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.prInspect.SearchQuery() != "" || m.state != ViewPRInspect {
		t.Fatal("expected esc to clear the search first")
	}
	m = pressKeys(m, "p")
	if cursorLine(m) != " start" {
		t.Errorf("expected p to go back to the first file, got %q", cursorLine(m))
	}
	m = pressKeys(m, "n")
	if cursorLine(m) != "+needle" {
		t.Errorf("expected n without a search to go to the next file, got %q", cursorLine(m))
	}
}
