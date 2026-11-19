package github

import (
	"testing"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestTimelineActivity_ConvertsPushesCommentsAndReviews(t *testing.T) {
	at := &github.Timestamp{Time: time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC)}
	anna := &github.User{Login: github.String("anna")}
	me := &github.User{Login: github.String("Johan")}
	tests := []struct {
		event *github.Timeline
		want  domain.Activity
	}{
		{
			&github.Timeline{Event: github.String(timelineEventCommented), User: anna, CreatedAt: at, Body: github.String("Looks off\nmore")},
			domain.Activity{Kind: domain.ActivityKindComment, Summary: "commented: Looks off"},
		},
		{
			&github.Timeline{Event: github.String(timelineEventReviewed), User: anna, SubmittedAt: at, State: github.String("changes_requested")},
			domain.Activity{Kind: domain.ActivityKindReview, Summary: "requested changes"},
		},
		{
			&github.Timeline{Event: github.String(timelineEventHeadForcePushed), Actor: me, CreatedAt: at},
			domain.Activity{Kind: domain.ActivityKindPush, Summary: forcePushedSummary, ByMe: true},
		},
		{
			&github.Timeline{Event: github.String(timelineEventCommitted), Message: github.String("fix: login"),
				Author: &github.CommitAuthor{Name: github.String("johan"), Date: at}},
			domain.Activity{Kind: domain.ActivityKindPush, Summary: "pushed: fix: login", ByMe: true},
		},
	}

	for _, test := range tests {
		got, ok := timelineActivity(test.event, "johan")
		if !ok || got.Kind != test.want.Kind || got.Summary != test.want.Summary || got.ByMe != test.want.ByMe || !got.At.Equal(at.Time) {
			t.Errorf("%s: expected %+v, got %+v", test.event.GetEvent(), test.want, got)
		}
	}
}

func TestTimelineActivity_LeavesOutOtherEvents(t *testing.T) {
	if _, ok := timelineActivity(&github.Timeline{Event: github.String("labeled")}, "johan"); ok {
		t.Error("expected labels to be left out")
	}
}
