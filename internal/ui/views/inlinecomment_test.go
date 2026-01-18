package views

import (
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

	view.Activate("file.go:42")

	if !view.IsActive() {
		t.Error("expected InlineCommentView to be active after Activate()")
	}
}

func TestInlineCommentView_Deactivate(t *testing.T) {
	view := NewInlineCommentView()
	view.Activate("file.go:42")

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
	view.Activate("file.go:42")
	view.SetValue("Test inline comment")

	if got := view.GetValue(); got != "Test inline comment" {
		t.Errorf("GetValue: expected %q, got %q", "Test inline comment", got)
	}
}

func TestInlineCommentView_SetValue(t *testing.T) {
	view := NewInlineCommentView()
	view.Activate("file.go:42")

	newValue := "Updated inline comment"
	view.SetValue(newValue)

	if got := view.GetValue(); got != newValue {
		t.Errorf("SetValue: expected %q, got %q", newValue, got)
	}
}

func TestInlineCommentView_GetComment(t *testing.T) {
	view := NewInlineCommentView()
	view.Activate("file.go:42")
	view.SetValue("This is a comment")

	if got := view.GetComment(); got != "This is a comment" {
		t.Errorf("GetComment: expected %q, got %q", "This is a comment", got)
	}
}

func TestInlineCommentView_ViewShowsEditorShortcut(t *testing.T) {
	view := NewInlineCommentView()
	view.SetSize(80, 24)
	view.Activate("file.go:42")

	output := view.View()

	if !strings.Contains(output, "Ctrl+G") {
		t.Error("expected output to contain 'Ctrl+G' shortcut for opening editor")
	}
}

func TestInlineCommentView_ViewShowsLineInfo(t *testing.T) {
	view := NewInlineCommentView()
	view.SetSize(80, 24)
	view.Activate("src/main.go:123")

	output := view.View()

	if !strings.Contains(output, "src/main.go:123") {
		t.Error("expected output to contain line info")
	}
}

func TestInlineCommentView_ViewShowsTitle(t *testing.T) {
	view := NewInlineCommentView()
	view.SetSize(80, 24)
	view.Activate("file.go:42")

	output := view.View()

	if !strings.Contains(output, "Add Inline Comment") {
		t.Error("expected output to contain 'Add Inline Comment'")
	}
}
