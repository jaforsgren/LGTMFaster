package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func finderTestFiles() []domain.FileDiff {
	changed := []domain.DiffHunk{{Lines: []domain.DiffLine{{Type: diffLineTypeAdd}, {Type: diffLineTypeAdd}, {Type: diffLineTypeDelete}, {Type: diffLineTypeCtx}}}}
	return []domain.FileDiff{
		{NewPath: "internal/ui/app.go", Hunks: changed},
		{NewPath: "internal/provider/azuredevops/client.go"},
		{NewPath: "README.md"},
		{OldPath: "internal/ui/old.go", IsDeleted: true},
	}
}

func typeInto(m *FileFinderViewModel, text string) {
	for _, r := range text {
		m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestFileFinder_ListsEveryFileWithTheCurrentOneSelected(t *testing.T) {
	m := NewFileFinderView()
	m.SetSize(120, 40)
	m.Activate(finderTestFiles(), 2)

	if index, ok := m.SelectedFile(); !ok || index != 2 {
		t.Errorf("expected the current file selected, got %d", index)
	}
	view := ansi.Strip(m.View())
	for _, want := range []string{"Find file", "4/4", "internal/ui/app.go", "+2 -1", "internal/ui/old.go"} {
		if !strings.Contains(view, want) {
			t.Errorf("expected %q in:\n%s", want, view)
		}
	}
	if lipgloss.Height(m.View()) != 40 {
		t.Errorf("expected the modal centred in the whole area, got %d rows", lipgloss.Height(m.View()))
	}
}

func TestFileFinder_FuzzyMatchesAndOpensTheSelectedFile(t *testing.T) {
	m := NewFileFinderView()
	m.SetSize(120, 40)
	m.Activate(finderTestFiles(), 0)

	typeInto(m, "azcli")
	if index, ok := m.SelectedFile(); !ok || index != 1 {
		t.Fatalf("expected the fuzzy match on the client, got %d", index)
	}
	if !strings.Contains(ansi.Strip(m.View()), "1/4") {
		t.Error("expected the match count")
	}
	if action, _ := m.HandleKey(tea.KeyMsg{Type: tea.KeyEnter}); action != FileFinderActionOpen {
		t.Error("expected enter to open the file")
	}

	typeInto(m, "zzz")
	if _, ok := m.SelectedFile(); ok || !strings.Contains(m.View(), fileFinderNoMatch) {
		t.Error("expected no match")
	}
	if action, _ := m.HandleKey(tea.KeyMsg{Type: tea.KeyEnter}); action != FileFinderActionNone {
		t.Error("expected enter to do nothing without a match")
	}
	if action, _ := m.HandleKey(tea.KeyMsg{Type: tea.KeyEsc}); action != FileFinderActionCancel {
		t.Error("expected esc to close")
	}
}

func TestFileFinder_MovesThroughTheResults(t *testing.T) {
	m := NewFileFinderView()
	m.SetSize(120, 40)
	m.Activate(finderTestFiles(), 0)
	typeInto(m, "ui")

	m.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlN})
	second, _ := m.SelectedFile()
	m.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlP})
	first, _ := m.SelectedFile()
	if first == second {
		t.Error("expected ctrl+n and ctrl+p to move between the matches")
	}
}

func TestHighlightMatches_StylesOnlyMatchedCharacters(t *testing.T) {
	plain := lipgloss.NewStyle()
	if got := ansi.Strip(highlightMatches("app.go", []int{0, 4}, plain)); got != "app.go" {
		t.Errorf("expected the text unchanged, got %q", got)
	}
}

func TestTruncateStart_KeepsTheEnd(t *testing.T) {
	tests := map[string]struct {
		in, want string
		width    int
		wantCut  int
	}{
		"fits":     {"a/b.go", "a/b.go", 10, 0},
		"cut":      {"internal/ui/views/filefinder.go", "...filefinder.go", 16, 18},
		"too tiny": {"internal/a.go", "", 3, 13},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, cut := truncateStart(tt.in, tt.width)
			if got != tt.want || cut != tt.wantCut {
				t.Errorf("truncateStart(%q, %d) = %q, %d; want %q, %d", tt.in, tt.width, got, cut, tt.want, tt.wantCut)
			}
		})
	}
}

func TestShiftMatches_FollowsTheCut(t *testing.T) {
	got := shiftMatches([]int{0, 18, 20}, 18)
	if len(got) != 2 || got[0] != 3 || got[1] != 5 {
		t.Errorf("expected cut positions dropped and the rest moved after the marker, got %v", got)
	}
}

func TestFileFinder_IsWideAndKeepsTheFileNameOfLongPaths(t *testing.T) {
	long := strings.Repeat("deeply/nested/", 12) + "important_file.go"
	m := NewFileFinderView()
	m.SetSize(150, 40)
	m.Activate([]domain.FileDiff{{NewPath: long}}, 0)
	typeInto(m, "important")

	view := ansi.Strip(m.View())
	if !strings.Contains(view, "...") || !strings.Contains(view, "nested/important_file.go") {
		t.Errorf("expected the start cut and the file name kept, got:\n%s", view)
	}
	boxWidth := 0
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "╭") {
			boxWidth = lipgloss.Width(strings.TrimSpace(line))
		}
	}
	if boxWidth != 150*fileFinderWidthPercent/percentScale {
		t.Errorf("expected the box to take %d%% of the width, got %d", fileFinderWidthPercent, boxWidth)
	}
}
