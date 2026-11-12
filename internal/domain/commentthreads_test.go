package domain

import "testing"

func TestSummarizeThreads_CountsThreadsNotReplies(t *testing.T) {
	summary := SummarizeThreads([]Comment{
		{ID: "1", ThreadID: "a", ThreadState: ThreadStateActive},
		{ID: "2", ThreadID: "a", ThreadState: ThreadStateActive},
		{ID: "3", ThreadID: "b", ThreadState: ThreadStateResolved},
		{ID: "4", ThreadID: "c", ThreadState: ThreadStateResolved},
		{ID: "5", ThreadID: "d", IsSystem: true},
	})

	if summary != (CommentThreads{Total: 3, Resolved: 2, ResolutionKnown: true}) {
		t.Errorf("unexpected summary %+v", summary)
	}
}

func TestSummarizeThreads_WithoutResolutionCountsOnly(t *testing.T) {
	summary := SummarizeThreads([]Comment{{ID: "1"}, {ID: "2", ThreadID: "1"}, {ID: "3"}})

	if summary != (CommentThreads{Total: 2}) {
		t.Errorf("expected two threads with unknown resolution, got %+v", summary)
	}
}
