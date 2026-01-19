package views

import (
	"strings"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestNewReviewView_InitializesInactive(t *testing.T) {
	view := NewReviewView()

	if view.IsActive() {
		t.Error("expected new ReviewView to be inactive")
	}
}

func TestReviewView_Activate(t *testing.T) {
	view := NewReviewView()

	view.Activate(ReviewModeApprove)

	if !view.IsActive() {
		t.Error("expected ReviewView to be active after Activate()")
	}
}

func TestReviewView_Deactivate(t *testing.T) {
	view := NewReviewView()
	view.Activate(ReviewModeComment)

	view.Deactivate()

	if view.IsActive() {
		t.Error("expected ReviewView to be inactive after Deactivate()")
	}

	if got := view.GetValue(); got != "" {
		t.Errorf("expected empty value after deactivate, got %q", got)
	}
}

func TestReviewView_GetValue(t *testing.T) {
	view := NewReviewView()
	view.Activate(ReviewModeComment)
	view.SetValue("Test review comment")

	if got := view.GetValue(); got != "Test review comment" {
		t.Errorf("GetValue: expected %q, got %q", "Test review comment", got)
	}
}

func TestReviewView_SetValue(t *testing.T) {
	view := NewReviewView()
	view.Activate(ReviewModeComment)

	newValue := "Updated review comment"
	view.SetValue(newValue)

	if got := view.GetValue(); got != newValue {
		t.Errorf("SetValue: expected %q, got %q", newValue, got)
	}
}

func TestReviewView_GetReview_ApproveMode(t *testing.T) {
	view := NewReviewView()
	view.Activate(ReviewModeApprove)
	view.SetValue("LGTM!")

	review := view.GetReview()

	if review.Action != domain.ReviewActionApprove {
		t.Errorf("expected action %v, got %v", domain.ReviewActionApprove, review.Action)
	}
	if review.Body != "LGTM!" {
		t.Errorf("expected body %q, got %q", "LGTM!", review.Body)
	}
}

func TestReviewView_GetReview_RequestChangesMode(t *testing.T) {
	view := NewReviewView()
	view.Activate(ReviewModeRequestChanges)
	view.SetValue("Please fix the bug")

	review := view.GetReview()

	if review.Action != domain.ReviewActionRequestChanges {
		t.Errorf("expected action %v, got %v", domain.ReviewActionRequestChanges, review.Action)
	}
}

func TestReviewView_GetReview_CommentMode(t *testing.T) {
	view := NewReviewView()
	view.Activate(ReviewModeComment)
	view.SetValue("Just a comment")

	review := view.GetReview()

	if review.Action != domain.ReviewActionComment {
		t.Errorf("expected action %v, got %v", domain.ReviewActionComment, review.Action)
	}
}

func TestReviewView_ViewShowsEditorShortcut(t *testing.T) {
	view := NewReviewView()
	view.SetSize(80, 24)
	view.Activate(ReviewModeComment)

	output := view.View()

	if !strings.Contains(output, "Ctrl+G") {
		t.Error("expected output to contain 'Ctrl+G' shortcut for opening editor")
	}
}

func TestReviewView_ViewShowsCorrectTitle(t *testing.T) {
	tests := []struct {
		mode          ReviewMode
		expectedTitle string
	}{
		{ReviewModeApprove, "Approve Pull Request"},
		{ReviewModeRequestChanges, "Request Changes"},
		{ReviewModeComment, "Add Comment"},
	}

	for _, tt := range tests {
		view := NewReviewView()
		view.SetSize(80, 24)
		view.Activate(tt.mode)

		output := view.View()

		if !strings.Contains(output, tt.expectedTitle) {
			t.Errorf("expected output to contain %q for mode %v", tt.expectedTitle, tt.mode)
		}
	}
}
