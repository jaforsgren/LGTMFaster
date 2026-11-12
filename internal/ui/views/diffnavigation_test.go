package views

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	navigationTestWidth  = 120
	navigationTestHeight = 20
	// navigationTestLines is long enough that the diff scrolls in the test's height.
	navigationTestLines = 200
)

// longDiffView shows a one-hunk diff of a file: a deleted line 1, then context lines whose new
// numbers run from 1, with every tenth one added.
func longDiffView() *PRInspectViewModel {
	lines := []domain.DiffLine{{Type: diffLineTypeDelete, Content: "-removed", OldLine: 1}}
	for number := 1; number <= navigationTestLines; number++ {
		line := domain.DiffLine{Type: diffLineTypeCtx, Content: fmt.Sprintf(" line %d", number), OldLine: number + 1, NewLine: number}
		if number%10 == 0 {
			line = domain.DiffLine{Type: diffLineTypeAdd, Content: fmt.Sprintf("+line %d", number), NewLine: number}
		}
		lines = append(lines, line)
	}
	m := NewPRInspectView()
	m.SetSize(navigationTestWidth, navigationTestHeight)
	m.SetDiff(&domain.Diff{Files: []domain.FileDiff{{NewPath: "main.go", Hunks: []domain.DiffHunk{{Header: "@@ -1,191 +1,200 @@", Lines: lines}}}}})
	m.SwitchToDiff()
	return m
}

func cursorContent(m *PRInspectViewModel) string {
	return m.GetCurrentLineInfo().Content
}

func TestDiffGutter_ShowsOldAndNewLineNumbers(t *testing.T) {
	m := longDiffView()
	view := ansi.Strip(m.View())

	for _, want := range []string{"  1     │ ", "  2   1 │ ", "     10 │ "} {
		if !strings.Contains(view, want) {
			t.Errorf("expected gutter %q in:\n%s", want, view)
		}
	}
	if got := diffGutterDigits(domain.FileDiff{Hunks: []domain.DiffHunk{{Lines: []domain.DiffLine{{NewLine: 12345}}}}}); got != 5 {
		t.Errorf("expected the gutter to fit 5 digits, got %d", got)
	}
}

func TestDiffMotion_GGAndG(t *testing.T) {
	m := longDiffView()

	m.PressShiftG()
	if cursorContent(m) != "+line 200" || !strings.Contains(m.View(), "+line 200") {
		t.Errorf("expected G to move to and show the last line, got %q", cursorContent(m))
	}

	m.PressG()
	if m.PendingMotion() != "g" || cursorContent(m) != "+line 200" {
		t.Fatal("expected a single g to wait for the second")
	}
	m.PressG()
	if cursorContent(m) != "-removed" || m.PendingMotion() != "" {
		t.Errorf("expected gg to move to the first line, got %q", cursorContent(m))
	}
}

func TestDiffMotion_CountGoesToLineInTheNewFile(t *testing.T) {
	m := longDiffView()

	for _, digit := range []int{0, 1, 5, 0} {
		m.PushMotionDigit(digit)
	}
	if m.PendingMotion() != "150" {
		t.Fatalf("expected a leading 0 to be ignored, got %q", m.PendingMotion())
	}
	if !strings.Contains(m.View(), "| 150") {
		t.Error("expected the pending count in the status line")
	}
	m.PressG()
	m.PressG()
	if cursorContent(m) != "+line 150" || !strings.Contains(m.View(), "+line 150") {
		t.Errorf("expected 150gg to show line 150, got %q", cursorContent(m))
	}

	m.PushMotionDigit(3)
	m.PressShiftG()
	if cursorContent(m) != " line 3" {
		t.Errorf("expected 3G to go to line 3, got %q", cursorContent(m))
	}

	m.PushMotionDigit(9)
	m.PushMotionDigit(9)
	m.PushMotionDigit(9)
	m.PressShiftG()
	if cursorContent(m) != "+line 200" {
		t.Errorf("expected a count past the end to go to the last line, got %q", cursorContent(m))
	}
}

func TestDiffMotion_CompactModeSkipsHiddenLines(t *testing.T) {
	m := longDiffView()
	m.ToggleDiffViewMode()

	m.PushMotionDigit(5)
	m.PressShiftG()
	if cursorContent(m) != "+line 10" {
		t.Errorf("expected 5G to go to the next shown line, got %q", cursorContent(m))
	}
	m.PressShiftG()
	if !strings.Contains(m.View(), "+line 200") {
		t.Error("expected G to scroll to the last shown line in compact mode")
	}
}

func TestDiffMotion_ResetDropsAHalfTypedMotion(t *testing.T) {
	m := longDiffView()
	m.PushMotionDigit(4)
	m.PressG()
	m.ResetMotion()
	m.PressG()
	if m.PendingMotion() != "g" || cursorContent(m) != "-removed" {
		t.Errorf("expected the reset to drop the count, got %q", m.PendingMotion())
	}
}

func twoFileDiffView() *PRInspectViewModel {
	file := func(path string, contents ...string) domain.FileDiff {
		lines := make([]domain.DiffLine, len(contents))
		for i, content := range contents {
			lines[i] = domain.DiffLine{Type: diffLineTypeCtx, Content: content, OldLine: i + 1, NewLine: i + 1}
		}
		return domain.FileDiff{NewPath: path, Hunks: []domain.DiffHunk{{Lines: lines}}}
	}
	m := NewPRInspectView()
	m.SetSize(navigationTestWidth, navigationTestHeight)
	m.SetDiff(&domain.Diff{Files: []domain.FileDiff{
		file("a.go", "func Alpha()", "return nil", "TODO later"),
		file("b.go", "package b", "// todo here"),
	}})
	m.SwitchToDiff()
	return m
}

func searchFor(m *PRInspectViewModel, query string) {
	m.ActivateSearch()
	for _, r := range query {
		m.UpdateSearch(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.DeactivateSearch()
}

func TestDiffSearch_MovesBetweenMatchesAcrossFilesAndWraps(t *testing.T) {
	m := twoFileDiffView()
	searchFor(m, "todo")

	if m.currentFile != 0 || cursorContent(m) != "TODO later" {
		t.Fatalf("expected the first match from the cursor, got %q", cursorContent(m))
	}
	if !strings.Contains(m.View(), "Search: todo 1/2") {
		t.Errorf("expected the match position in the status line, got:\n%s", m.View())
	}
	if !m.NextMatch() || m.currentFile != 1 || cursorContent(m) != "// todo here" {
		t.Errorf("expected n to go on into the next file, got %q", cursorContent(m))
	}
	if !m.NextMatch() || cursorContent(m) != "TODO later" {
		t.Errorf("expected n to wrap to the first match, got %q", cursorContent(m))
	}
	if !m.PrevMatch() || cursorContent(m) != "// todo here" {
		t.Errorf("expected N to wrap back to the last match, got %q", cursorContent(m))
	}

	m.ClearSearch()
	if m.NextMatch() || m.SearchQuery() != "" {
		t.Error("expected no matches once the search is cleared")
	}
}

func TestDiffLeader_SecondSpaceOpens(t *testing.T) {
	m := twoFileDiffView()
	if m.PressSpace() || m.PendingMotion() != pendingSpaceLabel {
		t.Fatal("expected the first space to wait")
	}
	if !m.PressSpace() || m.PendingMotion() != "" {
		t.Error("expected the second space to open the finder")
	}
	m.PressSpace()
	m.PushMotionDigit(1)
	if m.PressSpace() {
		t.Error("expected a digit to break the leader")
	}
}

func TestGoToFile_ShowsTheFileFromTheTop(t *testing.T) {
	m := twoFileDiffView()
	m.GoToFile(1)
	if m.currentFile != 1 || cursorContent(m) != "package b" {
		t.Errorf("expected the first line of b.go, got %q", cursorContent(m))
	}
	m.GoToFile(5)
	if m.currentFile != 1 {
		t.Error("expected an unknown file to be ignored")
	}
}

func TestDiffHeader_StaysInViewWhileScrolling(t *testing.T) {
	m := longDiffView()
	const header = "File 1/1: main.go"

	for name, move := range map[string]func(){
		"at the top":         func() {},
		"after G":            m.PressShiftG,
		"scrolling up again": func() { m.PrevLine(); m.PrevLine() },
		"after gg":           func() { m.PressG(); m.PressG() },
	} {
		move()
		if view := ansi.Strip(m.View()); !strings.Contains(view, header) {
			t.Errorf("%s: expected the file name in view, got:\n%s", name, view)
		}
	}
	if got := strings.Count(m.View(), "\n") + 1; got != outerSize(navigationTestHeight, paneVerticalPadding) {
		t.Errorf("expected the header to take rows from the diff, not grow the view, got %d rows", got)
	}
}

func TestDiffHeader_ShowsTheViewedCommit(t *testing.T) {
	m := longDiffView()
	diff := m.GetDiff()
	m.ShowCommitDiff(domain.Commit{ID: "abcdef1234", Message: "Fix login"}, diff)
	m.PressShiftG()
	if view := ansi.Strip(m.View()); !strings.Contains(view, "Fix login") || !strings.Contains(view, "main.go") {
		t.Errorf("expected the commit and file in view, got:\n%s", view)
	}
}

// changesDiffView has a.go with two runs of changes and b.go with one.
func changesDiffView() *PRInspectViewModel {
	line := func(lineType, content string) domain.DiffLine {
		return domain.DiffLine{Type: lineType, Content: content, OldLine: 1, NewLine: 1}
	}
	m := NewPRInspectView()
	m.SetSize(navigationTestWidth, navigationTestHeight)
	m.SetDiff(&domain.Diff{Files: []domain.FileDiff{
		{NewPath: "a.go", Hunks: []domain.DiffHunk{{Lines: []domain.DiffLine{
			line(diffLineTypeCtx, " one"), line(diffLineTypeDelete, "-two"), line(diffLineTypeAdd, "+two"),
			line(diffLineTypeCtx, " three"), line(diffLineTypeAdd, "+four"),
		}}}},
		{NewPath: "b.go", Hunks: []domain.DiffHunk{{Lines: []domain.DiffLine{
			line(diffLineTypeCtx, " b"), line(diffLineTypeAdd, "+b"),
		}}}},
	}})
	m.SwitchToDiff()
	return m
}

func TestDiffChanges_JumpBetweenRunsAcrossFiles(t *testing.T) {
	m := changesDiffView()
	for _, want := range []string{"-two", "+four", "+b"} {
		if !m.NextChange() || cursorContent(m) != want {
			t.Fatalf("expected the next change to start at %q, got %q", want, cursorContent(m))
		}
	}
	if m.NextChange() || cursorContent(m) != "+b" {
		t.Error("expected no wrap after the last change")
	}

	if !m.PrevChange() || m.currentFile != 0 || cursorContent(m) != "+four" {
		t.Errorf("expected the previous change in the previous file, got %q", cursorContent(m))
	}
	m.PrevLine()
	m.PrevLine()
	if cursorContent(m) != "+two" {
		t.Fatalf("setup: expected the cursor inside the first run, got %q", cursorContent(m))
	}
	if !m.PrevChange() || cursorContent(m) != "-two" {
		t.Errorf("expected K inside a run to go to its start, got %q", cursorContent(m))
	}
	if m.PrevChange() {
		t.Error("expected no change before the first")
	}
}
