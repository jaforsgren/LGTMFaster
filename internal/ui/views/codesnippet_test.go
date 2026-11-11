package views

import (
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestCommentIsOnLine_TellsDeletedAndAddedLinesWithTheSameNumberApart(t *testing.T) {
	deleted := domain.DiffLine{Type: diffLineTypeDelete, OldLine: 2}
	added := domain.DiffLine{Type: diffLineTypeAdd, NewLine: 2}
	onOldFile := domain.Comment{Line: 2, Side: domain.CommentSideLeft}
	onNewFile := domain.Comment{Line: 2, Side: domain.CommentSideRight}
	withoutSide := domain.Comment{Line: 2}

	if !commentIsOnLine(onOldFile, deleted) || commentIsOnLine(onOldFile, added) {
		t.Error("expected a left-side comment only on the deleted line")
	}
	if !commentIsOnLine(onNewFile, added) || commentIsOnLine(onNewFile, deleted) {
		t.Error("expected a right-side comment only on the added line")
	}
	if !commentIsOnLine(withoutSide, added) {
		t.Error("expected a comment without a side to count in the new file")
	}
}

func TestSnippetAround_StaysInsideTheFile(t *testing.T) {
	lines := fileDiffLines(testCommentFile())

	snippet := snippetAround(lines, 0)

	if snippet.focus != 0 || len(snippet.lines) != snippetLinesAfter+1 {
		t.Errorf("expected the first line and the one after, got %+v", snippet)
	}
}
