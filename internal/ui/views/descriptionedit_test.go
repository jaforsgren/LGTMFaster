package views

import (
	"strings"
	"testing"
)

func TestNewDescriptionEditView_InitializesInactive(t *testing.T) {
	view := NewDescriptionEditView()

	if view.IsActive() {
		t.Error("expected new DescriptionEditView to be inactive")
	}
}

func TestDescriptionEditView_Activate(t *testing.T) {
	view := NewDescriptionEditView()
	initialDesc := "Initial PR description"

	view.Activate(initialDesc)

	if !view.IsActive() {
		t.Error("expected DescriptionEditView to be active after Activate()")
	}

	if got := view.GetDescription(); got != initialDesc {
		t.Errorf("expected description %q, got %q", initialDesc, got)
	}
}

func TestDescriptionEditView_Deactivate(t *testing.T) {
	view := NewDescriptionEditView()
	view.Activate("Some description")

	view.Deactivate()

	if view.IsActive() {
		t.Error("expected DescriptionEditView to be inactive after Deactivate()")
	}

	if got := view.GetDescription(); got != "" {
		t.Errorf("expected empty description after deactivate, got %q", got)
	}
}

func TestDescriptionEditView_SetSize(t *testing.T) {
	view := NewDescriptionEditView()

	view.SetSize(100, 50)

	if view.width != 100 {
		t.Errorf("expected width 100, got %d", view.width)
	}
	if view.height != 50 {
		t.Errorf("expected height 50, got %d", view.height)
	}
}

func TestDescriptionEditView_ViewReturnsEmptyWhenInactive(t *testing.T) {
	view := NewDescriptionEditView()
	view.SetSize(80, 24)

	output := view.View()

	if output != "" {
		t.Errorf("expected empty output when inactive, got %q", output)
	}
}

func TestDescriptionEditView_ViewShowsTitleWhenActive(t *testing.T) {
	view := NewDescriptionEditView()
	view.SetSize(80, 24)
	view.Activate("Test description")

	output := view.View()

	if !strings.Contains(output, "Edit PR Description") {
		t.Error("expected output to contain 'Edit PR Description'")
	}
}

func TestDescriptionEditView_ViewShowsHelpText(t *testing.T) {
	view := NewDescriptionEditView()
	view.SetSize(80, 24)
	view.Activate("Test description")

	output := view.View()

	if !strings.Contains(output, "Ctrl+S: Save") {
		t.Error("expected output to contain save help")
	}
	if !strings.Contains(output, "Esc: Cancel") {
		t.Error("expected output to contain cancel help")
	}
}

func TestDescriptionEditView_GetDescription(t *testing.T) {
	view := NewDescriptionEditView()
	description := "This is a multi-line\ndescription for testing."

	view.Activate(description)

	if got := view.GetDescription(); got != description {
		t.Errorf("expected description %q, got %q", description, got)
	}
}

func TestDescriptionEditView_ActivateMultipleTimes(t *testing.T) {
	view := NewDescriptionEditView()

	view.Activate("First description")
	if view.GetDescription() != "First description" {
		t.Error("expected first description to be set")
	}

	view.Activate("Second description")
	if view.GetDescription() != "Second description" {
		t.Error("expected second description to overwrite first")
	}
}
