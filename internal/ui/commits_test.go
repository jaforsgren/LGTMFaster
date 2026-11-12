package ui

import (
	"strings"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

func diffWithLine(path, content string) *domain.Diff {
	return &domain.Diff{Files: []domain.FileDiff{{NewPath: path, Hunks: []domain.DiffHunk{{
		Header: "@@ -0,0 +1 @@", Lines: []domain.DiffLine{{Type: "add", Content: content, NewLine: 1}},
	}}}}}
}

func TestCommitNavigation_OpensACommitAndStepsBackToThePullRequest(t *testing.T) {
	provider := &mockProvider{
		commits: []domain.Commit{{ID: "aaaaaaaa1", Message: "First"}, {ID: "bbbbbbbb2", Message: "Second\n\nbody"}},
		commitDiffs: map[string]*domain.Diff{
			"bbbbbbbb2": diffWithLine("second.go", "+second commit"),
		},
	}
	m := createTestModel()
	m.providers = map[string]domain.Provider{testPATID: provider}
	pr := domain.PullRequest{Number: 9, Repository: domain.Repo{FullName: "Proj/api"}, PATID: testPATID}
	m.prInspect.SetSize(120, 40)
	m.prInspect.SetPR(&pr)
	m.prInspect.SetDiff(diffWithLine("pr.go", "+whole pull request"))
	m.prInspect.SwitchToDiff()

	m, cmd := handleCommitsKey(m)
	m, _ = m.handleCommitsLoaded(cmd().(CommitsLoadedMsg))
	if m.prInspect.GetMode() != views.PRInspectModeCommits || !strings.Contains(m.prInspect.View(), "Second") {
		t.Fatal("expected C to list the commits")
	}

	m, _ = handleDownKey(m)
	m, cmd = handleEnterKey(m)
	m, _ = m.handleCommitDiffLoaded(cmd().(CommitDiffLoadedMsg))
	if commit := m.prInspect.ViewingCommit(); commit == nil || commit.ID != "bbbbbbbb2" {
		t.Fatalf("expected the second commit open, got %+v", commit)
	}
	if !strings.Contains(m.prInspect.GetCurrentFileDiffText(), "+second commit") {
		t.Error("expected y to copy the commit's diff")
	}
	if !strings.Contains(m.prInspect.View(), "Commit bbbbbbbb: Second") {
		t.Error("expected the commit named above its diff")
	}

	m, _ = handleInlineCommentKey(m)
	if m.inlineCommentView.IsActive() || m.statusBar.Message() != commitCommentsBlockedMessage {
		t.Error("expected inline comments blocked on a commit's diff")
	}

	m, _ = handleQuitKey(m)
	if m.prInspect.GetMode() != views.PRInspectModeCommits || m.prInspect.ViewingCommit() != nil {
		t.Fatal("expected q to go back to the commits")
	}
	m, _ = handleEscKey(m)
	if m.prInspect.GetMode() != views.PRInspectModeDiff || !strings.Contains(m.prInspect.GetCurrentFileDiffText(), "+whole pull request") {
		t.Fatal("expected the pull request's diff back")
	}
	m, _ = handleQuitKey(m)
	if m.prInspect.GetMode() != views.PRInspectModeDescription || m.state != ViewPRInspect {
		t.Error("expected q to go back to the description")
	}
}

func TestCommitsKey_FetchesCommitsOnce(t *testing.T) {
	m := createTestModel()
	m.providers = map[string]domain.Provider{testPATID: &mockProvider{commits: []domain.Commit{{ID: "a"}}}}
	pr := domain.PullRequest{Number: 9, PATID: testPATID}
	m.prInspect.SetPR(&pr)

	m, cmd := handleCommitsKey(m)
	m, _ = m.handleCommitsLoaded(cmd().(CommitsLoadedMsg))
	m.prInspect.SwitchToDiff()

	if _, cmd = handleCommitsKey(m); cmd != nil {
		t.Error("expected the loaded commits to be reused")
	}
}
