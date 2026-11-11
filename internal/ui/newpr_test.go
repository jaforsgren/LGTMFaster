package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func newPRTestModel(t *testing.T, provider *mockProvider) Model {
	t.Helper()
	m := sizedTestModel(t, layoutTestWidth, layoutTestHeight)
	m.state = ViewPRList
	m.providers = map[string]domain.Provider{testPATID: provider}
	m.primaryPATID = testPATID
	return m
}

func recentTestBranches() []domain.PushedBranch {
	return []domain.PushedBranch{
		{Repository: domain.Repo{FullName: "Proj/api"}, Branch: "feature/login", PushedAt: time.Now().Add(-5 * time.Minute), PATID: testPATID},
		{Repository: domain.Repo{FullName: "Proj/web"}, Branch: "fix/css", PushedAt: time.Now().Add(-20 * time.Minute), PATID: testPATID},
	}
}

func typeCommand(t *testing.T, m Model, text string) (Model, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for _, r := range text {
		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		if r == ' ' {
			key = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{r}}
		}
		var next tea.Cmd
		m, next = m.handleCommandInput(key)
		if next != nil {
			cmd = next
		}
	}
	return m, cmd
}

func TestPRCommandArgument_FetchesAndSuggestsRecentBranches(t *testing.T) {
	provider := &mockProvider{pushedBranches: recentTestBranches()}
	m := newPRTestModel(t, provider)
	m.commandBar.Activate()

	m, cmd := typeCommand(t, m, "pr ")
	if !m.recentBranches.loading || cmd == nil {
		t.Fatal("expected starting a :pr argument to fetch the recent branches")
	}
	if !strings.Contains(m.helpLine(), "loading branches") {
		t.Errorf("expected a loading hint, got %q", m.helpLine())
	}

	msg := m.loadRecentBranches()().(RecentBranchesLoadedMsg)
	m, _ = m.handleRecentBranchesLoaded(msg)

	help := m.helpLine()
	for _, expected := range []string{"Proj/api:feature/login", "Proj/web:fix/css"} {
		if !strings.Contains(help, expected) {
			t.Errorf("expected %q suggested, got %q", expected, help)
		}
	}
	if msg.branches[0].PATID != testPATID {
		t.Error("expected the branches tagged with their PAT")
	}
}

func TestPRCommandArgument_TabCyclesTheMatchingBranches(t *testing.T) {
	m := newPRTestModel(t, &mockProvider{})
	m.recentBranches = recentBranches{branches: recentTestBranches(), fetchedAt: time.Now()}
	m.commandBar.Activate()
	m, _ = typeCommand(t, m, "pr ")

	m = m.completeCommand()
	if got := m.commandBar.Input(); got != "pr Proj/api:feature/login" {
		t.Fatalf("expected the newest branch first, got %q", got)
	}
	m = m.completeCommand()
	if got := m.commandBar.Input(); got != "pr Proj/web:fix/css" {
		t.Errorf("expected tab to move to the next branch, got %q", got)
	}
}

func TestPRCommandArgument_FiltersByTypedTextAndSkipsBranchesWithAPullRequest(t *testing.T) {
	m := newPRTestModel(t, &mockProvider{})
	m.recentBranches = recentBranches{branches: recentTestBranches(), fetchedAt: time.Now()}
	m.prCache = &PRCache{AllPRs: []domain.PullRequest{{Repository: domain.Repo{FullName: "Proj/web"}, SourceBranch: "fix/css"}}}
	m.commandBar.Activate()

	m, _ = typeCommand(t, m, "pr ")
	if suggestions := m.branchSuggestions(); len(suggestions) != 1 || suggestions[0].Branch != "feature/login" {
		t.Errorf("expected only the branch without a pull request, got %+v", suggestions)
	}

	m, _ = typeCommand(t, m, "fix")
	if suggestions := m.branchSuggestions(); len(suggestions) != 0 {
		t.Errorf("expected no match for fix once its branch has a pull request, got %+v", suggestions)
	}
}

func TestNewPR_CreatesThePullRequestAndOpensIt(t *testing.T) {
	created := &domain.PullRequest{Number: 7, Title: "Login", Repository: domain.Repo{FullName: "Proj/api"}}
	provider := &mockProvider{
		branches:  domain.RepositoryBranches{Default: "main", Names: []string{"main", "feature/login"}},
		createdPR: created,
	}
	m := newPRTestModel(t, provider)
	m.recentBranches = recentBranches{branches: recentTestBranches(), fetchedAt: time.Now()}

	m, cmd := handlePRCommand(m, []string{"Proj/api:feature/login"})
	if !m.newPRView.IsActive() {
		t.Fatal("expected :pr with a branch to open the form")
	}
	m, _ = m.handleNewPRBranchesLoaded(cmd().(NewPRBranchesLoadedMsg))

	m, cmd = m.handleNewPRKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	msg, ok := cmd().(PullRequestCreatedMsg)
	if !ok {
		t.Fatalf("expected the pull request to be created, got %#v", cmd())
	}
	if provider.lastNewPR == nil || provider.lastNewPR.TargetBranch != "main" || provider.lastNewPR.SourceBranch != "feature/login" {
		t.Errorf("unexpected request %+v", provider.lastNewPR)
	}

	m, _ = m.handlePullRequestCreated(msg)
	if m.newPRView.IsActive() || m.state != ViewPRInspect {
		t.Errorf("expected the form closed and the new pull request inspected, got state %v", m.state)
	}
	if msg.pr.PATID != testPATID {
		t.Error("expected the created pull request tagged with its PAT")
	}
}

func TestPRCommand_RejectsAnArgumentThatIsNotABranch(t *testing.T) {
	m := newPRTestModel(t, &mockProvider{})

	m, _ = handlePRCommand(m, []string{"just-a-branch"})

	if m.newPRView.IsActive() || m.statusBar.Message() != newPRUsage {
		t.Errorf("expected the usage hint, got %q", m.statusBar.Message())
	}
}
