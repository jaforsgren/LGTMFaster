package ui

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

const (
	prCommandName = "pr"
	// recentBranchesTTL keeps one fetch of pushed branches for the whole time a :pr is typed.
	recentBranchesTTL         = time.Minute
	maxBranchSuggestionsShown = 5
	branchSuggestionMarker    = "▸ "
	newPRUsage                = "Usage: :pr <repository>:<branch>"
)

type RecentBranchesLoadedMsg struct {
	branches       []domain.PushedBranch
	failedPATCount int
}

type NewPRBranchesLoadedMsg struct {
	branch   domain.PushedBranch
	branches domain.RepositoryBranches
	err      error
}

type PullRequestCreatedMsg struct {
	pr     domain.PullRequest
	branch domain.PushedBranch
}

// recentBranches is what :pr suggests: the user's recently pushed branches across every selected
// PAT, fetched at most once per recentBranchesTTL.
type recentBranches struct {
	branches  []domain.PushedBranch
	fetchedAt time.Time
	loading   bool
	// query is the text the suggestions were filtered by; tab cycles through its matches without
	// changing it.
	query string
}

// loadRecentBranches asks every selected PAT's provider for the branches pushed within the
// configured window, newest first.
func (m Model) loadRecentBranches() tea.Cmd {
	since := time.Now().Add(-m.currentSettings().RecentBranchWindow())
	providers := m.providers
	ctx := m.ctx
	return func() tea.Msg {
		var mu sync.Mutex
		var wg sync.WaitGroup
		msg := RecentBranchesLoadedMsg{}
		for patID, provider := range providers {
			wg.Add(1)
			go func(patID string, provider domain.Provider) {
				defer wg.Done()
				branches, err := provider.ListRecentlyPushedBranches(ctx, since)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					logger.LogError("LOAD_RECENT_BRANCHES", patID, err)
					msg.failedPATCount++
				}
				for _, branch := range branches {
					branch.PATID = patID
					msg.branches = append(msg.branches, branch)
				}
			}(patID, provider)
		}
		wg.Wait()
		sort.SliceStable(msg.branches, func(i, j int) bool {
			return msg.branches[i].PushedAt.After(msg.branches[j].PushedAt)
		})
		return msg
	}
}

func (m Model) ensureRecentBranches() (Model, tea.Cmd) {
	if m.recentBranches.loading || time.Since(m.recentBranches.fetchedAt) < recentBranchesTTL {
		return m, nil
	}
	m.recentBranches.loading = true
	return m, m.loadRecentBranches()
}

func (m Model) handleRecentBranchesLoaded(msg RecentBranchesLoadedMsg) (Model, tea.Cmd) {
	m.recentBranches.branches = msg.branches
	m.recentBranches.fetchedAt = time.Now()
	m.recentBranches.loading = false
	if msg.failedPATCount > 0 {
		m.statusBar.SetMessage(fmt.Sprintf("Could not list pushed branches for %d PAT(s)", msg.failedPATCount), true)
	}
	return m.updateCommandCompletion(), nil
}

// branchSuggestions are the recent branches matching the query that have no open pull request yet.
func (m Model) branchSuggestions() []domain.PushedBranch {
	query := strings.ToLower(m.recentBranches.query)
	var suggestions []domain.PushedBranch
	for _, branch := range m.recentBranches.branches {
		if m.hasOpenPullRequest(branch) || !strings.Contains(strings.ToLower(branch.Key()), query) {
			continue
		}
		suggestions = append(suggestions, branch)
	}
	return suggestions
}

func (m Model) hasOpenPullRequest(branch domain.PushedBranch) bool {
	if m.prCache == nil {
		return false
	}
	for _, pr := range m.prCache.AllPRs {
		if pr.Repository.FullName == branch.Repository.FullName && pr.SourceBranch == branch.Branch {
			return true
		}
	}
	return false
}

// prCommandArgument returns the typed argument when the command bar holds ":pr <argument>".
func (m Model) prCommandArgument() (string, bool) {
	name, argument, hasArgument := strings.Cut(m.commandBar.Input(), " ")
	if !hasArgument {
		return "", false
	}
	command, ok := m.commandRegistry.Resolve(name)
	if !ok || command.Name != prCommandName {
		return "", false
	}
	return strings.TrimSpace(argument), true
}

// updateCommandCompletion completes the command name, or for ":pr " the branch to open a pull
// request from.
func (m Model) updateCommandCompletion() Model {
	argument, isPRArgument := m.prCommandArgument()
	if !isPRArgument {
		m.commandBar.SetCompletion(m.commandRegistry.GetAutocompleteSuggestion(m.commandBar.Input(), m.state))
		return m
	}

	completion := ""
	if suggestions := m.branchSuggestions(); len(suggestions) > 0 && argument == m.recentBranches.query {
		completion = prCommandName + " " + suggestions[0].Key()
	}
	m.commandBar.SetCompletion(completion)
	return m
}

// handleCommandInput passes a key to the command bar and refreshes its completion, fetching the
// recent branches as soon as a :pr argument is started.
func (m Model) handleCommandInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	cmd := m.commandBar.Update(msg)
	if argument, isPRArgument := m.prCommandArgument(); isPRArgument {
		m.recentBranches.query = argument
		var load tea.Cmd
		m, load = m.ensureRecentBranches()
		cmd = tea.Batch(cmd, load)
	}
	return m.updateCommandCompletion(), cmd
}

// completeCommand accepts the command name completion, or cycles through the :pr branch
// suggestions on each press.
func (m Model) completeCommand() Model {
	argument, isPRArgument := m.prCommandArgument()
	if !isPRArgument {
		m.commandBar.AcceptCompletion()
		if _, isCommand := m.commandRegistry.Resolve(m.commandBar.Input()); isCommand {
			m.commandBar.SetInput(m.commandBar.Input() + " ")
		}
		return m
	}

	suggestions := m.branchSuggestions()
	if len(suggestions) == 0 {
		return m
	}
	next := 0
	for i, branch := range suggestions {
		if branch.Key() == argument {
			next = (i + 1) % len(suggestions)
		}
	}
	m.commandBar.SetInput(prCommandName + " " + suggestions[next].Key())
	m.commandBar.SetCompletion("")
	return m
}

// branchSuggestionHelp lists the :pr suggestions in the status bar, marking the one in the input.
func (m Model) branchSuggestionHelp(argument string) string {
	window := m.currentSettings().RecentBranchWindow()
	if m.recentBranches.loading {
		return fmt.Sprintf("loading branches pushed in the last %s...", window)
	}
	suggestions := m.branchSuggestions()
	if len(suggestions) == 0 {
		return fmt.Sprintf("no branches without a pull request pushed in the last %s", window)
	}

	hints := make([]string, 0, min(len(suggestions), maxBranchSuggestionsShown))
	for _, branch := range suggestions[:min(len(suggestions), maxBranchSuggestionsShown)] {
		hint := fmt.Sprintf("%s (%s)", branch.Key(), views.FormatAge(branch.PushedAt))
		if branch.Key() == argument {
			hint = branchSuggestionMarker + hint
		}
		hints = append(hints, hint)
	}
	help := "tab: next branch | enter: new pull request | " + strings.Join(hints, helpSeparator)
	if hidden := len(suggestions) - len(hints); hidden > 0 {
		help += fmt.Sprintf(" | +%d more", hidden)
	}
	return help
}

// startNewPR opens the form for a "<repository>:<branch>" argument, from the recent branches when
// it is one of them, else from the primary PAT.
func (m Model) startNewPR(argument string) (Model, tea.Cmd) {
	branch, found := m.recentBranchByKey(argument)
	if !found {
		repository, name, ok := strings.Cut(argument, domain.PushedBranchSeparator)
		if !ok || repository == "" || name == "" {
			m.statusBar.SetMessage(newPRUsage, true)
			return m, nil
		}
		branch = domain.PushedBranch{Repository: domain.Repo{FullName: repository}, Branch: name, PATID: m.primaryPATID}
	}

	provider := m.getProviderForPAT(branch.PATID)
	if provider == nil {
		m.statusBar.SetMessage("No active PAT. Please select a PAT first.", true)
		return m, nil
	}

	m.newPRView.Activate(branch)
	ctx := m.ctx
	return m, func() tea.Msg {
		branches, err := provider.ListBranches(ctx, branch.Repository.FullName)
		return NewPRBranchesLoadedMsg{branch: branch, branches: branches, err: err}
	}
}

func (m Model) recentBranchByKey(key string) (domain.PushedBranch, bool) {
	for _, branch := range m.recentBranches.branches {
		if branch.Key() == key {
			return branch, true
		}
	}
	return domain.PushedBranch{}, false
}

func (m Model) handleNewPRBranchesLoaded(msg NewPRBranchesLoadedMsg) (Model, tea.Cmd) {
	if !m.newPRView.IsActive() || m.newPRView.Branch().Key() != msg.branch.Key() {
		return m, nil
	}
	if msg.err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to list branches of %s: %v", msg.branch.Repository.FullName, msg.err), true)
		return m, nil
	}
	m.newPRView.SetBranches(msg.branches)
	return m, nil
}

// handleNewPRKey routes keys to the open new pull request form.
func (m Model) handleNewPRKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	action, cmd := m.newPRView.HandleKey(msg)
	switch action {
	case views.NewPRActionSubmit:
		return m.submitNewPR()
	case views.NewPRActionCancel:
		m.newPRView.Deactivate()
		return m, nil
	case views.NewPRActionOpenEditor:
		return m, m.openExternalEditor(m.newPRView.Description(), EditorSourceNewPR)
	}
	return m, cmd
}

func (m Model) submitNewPR() (Model, tea.Cmd) {
	request, err := m.newPRView.Request()
	if err != nil {
		m.statusBar.SetMessage(err.Error(), true)
		return m, nil
	}

	branch := m.newPRView.Branch()
	provider := m.getProviderForPAT(branch.PATID)
	ctx := m.ctx
	m.statusBar.SetMessage(fmt.Sprintf("Creating pull request from %s...", branch.Key()), false)
	return m, func() tea.Msg {
		pr, err := provider.CreatePullRequest(ctx, request)
		if err != nil {
			return ErrorMsg{err: fmt.Errorf("failed to create pull request: %w", err)}
		}
		pr.PATID = branch.PATID
		pr.ProviderType = provider.GetType()
		return PullRequestCreatedMsg{pr: *pr, branch: branch}
	}
}

// handlePullRequestCreated opens the new pull request; the PR list is reloaded the next time it is
// shown, since the cached one does not have it yet.
func (m Model) handlePullRequestCreated(msg PullRequestCreatedMsg) (Model, tea.Cmd) {
	m.newPRView.Deactivate()
	m.prCache = nil
	m.recentBranches.fetchedAt = time.Time{}
	m.statusBar.SetMessage(fmt.Sprintf("Created pull request #%d from %s", msg.pr.Number, msg.branch.Key()), false)
	inspected, cmd := m.inspectPR(msg.pr)
	return inspected, tea.Batch(cmd, clearStatusAfterDelay(4*time.Second))
}
