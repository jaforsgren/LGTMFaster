package github

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-github/v57/github"
)

func testEvent(eventType, repo, payload string, createdAt time.Time) *github.Event {
	raw := json.RawMessage(payload)
	return &github.Event{
		Type:       github.String(eventType),
		Repo:       &github.Repository{Name: github.String(repo)},
		RawPayload: &raw,
		CreatedAt:  &github.Timestamp{Time: createdAt},
	}
}

func TestPushedBranches_KeepsEachRecentBranchPushOnce(t *testing.T) {
	since := time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)
	latest := since.Add(30 * time.Minute)
	events := []*github.Event{
		testEvent(pushEventType, "owner/api", `{"ref":"refs/heads/feature/login"}`, latest),
		testEvent(pushEventType, "owner/api", `{"ref":"refs/heads/feature/login"}`, since.Add(10*time.Minute)),
		testEvent(pushEventType, "owner/api", `{"ref":"refs/tags/v1.0"}`, latest),
		testEvent("CreateEvent", "owner/api", `{"ref":"feature/other"}`, latest),
		testEvent(pushEventType, "owner/web", `{"ref":"refs/heads/old"}`, since.Add(-time.Minute)),
	}

	branches := pushedBranches(events, since)

	if len(branches) != 1 {
		t.Fatalf("expected one recent branch, got %+v", branches)
	}
	branch := branches[0]
	if branch.Key() != "owner/api:feature/login" || !branch.PushedAt.Equal(latest) {
		t.Errorf("expected the latest push of owner/api:feature/login, got %+v", branch)
	}
	if branch.Repository.Owner != "owner" || branch.Repository.Name != "api" {
		t.Errorf("expected owner and name split from the full name, got %+v", branch.Repository)
	}
}
