package ui

import (
	"strings"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func threadedComments() []domain.Comment {
	return []domain.Comment{
		{ID: "1", ThreadID: "a", ThreadState: domain.ThreadStateResolved},
		{ID: "2", ThreadID: "b", ThreadState: domain.ThreadStateActive},
		{ID: "3", ThreadID: "c", ThreadState: domain.ThreadStateActive},
		{ID: "4", Body: "X approved", IsSystem: true},
	}
}

func TestCommentsLoaded_ShowsResolvedThreadsInTheTopBar(t *testing.T) {
	m := sizedTestModel(t, layoutTestWidth, layoutTestHeight)
	pr := domain.PullRequest{Number: 3, Repository: domain.Repo{FullName: "Proj/api"}}
	m, _ = m.inspectPR(pr)
	m.prInspect.SetPR(&pr)
	if strings.Contains(m.View(), "Comments:") {
		t.Fatal("expected no count before the comments load")
	}

	m, _ = m.handleCommentsLoaded(CommentsLoadedMsg{prKey: pr.Key(), comments: threadedComments()})

	if !strings.Contains(m.View(), "💬 Comments: 1/3 resolved") {
		t.Error("expected the resolved thread count in the top bar")
	}
}

func TestPRListInfoPane_CountsCommentsOfTheSelectedPR(t *testing.T) {
	provider := &mockProvider{}
	m := newPRTestModel(t, provider)
	pr := domain.PullRequest{Number: 3, Repository: domain.Repo{FullName: "Proj/api"}, PATID: testPATID}
	m.prListView.SetPRs([]domain.PullRequest{pr})

	m, cmd := handleTogglePreviewKey(m)
	if cmd == nil {
		t.Fatal("expected opening the Info pane to count the selected PR's comments")
	}
	if !strings.Contains(m.View(), "loading...") {
		t.Error("expected a loading hint until the count arrives")
	}
	m, _ = m.handlePRCommentThreadsLoaded(PRCommentThreadsLoadedMsg{prKey: pr.Key(), threads: domain.SummarizeThreads(threadedComments())})

	if !strings.Contains(m.View(), "💬 1/3 resolved") {
		t.Error("expected the count in the Info pane")
	}
	if m.loadPRListComments() != nil {
		t.Error("expected a loaded count not to be fetched again")
	}
}
