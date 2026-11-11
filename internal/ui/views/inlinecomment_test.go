package views

import (
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"strings"
	"testing"
)

func TestNewInlineCommentView_InitializesInactive(t *testing.T) {
	view := NewInlineCommentView()

	if view.IsActive() {
		t.Error("expected new InlineCommentView to be inactive")
	}
}

func TestInlineCommentView_Activate(t *testing.T) {
	view := NewInlineCommentView()

	view.Activate(CommentTarget{FilePath: "file.go", Line: 42})

	if !view.IsActive() {
		t.Error("expected InlineCommentView to be active after Activate()")
	}
}

func TestInlineCommentView_Deactivate(t *testing.T) {
	view := NewInlineCommentView()
	view.Activate(CommentTarget{FilePath: "file.go", Line: 42})

	view.Deactivate()

	if view.IsActive() {
		t.Error("expected InlineCommentView to be inactive after Deactivate()")
	}

	if got := view.GetComment(); got != "" {
		t.Errorf("expected empty comment after deactivate, got %q", got)
	}
}

func TestInlineCommentView_GetValue(t *testing.T) {
	view := NewInlineCommentView()
	view.Activate(CommentTarget{FilePath: "file.go", Line: 42})
	view.SetValue("Test inline comment")

	if got := view.GetValue(); got != "Test inline comment" {
		t.Errorf("GetValue: expected %q, got %q", "Test inline comment", got)
	}
}

func TestInlineCommentView_SetValue(t *testing.T) {
	view := NewInlineCommentView()
	view.Activate(CommentTarget{FilePath: "file.go", Line: 42})

	newValue := "Updated inline comment"
	view.SetValue(newValue)

	if got := view.GetValue(); got != newValue {
		t.Errorf("SetValue: expected %q, got %q", newValue, got)
	}
}

func TestInlineCommentView_GetComment(t *testing.T) {
	view := NewInlineCommentView()
	view.Activate(CommentTarget{FilePath: "file.go", Line: 42})
	view.SetValue("This is a comment")

	if got := view.GetComment(); got != "This is a comment" {
		t.Errorf("GetComment: expected %q, got %q", "This is a comment", got)
	}
}

func TestInlineCommentView_ViewShowsEditorShortcut(t *testing.T) {
	view := NewInlineCommentView()
	view.SetSize(80, 24)
	view.Activate(CommentTarget{FilePath: "file.go", Line: 42})

	output := view.View()

	if !strings.Contains(output, "Ctrl+G") {
		t.Error("expected output to contain 'Ctrl+G' shortcut for opening editor")
	}
}

func testCommentFile() domain.FileDiff {
	return domain.FileDiff{
		NewPath: "src/main.go",
		Hunks: []domain.DiffHunk{{Lines: []domain.DiffLine{
			{Type: "context", Content: " package main", OldLine: 1, NewLine: 1},
			{Type: "delete", Content: "-func old() {}", OldLine: 2},
			{Type: "add", Content: "+func renamed() {}", NewLine: 2},
			{Type: "context", Content: " func main() {}", OldLine: 3, NewLine: 3},
		}}},
	}
}

func TestInlineCommentView_ShowsTheLineAndTheCodeAroundIt(t *testing.T) {
	target, ok := commentTargetAt(testCommentFile(), 2)
	if !ok {
		t.Fatal("expected a target for the added line")
	}
	view := NewInlineCommentView()
	view.SetSize(80, 24)
	view.Activate(target)

	output := view.View()

	for _, expected := range []string{"src/main.go · line 2", "package main", "+func renamed() {}", snippetFocusMarker} {
		if !strings.Contains(output, expected) {
			t.Errorf("expected %q in the inline comment box", expected)
		}
	}
}

func TestCommentTargetAt_DeletedLineCountsInTheOldFile(t *testing.T) {
	target, _ := commentTargetAt(testCommentFile(), 1)

	if target.Line != 2 || target.Side != domain.CommentSideLeft {
		t.Errorf("expected line 2 of the old file, got %+v", target)
	}
	if !strings.Contains(target.Describe(), "(deleted line)") {
		t.Errorf("expected the description to say it is a deleted line, got %q", target.Describe())
	}
}

func TestInlineCommentView_ViewShowsTitle(t *testing.T) {
	view := NewInlineCommentView()
	view.SetSize(80, 24)
	view.Activate(CommentTarget{FilePath: "file.go", Line: 42})

	output := view.View()

	if !strings.Contains(output, "Add Inline Comment") {
		t.Error("expected output to contain 'Add Inline Comment'")
	}
}
