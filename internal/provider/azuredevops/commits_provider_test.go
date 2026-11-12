package azuredevops

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func changeEntry(changeType, path, objectID, originalObjectID string) interface{} {
	item := map[string]interface{}{"path": path, "objectId": objectID}
	if originalObjectID != "" {
		item["originalObjectId"] = originalObjectID
	}
	return map[string]interface{}{"changeType": changeType, "item": item}
}

func TestGetCommitDiff_ReadsTheOldVersionAtTheParentWhenTheChangeHasNone(t *testing.T) {
	parent := "parent-sha"
	gitClient := &mockGitClient{
		commitChanges: &git.GitCommitChanges{Changes: &[]interface{}{
			changeEntry("edit", "/src/app.go", "new-blob", ""),
			changeEntry("add", "/src/new.go", "added-blob", ""),
			map[string]interface{}{"changeType": "edit", "item": map[string]interface{}{"path": "/src", "isFolder": true}},
		}},
		commit:        &git.GitCommit{Parents: &[]string{parent}},
		itemObjectIDs: map[string]string{"/src/app.go@" + parent: "old-blob"},
		blobContent: map[string]string{
			"old-blob":   "package app\nfunc Run() {}\n",
			"new-blob":   "package app\nfunc Run() error { return nil }\n",
			"added-blob": "package app\n",
		},
	}

	diff, err := newPullRequestTestProvider(gitClient).GetCommitDiff(context.Background(), domain.PRIdentifier{Repository: testRepository, Number: 1}, "commit-sha")
	if err != nil {
		t.Fatal(err)
	}

	if len(diff.Files) != 2 {
		t.Fatalf("expected the edited and the added file, got %+v", diff.Files)
	}
	var lines []string
	for _, line := range diff.Files[0].Hunks[0].Lines {
		lines = append(lines, line.Content)
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "-func Run() {}") || !strings.Contains(joined, "+func Run() error { return nil }") {
		t.Errorf("expected the edit against the parent's version, got %q", joined)
	}
}

func TestListCommits_ReturnsTheCommitsOldestFirst(t *testing.T) {
	first, second := "aaaaaaaa11", "bbbbbbbb22"
	gitClient := &mockGitClient{prCommits: &git.GetPullRequestCommitsResponseValue{Value: []git.GitCommitRef{
		{CommitId: &second, Comment: strPtr("Second\n\nbody"), Author: &git.GitUserDate{Name: strPtr("Erik"), Date: &azuredevops.Time{Time: time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)}}},
		{CommitId: &first, Comment: strPtr("First"), Author: &git.GitUserDate{Name: strPtr("Anna"), Date: &azuredevops.Time{Time: time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)}}},
	}}}

	commits, err := newPullRequestTestProvider(gitClient).ListCommits(context.Background(), domain.PRIdentifier{Repository: testRepository, Number: 1})
	if err != nil {
		t.Fatal(err)
	}

	if len(commits) != 2 || commits[0].ID != first || commits[1].Author.Username != "Erik" {
		t.Fatalf("unexpected commits %+v", commits)
	}
	if commits[1].Subject() != "Second" || commits[1].ShortID() != "bbbbbbbb" {
		t.Errorf("expected the subject line and a short ID, got %q %q", commits[1].Subject(), commits[1].ShortID())
	}
}
