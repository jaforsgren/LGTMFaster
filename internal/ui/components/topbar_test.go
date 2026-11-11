package components

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	wideTerminal   = 240
	narrowTerminal = 60
)

func testShortcuts(count int) []string {
	shortcuts := make([]string, count)
	for i := range shortcuts {
		shortcuts[i] = fmt.Sprintf("<k%d> Action %d", i, i)
	}
	return shortcuts
}

func TestBuildShortcutsDisplay_FlowsIntoColumnsOfSectionRows(t *testing.T) {
	m := NewTopBar()
	m.SetShortcuts(testShortcuts(13))

	rows := m.buildShortcutsDisplay(wideTerminal)

	if len(rows) != topBarSectionRows {
		t.Fatalf("expected %d rows, got %d", topBarSectionRows, len(rows))
	}
	if !strings.Contains(rows[0], "Action 12") {
		t.Errorf("expected the third column to start on the first row, got %q", rows[0])
	}
}

func TestBuildShortcutsDisplay_DropsColumnsThatDoNotFit(t *testing.T) {
	m := NewTopBar()
	m.SetShortcuts(testShortcuts(13))

	rows := m.buildShortcutsDisplay(25)

	for _, row := range rows {
		if strings.Contains(row, "Action 6") {
			t.Fatalf("expected only the first column to fit, got %q", row)
		}
	}
}

func TestBuildShortcutsDisplay_Empty(t *testing.T) {
	if rows := NewTopBar().buildShortcutsDisplay(wideTerminal); len(rows) != 0 {
		t.Errorf("expected no rows, got %d", len(rows))
	}
}

func TestTopBarView_RendersEveryShortcutOnWideTerminal(t *testing.T) {
	m := NewTopBar()
	m.SetWidth(wideTerminal)
	m.SetView(ViewNamePipelines)
	m.SetShortcuts(testShortcuts(13))

	view := m.View()

	for i := range 13 {
		if !strings.Contains(view, fmt.Sprintf("Action %d", i)) {
			t.Errorf("expected shortcut %d in top bar", i)
		}
	}
}

func TestTopBarView_HasFixedHeightAndNeverWraps(t *testing.T) {
	m := NewTopBar()
	m.SetWidth(narrowTerminal)
	m.SetView(ViewNamePRList)
	m.SetAccounts([]Account{{Name: strings.Repeat("long-account-name", 4), Selected: true}})
	m.SetShortcuts(testShortcuts(30))

	view := m.View()

	if height := lipgloss.Height(view); height != TopBarHeight {
		t.Errorf("expected height %d, got %d", TopBarHeight, height)
	}
	for line := range strings.SplitSeq(view, "\n") {
		if lipgloss.Width(line) > narrowTerminal {
			t.Errorf("expected lines to be truncated to %d, got %d", narrowTerminal, lipgloss.Width(line))
		}
	}
}

func TestTopBarView_KeepsHeightWithShortSections(t *testing.T) {
	m := NewTopBar()
	m.SetWidth(wideTerminal)

	if height := lipgloss.Height(m.View()); height != TopBarHeight {
		t.Errorf("expected height %d, got %d", TopBarHeight, height)
	}
}

func TestTopBarView_HeadedSections(t *testing.T) {
	m := NewTopBar()
	m.SetWidth(wideTerminal)
	m.SetPATCounts(1, 2)

	view := m.View()

	for _, heading := range []string{"CONTEXT", "ACCOUNTS 1/2", "SHORTCUTS"} {
		if !strings.Contains(view, heading) {
			t.Errorf("expected %q heading in top bar", heading)
		}
	}
}

func TestTopBarView_PipelineContext(t *testing.T) {
	m := NewTopBar()
	m.SetWidth(wideTerminal)
	m.SetView(ViewNamePipelineRun)
	m.SetPipelineContext("Proj/api", "deploy", "20260101.1 [running]")

	view := m.View()

	for _, expected := range []string{"Proj/api", "deploy", "20260101.1 [running]"} {
		if !strings.Contains(view, expected) {
			t.Errorf("expected %q in top bar", expected)
		}
	}
}

func TestBuildAccountsSection_MarksSelectedAndSummarisesOverflow(t *testing.T) {
	m := NewTopBar()
	accounts := make([]Account, topBarSectionRows+2)
	for i := range accounts {
		accounts[i] = Account{Name: fmt.Sprintf("pat-%d", i), Selected: i == 0}
	}
	m.SetAccounts(accounts)

	lines := m.buildAccountsSection()

	if len(lines) != topBarSectionRows {
		t.Fatalf("expected %d lines, got %d", topBarSectionRows, len(lines))
	}
	if !strings.Contains(lines[0], accountActiveMarker+"pat-0") {
		t.Errorf("expected the selected account to be marked active, got %q", lines[0])
	}
	if !strings.Contains(lines[1], accountPassiveMarker+"pat-1") {
		t.Errorf("expected an unselected account to be marked passive, got %q", lines[1])
	}
	if !strings.Contains(lines[len(lines)-1], "+3 more") {
		t.Errorf("expected an overflow summary, got %q", lines[len(lines)-1])
	}
}

func TestTruncatePathStart_KeepsTheEnd(t *testing.T) {
	got := truncatePathStart("/Users/someone/DEV/projects/very/deep/repository", 20)

	if !strings.HasSuffix(got, "repository") || !strings.HasPrefix(got, truncationTail) {
		t.Errorf("expected the path end to be kept behind %q, got %q", truncationTail, got)
	}
	if width := lipgloss.Width(got); width != 20 {
		t.Errorf("expected width 20, got %d", width)
	}
}

func TestTopBarView_ShowsAutoRefreshInPipelineViews(t *testing.T) {
	m := NewTopBar()
	m.SetWidth(wideTerminal)
	m.SetView(ViewNamePipelines)

	m.SetAutoRefresh(true, 15*time.Second)
	if view := m.View(); !strings.Contains(view, "Auto-refresh: ON every 15s") {
		t.Errorf("expected auto-refresh on in the context, got %q", view)
	}

	m.SetLastRefresh(time.Date(2026, 11, 11, 23, 44, 9, 0, time.Local))
	if view := m.View(); !strings.Contains(view, "refreshed 23:44:09") {
		t.Errorf("expected the last refresh time in the context, got %q", view)
	}

	m.SetAutoRefresh(false, 15*time.Second)
	if view := m.View(); !strings.Contains(view, "Auto-refresh: OFF") {
		t.Error("expected auto-refresh off in the context")
	}

	m.SetView(ViewNamePRList)
	if strings.Contains(m.View(), "Auto-refresh") {
		t.Error("expected no auto-refresh line outside the pipeline views")
	}
}
