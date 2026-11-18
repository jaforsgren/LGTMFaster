package views

import (
	"strings"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestDiffExclusions_MatchFoldersNamesAndPaths(t *testing.T) {
	exclusions := newDiffExclusions([]string{"tests/", "*.test.*", "*.props", "src/*.json", "web/e2e/"})

	cases := []struct {
		path     string
		excluded bool
	}{
		{"Api/tests/OrderTests.cs", true},
		{"Tests/OrderTests.cs", true},
		{"/Api/Tests/OrderTests.cs", true},
		{"src/testsuite/Order.cs", false},
		{"web/app.test.ts", true},
		{"Api/Order.Test.cs", true},
		{"src/contest.cs", false},
		{"Directory.Build.props", true},
		{"src/Api/Api.csproj", false},
		{"src/appsettings.json", true},
		{"lib/src/appsettings.json", false},
		{"web/e2e/login.ts", true},
		{"e2e/login.ts", false},
	}
	for _, c := range cases {
		if got := exclusions.excludes(c.path); got != c.excluded {
			t.Errorf("%s: expected excluded %v, got %v", c.path, c.excluded, got)
		}
	}
}

func TestDiffExclusions_FolderPatternsSkipFilesAtTheRoot(t *testing.T) {
	if newDiffExclusions([]string{"*/"}).excludes("README.md") {
		t.Error("expected a folder pattern to leave files outside folders alone")
	}
}

func TestNewDiffExclusions_DropsBlankAndInvalidPatterns(t *testing.T) {
	exclusions := newDiffExclusions([]string{" ", "/", "[unclosed", " *.Props "})

	if len(exclusions) != 1 || exclusions[0] != "*.props" {
		t.Errorf("expected only the valid pattern, trimmed and lower-cased, got %q", exclusions)
	}
}

func exclusionTestView() *PRInspectViewModel {
	m := NewPRInspectView()
	m.SetSize(120, 40)
	m.SetDiffExclusions([]string{"tests/", "*.props"})
	m.SetDiff(&domain.Diff{Files: []domain.FileDiff{
		{NewPath: "src/Order.cs", Hunks: []domain.DiffHunk{{Header: "@@ -1 +1 @@", Lines: []domain.DiffLine{
			{Type: "add", Content: "+class Order {}", NewLine: 1},
		}}}},
		{NewPath: "tests/OrderTests.cs", Hunks: []domain.DiffHunk{{Header: "@@ -1 +1 @@", Lines: []domain.DiffLine{
			{Type: "add", Content: "+class OrderTests {}", NewLine: 1},
		}}}},
		{NewPath: "Directory.Build.props", Hunks: []domain.DiffHunk{{Header: "@@ -1 +1 @@", Lines: []domain.DiffLine{
			{Type: "add", Content: "+<Project />", NewLine: 1},
		}}}},
	}})
	m.SwitchToDiff()
	return m
}

func TestExcludedFiles_ShownUntilToggled(t *testing.T) {
	m := exclusionTestView()

	if m.HidesExcludedFiles() || m.ShownFileCount() != 3 {
		t.Fatalf("expected every file shown by default, got %d", m.ShownFileCount())
	}
}

func TestExcludedFiles_HiddenFromTheDiffAndTheYank(t *testing.T) {
	m := exclusionTestView()

	m.ToggleExcludedFiles()

	all := m.GetAllFilesDiffText()
	if strings.Contains(all, "OrderTests") || strings.Contains(all, "Directory.Build.props") || !strings.Contains(all, "src/Order.cs") {
		t.Errorf("expected Y to copy only the shown file, got %q", all)
	}
	if strings.Contains(m.GetCurrentFileDiffText(), "OrderTests") {
		t.Error("expected y to copy a shown file")
	}
	if len(m.GetDiff().Files) != 3 {
		t.Error("expected the comment view to keep the whole diff")
	}
	if !strings.Contains(m.View(), "1/3 files, hiding "+excludedFilesLabel) {
		t.Errorf("expected the status line to say what is hidden, got %q", m.statusLine())
	}

	m.ToggleExcludedFiles()
	if m.ShownFileCount() != 3 {
		t.Error("expected toggling back to show the excluded files")
	}
}

func TestExcludedFiles_CombineWithTheExtensionFilter(t *testing.T) {
	m := exclusionTestView()
	m.ToggleExcludedFiles()

	m.SetHiddenExtensions(map[string]bool{".cs": true})

	if m.ShownFileCount() != 0 || !strings.Contains(m.View(), allFilesFilteredText) {
		t.Error("expected every file hidden by the two filters together")
	}
}
