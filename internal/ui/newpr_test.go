package ui

import (
	"errors"
	"slices"
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

func newPRFormTestModel(t *testing.T, provider *mockProvider) Model {
	t.Helper()
	provider.branches = domain.RepositoryBranches{Default: "main", Names: []string{"main", "feature/login"}}
	if provider.createdPR == nil {
		provider.createdPR = &domain.PullRequest{Number: 7, Title: "Login", Repository: domain.Repo{FullName: "Proj/api"}}
	}
	m := newPRTestModel(t, provider)
	m.recentBranches = recentBranches{branches: recentTestBranches(), fetchedAt: time.Now()}
	m, cmd := handlePRCommand(m, []string{"Proj/api:feature/login"})
	m, _ = m.handleNewPRBranchesLoaded(cmd().(NewPRBranchesLoadedMsg))
	return m
}

// typeWorkItems moves from the title, where the form opens, to the work items field and types.
func typeWorkItems(t *testing.T, m Model, text string) Model {
	t.Helper()
	for range 2 {
		m, _ = m.handleNewPRKey(tea.KeyMsg{Type: tea.KeyTab})
	}
	m, _ = m.handleNewPRKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
	return m
}

func TestNewPR_LinksTheTypedWorkItemsOnceCreated(t *testing.T) {
	provider := &mockProvider{}
	m := typeWorkItems(t, newPRFormTestModel(t, provider), "1234, #5678")

	m, cmd := m.handleNewPRKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	msg := cmd().(PullRequestCreatedMsg)
	m, _ = m.handlePullRequestCreated(msg)

	if !slices.Equal(provider.linkWorkItemCalls, []int{1234, 5678}) {
		t.Errorf("expected 1234 and 5678 linked, got %v", provider.linkWorkItemCalls)
	}
	if msg := m.statusBar.Message(); msg != "Created pull request #7 from Proj/api:feature/login" {
		t.Errorf("expected only the creation reported, got %q", msg)
	}
}

func TestNewPR_ReportsTheWorkItemsThatCouldNotBeLinked(t *testing.T) {
	provider := &mockProvider{linkWorkItemErrs: map[int]error{1234: errors.New("does not exist")}}
	m := typeWorkItems(t, newPRFormTestModel(t, provider), "1234 5678")

	m, cmd := m.handleNewPRKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	m, _ = m.handlePullRequestCreated(cmd().(PullRequestCreatedMsg))

	if !slices.Equal(provider.linkWorkItemCalls, []int{1234, 5678}) {
		t.Errorf("expected every work item tried, got %v", provider.linkWorkItemCalls)
	}
	if msg := m.statusBar.Message(); !strings.Contains(msg, "Created pull request #7") || !strings.Contains(msg, "could not link #1234: does not exist") {
		t.Errorf("expected the PR created and the failed link reported, got %q", msg)
	}
	if m.state != ViewPRInspect {
		t.Errorf("expected the new pull request opened, got state %v", m.state)
	}
}

func TestNewPR_RejectsWorkItemsThatAreNotNumbers(t *testing.T) {
	provider := &mockProvider{}
	m := typeWorkItems(t, newPRFormTestModel(t, provider), "login")

	m, cmd := m.handleNewPRKey(tea.KeyMsg{Type: tea.KeyCtrlS})

	if cmd != nil || provider.lastNewPR != nil {
		t.Error("expected nothing created")
	}
	if !m.newPRView.IsActive() || !strings.Contains(m.statusBar.Message(), "not a work item number") {
		t.Errorf("expected the form kept open with the error, got %q", m.statusBar.Message())
	}
}

func TestPRCommand_RejectsAnArgumentThatIsNotABranch(t *testing.T) {
	m := newPRTestModel(t, &mockProvider{})

	m, _ = handlePRCommand(m, []string{"just-a-branch"})

	if m.newPRView.IsActive() || m.statusBar.Message() != newPRUsage {
		t.Errorf("expected the usage hint, got %q", m.statusBar.Message())
	}
}

func TestNewPRKeys_OpenTheCommandBarOnPRAndFetchRecentBranches(t *testing.T) {
	for _, view := range []ViewState{ViewPRList, ViewDashboard} {
		m := newPRTestModel(t, &mockProvider{pushedBranches: recentTestBranches()})
		m.state = view

		m, _ = pressKey(t, m, keyNewPRPrefix)
		m, cmd := pressKey(t, m, keyNewPR)

		if !m.commandBar.IsActive() {
			t.Fatalf("view %v: expected pr to open the command bar", view)
		}
		if got := m.commandBar.Input(); got != prCommandName+" " {
			t.Errorf("view %v: expected %q, got %q", view, prCommandName+" ", got)
		}
		if !m.recentBranches.loading || cmd == nil {
			t.Errorf("view %v: expected the recent branches fetched", view)
		}
		if m.pendingPrefix != "" {
			t.Errorf("view %v: expected the prefix finished", view)
		}
	}
}

func TestNewPRPrefix_AnotherKeyDropsItAndRunsAsUsual(t *testing.T) {
	m, repository := sortTestModel(t)

	m, _ = pressKey(t, m, keyNewPRPrefix)
	m, _ = pressKey(t, m, "2")

	if m.pendingPrefix != "" {
		t.Error("expected the prefix dropped")
	}
	if m.commandBar.IsActive() {
		t.Error("expected the command bar closed")
	}
	if repository.settings.PRSortMode != domain.PRSortTitle {
		t.Errorf("expected the key to sort as usual, got %q", repository.settings.PRSortMode)
	}
}

func TestNewPRShortcut_ShownInTheTopBar(t *testing.T) {
	registry := NewCommandRegistry()
	for _, view := range []ViewState{ViewPRList, ViewDashboard} {
		shortcuts := strings.Join(registry.GetContextualShortcuts(view, false), "\n")
		if !strings.Contains(shortcuts, "<"+keyNewPRPrefix+keyNewPR+"> New pull request") {
			t.Errorf("view %v: expected the pr shortcut listed, got %q", view, shortcuts)
		}
	}
}

func startupTestDraft() domain.NewPullRequestDraft {
	return domain.NewPullRequestDraft{
		Remote:       domain.GitRemote{Provider: domain.ProviderGitHub, RepositoryFullName: "Proj/api"},
		SourceBranch: "feature/login",
		TargetBranch: "develop",
		Description:  "From the command line",
		WorkItemIDs:  []int{1234, 5678},
	}
}

func TestWithNewPullRequest_OpensThePrefilledFormOnceThePATsLoad(t *testing.T) {
	repository := &mockRepository{pats: map[string]*domain.PAT{}}
	m := NewModel(repository).WithNewPullRequest(startupTestDraft(), testPATID)

	updated, _ := m.Update(PATsLoadedMsg{pats: []domain.PAT{{ID: testPATID, Name: "gh", Provider: domain.ProviderGitHub, IsSelected: true, IsPrimary: true}}})
	m = updated.(Model)

	if !m.newPRView.IsActive() || m.newPRView.Branch().Key() != "Proj/api:feature/login" || m.newPRView.Branch().PATID != testPATID {
		t.Fatalf("expected the form for Proj/api:feature/login, got %+v", m.newPRView.Branch())
	}
	if m.startupPR != nil {
		t.Error("expected the startup pull request consumed")
	}
}

func TestWithNewPullRequest_PrefillsTargetDescriptionAndWorkItems(t *testing.T) {
	provider := &mockProvider{
		branches:  domain.RepositoryBranches{Default: "main", Names: []string{"main", "develop", "feature/login"}},
		createdPR: &domain.PullRequest{Number: 7, Title: "Login", Repository: domain.Repo{FullName: "Proj/api"}},
	}
	m := newPRTestModel(t, provider).WithNewPullRequest(startupTestDraft(), testPATID)

	m, cmd := m.openStartupPR()
	m, _ = m.handleNewPRBranchesLoaded(cmd().(NewPRBranchesLoadedMsg))
	m, cmd = m.handleNewPRKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	cmd()

	want := domain.NewPullRequest{
		RepositoryFullName: "Proj/api", SourceBranch: "feature/login", TargetBranch: "develop",
		Title: "Login", Description: "From the command line",
	}
	if provider.lastNewPR == nil || *provider.lastNewPR != want {
		t.Errorf("expected %+v, got %+v", want, provider.lastNewPR)
	}
	if !slices.Equal(provider.linkWorkItemCalls, []int{1234, 5678}) {
		t.Errorf("expected the work items linked, got %v", provider.linkWorkItemCalls)
	}
}
