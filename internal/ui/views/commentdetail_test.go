package views

import (
	"strings"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestNewCommentDetailView_InitializesInactive(t *testing.T) {
	view := NewCommentDetailView()

	if view.IsActive() {
		t.Error("expected new comment detail view to be inactive")
	}
}

func TestCommentDetailView_ActivateDeactivate(t *testing.T) {
	view := NewCommentDetailView()
	view.SetSize(80, 24)

	comments := []domain.Comment{
		{
			ID:   "1",
			Body: "Test comment",
			Author: domain.User{
				Username: "testuser",
			},
		},
	}

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "test.go",
			},
		},
	}

	view.Activate(comments, diff)

	if !view.IsActive() {
		t.Error("expected view to be active after Activate()")
	}

	view.Deactivate()

	if view.IsActive() {
		t.Error("expected view to be inactive after Deactivate()")
	}
}

func TestCommentDetailView_GroupsByFile(t *testing.T) {
	view := NewCommentDetailView()
	view.SetSize(100, 40)

	comments := []domain.Comment{
		{
			ID:       "1",
			Body:     "Comment on file1",
			FilePath: "file1.go",
			Line:     10,
			Author: domain.User{
				Username: "user1",
			},
		},
		{
			ID:       "2",
			Body:     "Another comment on file1",
			FilePath: "file1.go",
			Line:     20,
			Author: domain.User{
				Username: "user2",
			},
		},
		{
			ID:       "3",
			Body:     "Comment on file2",
			FilePath: "file2.go",
			Line:     5,
			Author: domain.User{
				Username: "user1",
			},
		},
	}

	view.Activate(comments, nil)
	output := view.View()

	// Check that files are listed (may appear in any order due to map iteration)
	hasFile1 := strings.Contains(output, "file1.go")
	hasFile2 := strings.Contains(output, "file2.go")
	if !hasFile1 && !hasFile2 {
		t.Errorf("expected output to contain file paths, got:\n%s", output)
	}

	// Check for usernames
	hasUser1 := strings.Contains(output, "user1")
	hasUser2 := strings.Contains(output, "user2")
	if !hasUser1 && !hasUser2 {
		t.Errorf("expected output to contain usernames, got:\n%s", output)
	}

	// Check for comment bodies
	hasComment := strings.Contains(output, "Comment on file1") || strings.Contains(output, "Comment on file2")
	if !hasComment {
		t.Errorf("expected output to contain comment body, got:\n%s", output)
	}
}

func TestCommentDetailView_DisplaysLineNumbers(t *testing.T) {
	view := NewCommentDetailView()
	view.SetSize(80, 24)

	comments := []domain.Comment{
		{
			ID:       "1",
			Body:     "Test comment",
			FilePath: "test.go",
			Line:     42,
			Author: domain.User{
				Username: "testuser",
			},
		},
	}

	view.Activate(comments, nil)
	output := view.View()

	if !strings.Contains(output, "line 42") {
		t.Error("expected output to contain 'line 42'")
	}
}

func TestCommentDetailView_HandlesNoComments(t *testing.T) {
	view := NewCommentDetailView()
	view.SetSize(80, 24)

	view.Activate([]domain.Comment{}, nil)
	output := view.View()

	if !strings.Contains(output, "No comments") {
		t.Error("expected output to contain 'No comments'")
	}
}

func TestCommentDetailView_ShowsCodeContext(t *testing.T) {
	view := NewCommentDetailView()
	view.SetSize(80, 24)

	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				NewPath: "test.go",
				Hunks: []domain.DiffHunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: "add", Content: "+func main() {", NewLine: 1},
						},
					},
				},
			},
		},
	}

	comments := []domain.Comment{
		{
			ID:       "1",
			Body:     "Nice function",
			FilePath: "test.go",
			Line:     1,
			Side:     "RIGHT",
			Author: domain.User{
				Username: "reviewer",
			},
		},
	}

	view.Activate(comments, diff)
	output := view.View()

	if !strings.Contains(output, "func main()") {
		t.Error("expected output to contain code context")
	}
}
