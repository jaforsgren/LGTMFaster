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

	expectedHelp := "n/p/left/right: Navigate Files"
	if !contains(output, expectedHelp) {
		t.Errorf("expected help text to contain '%s'", expectedHelp)
	}

	expectedBrowserHelp := "ctrl+o: Open in Browser"
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
