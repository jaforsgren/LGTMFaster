package azuredevops

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
)

var activityStart = time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC)

func activityTime(minutes int) *azuredevops.Time {
	return &azuredevops.Time{Time: activityStart.Add(time.Duration(minutes) * time.Minute)}
}

func activityIdentity(id, name string) *webapi.IdentityRef {
	return &webapi.IdentityRef{Id: strPtr(id), DisplayName: strPtr(name)}
}

func activityComment(id, minutes int, content string, commentType git.CommentType, author *webapi.IdentityRef) git.Comment {
	return git.Comment{Id: &id, Content: &content, CommentType: &commentType, PublishedDate: activityTime(minutes), Author: author}
}

func iteration(id, minutes int, reason git.IterationReason, pusher *webapi.IdentityRef) git.GitPullRequestIteration {
	return git.GitPullRequestIteration{Id: &id, CreatedDate: activityTime(minutes), Reason: &reason, Push: &git.GitPushRef{PushedBy: pusher}}
}

func TestListActivity_ReadsCommentsVotesAndPushesOldestFirst(t *testing.T) {
	anna, me := activityIdentity("anna-id", "Anna"), activityIdentity(testUserID, "Johan")
	deleted := activityComment(5, 4, "Oops", git.CommentTypeValues.Text, anna)
	deleted.IsDeleted = boolPtr(true)
	gitClient := &mockGitClient{
		threads: &[]git.GitPullRequestCommentThread{
			{Comments: &[]git.Comment{
				activityComment(1, 30, "Please rename\nthe second line", git.CommentTypeValues.Text, anna),
				activityComment(2, 40, "Done", git.CommentTypeValues.Text, me),
				deleted,
			}},
			{Comments: &[]git.Comment{activityComment(3, 50, "Anna voted 10", git.CommentTypeValues.System, anna)}},
			{Comments: &[]git.Comment{activityComment(4, 55, "Anna added Bob as a reviewer", git.CommentTypeValues.System, anna)}},
		},
		iterations: &[]git.GitPullRequestIteration{
			iteration(1, 0, git.IterationReasonValues.Create, me),
			iteration(2, 35, git.IterationReasonValues.ForcePush, anna),
		},
	}

	activity, err := newPullRequestTestProvider(gitClient).ListActivity(context.Background(), domain.PRIdentifier{Repository: testRepository, Number: 1})
	if err != nil {
		t.Fatal(err)
	}

	expected := []domain.Activity{
		{Kind: domain.ActivityKindPush, Summary: "opened the pull request", ByMe: true},
		{Kind: domain.ActivityKindComment, Summary: "commented: Please rename"},
		{Kind: domain.ActivityKindPush, Summary: "force-pushed"},
		{Kind: domain.ActivityKindComment, Summary: "commented: Done", ByMe: true},
		{Kind: domain.ActivityKindReview, Summary: "approved"},
	}
	if len(activity) != len(expected) {
		t.Fatalf("expected %d items, got %+v", len(expected), activity)
	}
	for i, want := range expected {
		got := activity[i]
		if got.Kind != want.Kind || got.Summary != want.Summary || got.ByMe != want.ByMe {
			t.Errorf("item %d: expected %+v, got %+v", i, want, got)
		}
	}
	if activity[1].Author.Username != "Anna" {
		t.Errorf("expected Anna as the commenter, got %+v", activity[1].Author)
	}
}

func TestListActivity_ReportsIterationErrors(t *testing.T) {
	gitClient := &mockGitClient{threads: &[]git.GitPullRequestCommentThread{}, getIterationsErr: errors.New("forbidden")}

	_, err := newPullRequestTestProvider(gitClient).ListActivity(context.Background(), domain.PRIdentifier{Repository: testRepository, Number: 1})

	if err == nil {
		t.Error("expected the iterations error")
	}
}
