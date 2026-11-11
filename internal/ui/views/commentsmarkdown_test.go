package views

import (
	"strings"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestCommentsMarkdown_DocumentsEveryCommentWithItsCode(t *testing.T) {
	view := NewCommentDetailView()
	view.SetSize(120, 40)
	at := time.Date(2026, 11, 12, 9, 30, 0, 0, time.UTC)
	view.Activate([]domain.Comment{
		{Body: "Johan approved", IsSystem: true, CreatedAt: at},
		{Author: domain.User{Username: "lead"}, Body: "Nice work overall", CreatedAt: at},
		{Author: domain.User{Username: "reviewer"}, Body: "Better name", FilePath: "src/main.go", Line: 2, Side: domain.CommentSideRight, CreatedAt: at},
		{Author: domain.User{Username: "reviewer"}, Body: "Keep this?", FilePath: "src/main.go", Line: 2, Side: domain.CommentSideLeft, CreatedAt: at},
	}, &domain.Diff{Files: []domain.FileDiff{testCommentFile()}})
	pr := &domain.PullRequest{Number: 42, Title: "Rename", Repository: domain.Repo{FullName: "Proj/api"}, SourceBranch: "feature", TargetBranch: "main", URL: "https://pr"}

	markdown := view.Markdown(pr)

	for _, expected := range []string{
		"# #42 Rename",
		"Proj/api · `feature` → `main` · 4 comments",
		"<https://pr>",
		"## Review comments",
		"- **Johan approved** · 2026-11-12 09:30",
		"### lead · 2026-11-12 09:30\n\nNice work overall",
		"## Inline comments",
		"### `src/main.go`",
		"#### Line 2 · reviewer",
		"#### Line 2 (deleted line) · reviewer",
		"```diff\n package main\n-func old() {}\n+func renamed() {}\n func main() {}\n```",
		"Better name",
	} {
		if !strings.Contains(markdown, expected) {
			t.Errorf("expected %q in the markdown, got:\n%s", expected, markdown)
		}
	}
}

func TestCommentsMarkdown_SaysWhenThereAreNoComments(t *testing.T) {
	view := NewCommentDetailView()
	view.Activate(nil, nil)

	if markdown := view.Markdown(nil); !strings.Contains(markdown, markdownEmptyNotice) {
		t.Errorf("expected the empty notice, got %q", markdown)
	}
}
