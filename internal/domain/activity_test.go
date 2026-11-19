package domain

import (
	"testing"
	"time"
)

func TestActivitySince_KeepsWhatOthersDidAfterTheTimeNewestFirst(t *testing.T) {
	since := time.Now().Add(-time.Hour)
	activity := []Activity{
		{Summary: "before", At: since.Add(-time.Minute)},
		{Summary: "older", At: since.Add(time.Minute)},
		{Summary: "mine", At: since.Add(2 * time.Minute), ByMe: true},
		{Summary: "newer", At: since.Add(3 * time.Minute)},
	}

	news := ActivitySince(activity, since)

	if len(news) != 2 || news[0].Summary != "newer" || news[1].Summary != "older" {
		t.Errorf("expected newer then older, got %+v", news)
	}
}

func TestOpenedActivity_IsTheUsersOwnOnlyOnAuthoredPRs(t *testing.T) {
	createdAt := time.Now().Add(-time.Hour)
	pr := PullRequest{Author: User{Username: "anna"}, CreatedAt: createdAt, Category: PRCategoryAssigned}

	opened := pr.OpenedActivity()
	if opened.Kind != ActivityKindOpened || opened.Author.Username != "anna" || !opened.At.Equal(createdAt) || opened.ByMe {
		t.Errorf("expected Anna's opening, got %+v", opened)
	}

	pr.Category = PRCategoryAuthored
	if !pr.OpenedActivity().ByMe {
		t.Error("expected the opening of an authored PR to be the user's own")
	}
}
