package views

import (
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestNewPRInspectView_InitializesDescriptionMode(t *testing.T) {
	view := NewPRInspectView()

	if view.mode != PRInspectModeDescription {
		t.Errorf("expected mode to be PRInspectModeDescription, got %v", view.mode)
	}
}

func TestSwitchToDiff(t *testing.T) {
	view := NewPRInspectView()
	view.SwitchToDiff()

	if view.mode != PRInspectModeDiff {
		t.Errorf("expected mode to be PRInspectModeDiff, got %v", view.mode)
	}
}

func TestSwitchToDescription(t *testing.T) {
	view := NewPRInspectView()
	view.SwitchToDiff()
	view.SwitchToDescription()

	if view.mode != PRInspectModeDescription {
		t.Errorf("expected mode to be PRInspectModeDescription, got %v", view.mode)
	}
}

func TestGetMode(t *testing.T) {
	view := NewPRInspectView()

	if view.GetMode() != PRInspectModeDescription {
		t.Errorf("expected GetMode to return PRInspectModeDescription, got %v", view.GetMode())
	}

	view.SwitchToDiff()

	if view.GetMode() != PRInspectModeDiff {
		t.Errorf("expected GetMode to return PRInspectModeDiff, got %v", view.GetMode())
	}
}

func TestSetPR_ResetsModeToDescription(t *testing.T) {
	view := NewPRInspectView()
	view.SwitchToDiff()

	pr := &domain.PullRequest{
		ID:     "test-pr",
		Number: 1,
		Title:  "Test PR",
	}

	view.SetPR(pr)

	if view.mode != PRInspectModeDescription {
		t.Errorf("expected mode to reset to PRInspectModeDescription after SetPR, got %v", view.mode)
	}
}

func TestView_ShowsCorrectHelpTextForDescriptionMode(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	output := view.View()

	if len(output) == 0 {
		t.Error("expected View to return non-empty output")
	}

	expectedHelp := "d: View Diff"
	if !contains(output, expectedHelp) {
		t.Errorf("expected help text to contain '%s'", expectedHelp)
	}

	expectedBrowserHelp := "ctrl+o: Open in Browser"
	if !contains(output, expectedBrowserHelp) {
		t.Errorf("expected help text to contain '%s'", expectedBrowserHelp)
	}
}

func TestView_ShowsCorrectHelpTextForDiffMode(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)
	view.SwitchToDiff()

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "test.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: "add", Content: "+test"},
						},
					},
				},
			},
		},
	}
	view.SetDiff(diff)

	output := view.View()

	expectedHelp := "Files: n/p"
	if !contains(output, expectedHelp) {
		t.Errorf("expected help text to contain '%s'", expectedHelp)
	}

	expectedBrowserHelp := "ctrl+o: Browser"
	if !contains(output, expectedBrowserHelp) {
		t.Errorf("expected help text to contain '%s'", expectedBrowserHelp)
	}
}

func TestSetDiff_ResetsToDescriptionModeIsPreserved(t *testing.T) {
	view := NewPRInspectView()
	view.SwitchToDiff()

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{NewPath: "test.go"},
		},
	}

	view.SetDiff(diff)

	if view.mode != PRInspectModeDiff {
		t.Errorf("expected mode to remain PRInspectModeDiff after SetDiff, got %v", view.mode)
	}

	if view.currentFile != 0 {
		t.Errorf("expected currentFile to be 0 after SetDiff, got %d", view.currentFile)
	}
}

func TestRenderingInDescriptionMode_OnlyShowsPRHeader(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	pr := &domain.PullRequest{
		ID:          "test-pr",
		Number:      42,
		Title:       "Test PR Title",
		Description: "Test description",
		Repository: domain.Repo{
			FullName: "owner/repo",
		},
		SourceBranch: "feature",
		TargetBranch: "main",
		Author: domain.User{
			Username: "testuser",
		},
		Status: domain.PRStatusOpen,
	}

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "test.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: "add", Content: "+test line"},
						},
					},
				},
			},
		},
	}

	view.SetPR(pr)
	view.SetDiff(diff)

	output := view.View()

	if !contains(output, "Test PR Title") {
		t.Error("expected description mode to show PR title")
	}

	if !contains(output, "Test description") {
		t.Error("expected description mode to show PR description")
	}

	if contains(output, "+test line") {
		t.Error("expected description mode to NOT show diff content")
	}
}

func TestRenderingInDiffMode_OnlyShowsDiff(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	pr := &domain.PullRequest{
		ID:          "test-pr",
		Number:      42,
		Title:       "Test PR Title",
		Description: "Test description",
		Repository: domain.Repo{
			FullName: "owner/repo",
		},
		SourceBranch: "feature",
		TargetBranch: "main",
		Author: domain.User{
			Username: "testuser",
		},
		Status: domain.PRStatusOpen,
	}

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "test.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: "add", Content: "+test line"},
						},
					},
				},
			},
		},
	}

	view.SetPR(pr)
	view.SetDiff(diff)
	view.SwitchToDiff()

	output := view.View()

	if contains(output, "Test PR Title") {
		t.Error("expected diff mode to NOT show PR title")
	}

	if contains(output, "Test description") {
		t.Error("expected diff mode to NOT show PR description")
	}

	if !contains(output, "+test line") {
		t.Error("expected diff mode to show diff content")
	}

	if !contains(output, "File 1/1: test.go") {
		t.Error("expected diff mode to show file header")
	}
}

func TestModeSwitchingPreservesFilePosition(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{NewPath: "file1.go"},
			{NewPath: "file2.go"},
			{NewPath: "file3.go"},
		},
	}

	view.SetDiff(diff)
	view.SwitchToDiff()
	view.NextFile()
	view.NextFile()

	if view.currentFile != 2 {
		t.Errorf("expected currentFile to be 2, got %d", view.currentFile)
	}

	view.SwitchToDescription()
	view.SwitchToDiff()

	if view.currentFile != 2 {
		t.Errorf("expected currentFile to remain 2 after mode switch, got %d", view.currentFile)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDiffViewMode_DefaultsToFull(t *testing.T) {
	view := NewPRInspectView()

	if view.GetDiffViewMode() != DiffViewModeFull {
		t.Errorf("expected default diff view mode to be DiffViewModeFull, got %v", view.GetDiffViewMode())
	}
}

func TestDiffViewMode_Toggle(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	view.ToggleDiffViewMode()
	if view.GetDiffViewMode() != DiffViewModeCompact {
		t.Errorf("expected diff view mode to be DiffViewModeCompact after toggle, got %v", view.GetDiffViewMode())
	}

	view.ToggleDiffViewMode()
	if view.GetDiffViewMode() != DiffViewModeFull {
		t.Errorf("expected diff view mode to be DiffViewModeFull after second toggle, got %v", view.GetDiffViewMode())
	}
}

func TestDiffViewMode_CompactHidesContextLines(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "test.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,5 +1,5 @@",
						Lines: []domain.DiffLine{
							{Type: "context", Content: " context line 1"},
							{Type: "delete", Content: "-deleted line"},
							{Type: "add", Content: "+added line"},
							{Type: "context", Content: " context line 2"},
						},
					},
				},
			},
		},
	}

	view.SetDiff(diff)
	view.SwitchToDiff()

	fullOutput := view.View()
	if !contains(fullOutput, "context line 1") {
		t.Error("expected full mode to show context lines")
	}

	view.ToggleDiffViewMode()
	compactOutput := view.View()

	if contains(compactOutput, "context line 1") {
		t.Error("expected compact mode to hide context lines")
	}
	if !contains(compactOutput, "-deleted line") {
		t.Error("expected compact mode to show deleted lines")
	}
	if !contains(compactOutput, "+added line") {
		t.Error("expected compact mode to show added lines")
	}
}

func TestGetCurrentFileDiffText_ReturnsEmptyWhenNoDiff(t *testing.T) {
	view := NewPRInspectView()

	result := view.GetCurrentFileDiffText()
	if result != "" {
		t.Errorf("expected empty string when no diff, got %q", result)
	}
}

func TestGetCurrentFileDiffText_ReturnsCurrentFileContent(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "file1.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: "add", Content: "+line in file1"},
						},
					},
				},
			},
			{
				NewPath: "file2.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: "add", Content: "+line in file2"},
						},
					},
				},
			},
		},
	}

	view.SetDiff(diff)
	view.SwitchToDiff()

	result := view.GetCurrentFileDiffText()
	if !contains(result, "+line in file1") {
		t.Error("expected current file diff text to contain file1 content")
	}
	if contains(result, "+line in file2") {
		t.Error("expected current file diff text to NOT contain file2 content")
	}

	view.NextFile()
	result = view.GetCurrentFileDiffText()
	if !contains(result, "+line in file2") {
		t.Error("expected current file diff text to contain file2 content after NextFile")
	}
}

func TestGetCurrentFileDiffText_RespectsCompactMode(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "test.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,3 +1,3 @@",
						Lines: []domain.DiffLine{
							{Type: "context", Content: " context"},
							{Type: "add", Content: "+added"},
							{Type: "context", Content: " more context"},
						},
					},
				},
			},
		},
	}

	view.SetDiff(diff)
	view.SwitchToDiff()

	fullResult := view.GetCurrentFileDiffText()
	if !contains(fullResult, " context") {
		t.Error("expected full mode diff text to contain context lines")
	}

	view.ToggleDiffViewMode()
	compactResult := view.GetCurrentFileDiffText()
	if contains(compactResult, " context") {
		t.Error("expected compact mode diff text to NOT contain context lines")
	}
	if !contains(compactResult, "+added") {
		t.Error("expected compact mode diff text to contain added lines")
	}
}

func TestGetAllFilesDiffText_ReturnsEmptyWhenNoDiff(t *testing.T) {
	view := NewPRInspectView()

	result := view.GetAllFilesDiffText()
	if result != "" {
		t.Errorf("expected empty string when no diff, got %q", result)
	}
}

func TestGetAllFilesDiffText_IncludesAllFilesWithHeaders(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "file1.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: "add", Content: "+content1"},
						},
					},
				},
			},
			{
				NewPath: "file2.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: "add", Content: "+content2"},
						},
					},
				},
			},
		},
	}

	view.SetDiff(diff)
	view.SwitchToDiff()

	result := view.GetAllFilesDiffText()

	if !contains(result, "=== file1.go ===") {
		t.Error("expected all files diff text to contain file1.go header")
	}
	if !contains(result, "=== file2.go ===") {
		t.Error("expected all files diff text to contain file2.go header")
	}
	if !contains(result, "+content1") {
		t.Error("expected all files diff text to contain file1 content")
	}
	if !contains(result, "+content2") {
		t.Error("expected all files diff text to contain file2 content")
	}
}

func TestClampViewportOffset_PreventsNegativeOffset(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "test.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: "add", Content: "+test"},
						},
					},
				},
			},
		},
	}

	view.SetDiff(diff)
	view.SwitchToDiff()

	view.viewport.YOffset = -10
	view.clampViewportOffset()

	if view.viewport.YOffset < 0 {
		t.Errorf("expected YOffset to be >= 0 after clamp, got %d", view.viewport.YOffset)
	}
}

func TestClampViewportOffset_PreventsExcessiveOffset(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "test.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: "add", Content: "+test"},
						},
					},
				},
			},
		},
	}

	view.SetDiff(diff)
	view.SwitchToDiff()

	view.viewport.YOffset = 10000
	view.clampViewportOffset()

	maxAllowed := max(view.contentLines-view.viewport.Height, 0)

	if view.viewport.YOffset > maxAllowed {
		t.Errorf("expected YOffset to be <= %d after clamp, got %d", maxAllowed, view.viewport.YOffset)
	}
}

func TestContentLinesTracking(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "test.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,5 +1,5 @@",
						Lines: []domain.DiffLine{
							{Type: "context", Content: " line1"},
							{Type: "context", Content: " line2"},
							{Type: "add", Content: "+line3"},
							{Type: "context", Content: " line4"},
							{Type: "context", Content: " line5"},
						},
					},
				},
			},
		},
	}

	view.SetDiff(diff)
	view.SwitchToDiff()

	if view.contentLines <= 0 {
		t.Error("expected contentLines to be tracked after setting diff")
	}

	initialLines := view.contentLines

	view.ToggleDiffViewMode()

	if view.contentLines >= initialLines {
		t.Error("expected contentLines to decrease in compact mode")
	}
}

func TestHelpText_ShowsToggleViewInfo(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "test.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: "add", Content: "+test"},
						},
					},
				},
			},
		},
	}

	view.SetDiff(diff)
	view.SwitchToDiff()

	output := view.View()
	if !contains(output, "f: Toggle view") {
		t.Error("expected help text to show toggle view shortcut")
	}
	if !contains(output, "(full)") {
		t.Error("expected help text to show current view mode (full)")
	}

	view.ToggleDiffViewMode()
	output = view.View()
	if !contains(output, "(compact)") {
		t.Error("expected help text to show current view mode (compact)")
	}
}

func TestHelpText_ShowsYankShortcuts(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "test.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: "add", Content: "+test"},
						},
					},
				},
			},
		},
	}

	view.SetDiff(diff)
	view.SwitchToDiff()

	output := view.View()
	if !contains(output, "y/Y: Yank") {
		t.Error("expected help text to show yank shortcuts")
	}
}

func TestHelpText_ShowsEditDescriptionInDescriptionMode(t *testing.T) {
	view := NewPRInspectView()
	view.SetSize(80, 24)

	output := view.View()
	if !contains(output, "e: Edit Description") {
		t.Error("expected help text to show edit description shortcut in description mode")
	}
}
