package ui

import (
	"strings"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/ui/components"
)

func exclusionTestModel(t *testing.T, exclusions []string) Model {
	t.Helper()
	m := newPRTestModel(t, &mockProvider{})
	m.state = ViewPRInspect
	m.setDiffExclusions(exclusions)
	m.prInspect.SetDiff(&domain.Diff{Files: []domain.FileDiff{
		{NewPath: "src/Order.cs"},
		{NewPath: "tests/OrderTests.cs"},
	}})
	m.prInspect.SwitchToDiff()
	return m
}

func TestExcludedFilesKey_TogglesTheExcludedFiles(t *testing.T) {
	m := exclusionTestModel(t, []string{"tests/"})

	m.topBar.SetContext("Proj/api", "7")
	m.topBar.SetView(components.ViewNamePRDiff)

	m, _ = handleExcludedFilesKey(m)
	if !m.prInspect.HidesExcludedFiles() || m.prInspect.ShownFileCount() != 1 {
		t.Fatal("expected the test file hidden")
	}
	if !strings.Contains(m.View(), "Excluded files: HIDDEN") {
		t.Error("expected the top bar to show the excluded files hidden")
	}

	m, _ = handleExcludedFilesKey(m)
	if m.prInspect.HidesExcludedFiles() || m.prInspect.ShownFileCount() != 2 {
		t.Error("expected the test file shown again")
	}
}

func TestExcludedFilesKey_ExplainsWhereToConfigureExclusions(t *testing.T) {
	m := exclusionTestModel(t, nil)

	m, _ = handleExcludedFilesKey(m)

	if m.prInspect.HidesExcludedFiles() || !strings.Contains(m.View(), "DiffExclusions") {
		t.Error("expected a hint about DiffExclusions instead of a toggle")
	}
}

func TestNewModel_ReadsTheDiffExclusionsFromTheSettings(t *testing.T) {
	m := NewModel(&mockRepository{pats: map[string]*domain.PAT{}, settings: domain.Settings{DiffExclusions: []string{"tests/"}}})

	if !m.prInspect.HasDiffExclusions() {
		t.Error("expected the configured exclusions in PR inspect")
	}
}
