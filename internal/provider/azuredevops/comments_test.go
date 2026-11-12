package azuredevops

import (
	"context"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func testThreadComment(id int, content string, commentType git.CommentType) git.Comment {
	now := &azuredevops.Time{Time: time.Now()}
	return git.Comment{Id: &id, Content: &content, CommentType: &commentType, PublishedDate: now, LastUpdatedDate: now}
}

func TestDescribeVote(t *testing.T) {
	tests := map[string]string{
		"Johan Forsgren voted 10":                    "Johan Forsgren approved",
		"Johan Forsgren voted 5":                     "Johan Forsgren approved with suggestions",
		"Johan Forsgren voted 0":                     "Johan Forsgren reset their vote",
		"Johan Forsgren voted -5":                    "Johan Forsgren is waiting for the author",
		"Johan Forsgren voted -10":                   "Johan Forsgren rejected",
		"Johan Forsgren voted 3":                     "Johan Forsgren voted 3",
		"The reference refs/heads/main was updated.": "The reference refs/heads/main was updated.",
	}
	for content, expected := range tests {
		if got := describeVote(content); got != expected {
			t.Errorf("describeVote(%q) = %q, want %q", content, got, expected)
		}
	}
}

func TestGetComments_CarriesTheThreadAndItsResolution(t *testing.T) {
	activeID, fixedID := 1, 2
	active, fixed := git.CommentThreadStatusValues.Active, git.CommentThreadStatusValues.Fixed
	gitClient := &mockGitClient{threads: &[]git.GitPullRequestCommentThread{
		{Id: &activeID, Status: &active, Comments: &[]git.Comment{
			testThreadComment(1, "Question", git.CommentTypeValues.Text),
			testThreadComment(2, "Reply", git.CommentTypeValues.Text),
		}},
		{Id: &fixedID, Status: &fixed, Comments: &[]git.Comment{testThreadComment(3, "Done", git.CommentTypeValues.Text)}},
	}}

	comments, err := newPullRequestTestProvider(gitClient).GetComments(context.Background(), domain.PRIdentifier{Repository: testRepository, Number: 1})
	if err != nil {
		t.Fatal(err)
	}

	if comments[1].ThreadID != "1" || comments[1].ThreadState != domain.ThreadStateActive {
		t.Errorf("expected the reply in the active thread 1, got %+v", comments[1])
	}
	if summary := domain.SummarizeThreads(comments); summary.Resolved != 1 || summary.Total != 2 {
		t.Errorf("expected 1 of 2 threads resolved, got %+v", summary)
	}
}

func TestGetComments_TranslatesVotesAndNormalisesPositions(t *testing.T) {
	right, left := 12, 4
	path := "/src/app.go"
	gitClient := &mockGitClient{threads: &[]git.GitPullRequestCommentThread{
		{Comments: &[]git.Comment{testThreadComment(1, "Johan voted 10", git.CommentTypeValues.System)}},
		{
			ThreadContext: &git.CommentThreadContext{FilePath: &path, RightFileStart: &git.CommentPosition{Line: &right}},
			Comments:      &[]git.Comment{testThreadComment(2, "Rename this", git.CommentTypeValues.Text)},
		},
		{
			ThreadContext: &git.CommentThreadContext{FilePath: &path, LeftFileStart: &git.CommentPosition{Line: &left}},
			Comments:      &[]git.Comment{testThreadComment(3, "Why delete?", git.CommentTypeValues.Text)},
		},
	}}

	comments, err := newPullRequestTestProvider(gitClient).GetComments(context.Background(), domain.PRIdentifier{Repository: testRepository, Number: 1})
	if err != nil {
		t.Fatal(err)
	}

	if comments[0].Body != "Johan approved" || !comments[0].IsSystem {
		t.Errorf("expected the vote translated and marked as system, got %+v", comments[0])
	}
	if comments[1].FilePath != "src/app.go" || comments[1].Line != right || comments[1].Side != domain.CommentSideRight {
		t.Errorf("expected a right-side comment on src/app.go:12, got %+v", comments[1])
	}
	if comments[2].Line != left || comments[2].Side != domain.CommentSideLeft || comments[2].IsSystem {
		t.Errorf("expected a left-side comment on line 4, got %+v", comments[2])
	}
}
