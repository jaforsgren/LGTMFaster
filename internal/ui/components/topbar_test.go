package components

import (
	"fmt"
	"strings"
	"testing"
)

const wideTerminal = 240

func testShortcuts(count int) []string {
	shortcuts := make([]string, count)
	for i := range shortcuts {
		shortcuts[i] = fmt.Sprintf("<k%d> Action %d", i, i)
	}
	return shortcuts
}

func TestLayoutShortcutColumns_AddsColumnsBeyondTwo(t *testing.T) {
	columns := layoutShortcutColumns(testShortcuts(13), topBarRows, wideTerminal)

	if len(columns) != 3 {
		t.Fatalf("expected 3 columns for 13 shortcuts, got %d", len(columns))
	}
	if len(columns[2].cells) != 3 {
		t.Errorf("expected last column to hold the remaining 3 shortcuts, got %d", len(columns[2].cells))
	}
}

func TestLayoutShortcutColumns_StopsWhenOutOfWidth(t *testing.T) {
	columns := layoutShortcutColumns(testShortcuts(13), topBarRows, 25)

	if len(columns) != 1 {
		t.Errorf("expected only one column to fit, got %d", len(columns))
	}
}

func TestLayoutShortcutColumns_Empty(t *testing.T) {
	if columns := layoutShortcutColumns(nil, topBarRows, wideTerminal); len(columns) != 0 {
		t.Errorf("expected no columns, got %d", len(columns))
	}
}

func TestTopBarView_RendersEveryShortcutOnWideTerminal(t *testing.T) {
	m := NewTopBar()
	m.SetWidth(wideTerminal)
	m.SetView(ViewNamePipelines)
	m.SetShortcuts(testShortcuts(13))

	view := m.View()

	for i := 0; i < 13; i++ {
		if !strings.Contains(view, fmt.Sprintf("Action %d", i)) {
			t.Errorf("expected shortcut %d in top bar", i)
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
