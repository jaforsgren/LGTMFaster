package ui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/git"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/ui/components"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

type CommandHandler func(Model, []string) (Model, tea.Cmd)

type KeyHandler func(Model) (Model, tea.Cmd)

type Command struct {
	Name        string
	Aliases     []string
	Description string
	ShortHelp   string
	Handler     CommandHandler
	AvailableIn []ViewState
}

type KeyBinding struct {
	Keys               []string
	Description        string
	ShortHelp          string
	Handler            KeyHandler
	AvailableIn        []ViewState
	RequiresLinkedRepo bool
	// Navigation bindings are listed in the status bar help line instead of the top bar shortcuts.
	Navigation bool
}

const helpSeparator = " | "

var allViewStates = []ViewState{ViewPATs, ViewPRList, ViewPRInspect, ViewPipelineList, ViewPipelineRuns, ViewPipelineRun, ViewDashboard}

var pipelineViewStates = []ViewState{ViewPipelineList, ViewPipelineRuns, ViewPipelineRun}

// splitViewStates are the lists with a details pane beside them.
var splitViewStates = []ViewState{ViewPRList, ViewPipelineList, ViewPipelineRuns}

type CommandRegistry struct {
	commands    map[string]*Command
	keyBindings []*KeyBinding
}

func NewCommandRegistry() *CommandRegistry {
	registry := &CommandRegistry{
		commands:    make(map[string]*Command),
		keyBindings: []*KeyBinding{},
	}
	registry.registerCommands()
	registry.registerKeyBindings()
	registry.registerPipelineSortBindings()
	return registry
}

func (cr *CommandRegistry) registerCommands() {
	commands := []*Command{
		{
			Name:        "pats",
			Aliases:     []string{"p", "pat"},
			Description: "Switch to PAT management view",
			ShortHelp:   ":p",
			Handler:     handlePATsCommand,
			AvailableIn: allViewStates,
		},
		{
			Name:        "pr",
			Aliases:     []string{"prs", "pulls"},
			Description: "List pull requests, or :pr <repository>:<branch> to open one",
			ShortHelp:   ":pr",
			Handler:     handlePRCommand,
			AvailableIn: allViewStates,
		},
		{
			Name:        "pipelines",
			Aliases:     []string{"pl", "pipes"},
			Description: "List pipelines (Azure DevOps)",
			ShortHelp:   ":pl",
			Handler:     handlePipelinesCommand,
			AvailableIn: allViewStates,
		},
		{
			Name:        "dashboard",
			Aliases:     []string{"dash", "d"},
			Description: "PRs waiting for your review, failing checks, ready to merge",
			ShortHelp:   ":d",
			Handler:     handleDashboardCommand,
			AvailableIn: allViewStates,
		},
		{
			Name:        "session",
			Aliases:     []string{"resume"},
			Description: "Resume where you were when you last quit",
			ShortHelp:   ":session",
			Handler:     handleSessionCommand,
			AvailableIn: allViewStates,
		},
		{
			Name:        "logs",
			Aliases:     []string{"log"},
			Description: "View session logs",
			ShortHelp:   ":logs",
			Handler:     handleLogsCommand,
			AvailableIn: allViewStates,
		},
		{
			Name:        "merge",
			Aliases:     []string{"m"},
			Description: "Merge pull request",
			ShortHelp:   ":merge",
			Handler:     handleMergeCommand,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Name:        "quit",
			Aliases:     []string{"q", "exit"},
			Description: "Quit application",
			ShortHelp:   ":q",
			Handler:     handleQuitCommand,
			AvailableIn: allViewStates,
		},
		{
			Name:        "link",
			Aliases:     []string{"l"},
			Description: "Link PR to local repository",
			ShortHelp:   ":link",
			Handler:     handleLinkCommand,
			AvailableIn: []ViewState{ViewPRInspect, ViewPRList},
		},
		{
			Name:        "repos",
			Aliases:     []string{},
			Description: "Manage linked repositories",
			ShortHelp:   ":repos",
			Handler:     handleReposCommand,
			AvailableIn: allViewStates,
		},
		{
			Name:        "settings",
			Aliases:     []string{"s"},
			Description: "Edit settings file",
			ShortHelp:   ":s",
			Handler:     handleSettingsCommand,
			AvailableIn: allViewStates,
		},
		{
			Name:        "help",
			Aliases:     []string{"h", "?"},
			Description: "Show help",
			ShortHelp:   ":h",
			Handler:     nil,
			AvailableIn: allViewStates,
		},
	}

	for _, cmd := range commands {
		cr.commands[cmd.Name] = cmd
		for _, alias := range cmd.Aliases {
			cr.commands[alias] = cmd
		}
	}
}

func (cr *CommandRegistry) registerKeyBindings() {
	cr.keyBindings = []*KeyBinding{
		{
			Keys:        []string{keyQuit},
			Description: "back (quit at the top)",
			ShortHelp:   keyQuit,
			Handler:     handleQuitKey,
			AvailableIn: allViewStates,
			Navigation:  true,
		},
		{
			Keys:        []string{keyEnter},
			Description: "select",
			ShortHelp:   keyEnter,
			Handler:     handleEnterKey,
			AvailableIn: []ViewState{ViewPATs, ViewPRList, ViewPRInspect, ViewPipelineList, ViewPipelineRuns, ViewDashboard},
			Navigation:  true,
		},
		{
			Keys:        []string{keySpace},
			Description: "Toggle selection (multi-select)",
			ShortHelp:   "space",
			Handler:     handleSpaceKey,
			AvailableIn: []ViewState{ViewPATs},
		},
		{
			Keys:        []string{keyBackspace, keyBack},
			Description: "back",
			ShortHelp:   keyBack,
			Handler:     handleBackKey,
			AvailableIn: []ViewState{ViewPRList, ViewPRInspect, ViewPipelineList, ViewPipelineRuns, ViewPipelineRun, ViewDashboard},
			Navigation:  true,
		},
		{
			Keys:        []string{keyUp, keyUpVim},
			Description: "navigate",
			ShortHelp:   keyDownVim + "/" + keyUpVim,
			Handler:     handleUpKey,
			AvailableIn: allViewStates,
			Navigation:  true,
		},
		{
			Keys:        []string{keyDown, keyDownVim},
			Description: "navigate",
			ShortHelp:   keyDownVim + "/" + keyUpVim,
			Handler:     handleDownKey,
			AvailableIn: allViewStates,
			Navigation:  true,
		},
		{
			Keys:        []string{keyFocusPaneIn},
			Description: "focus pane",
			ShortHelp:   "→/←",
			Handler:     handleFocusPaneInKey,
			AvailableIn: splitViewStates,
			Navigation:  true,
		},
		{
			Keys:        []string{keyFocusPaneOut},
			Description: "focus pane",
			ShortHelp:   "→/←",
			Handler:     handleFocusPaneOutKey,
			AvailableIn: splitViewStates,
			Navigation:  true,
		},
		{
			Keys:        []string{keyFilter},
			Description: "filter",
			ShortHelp:   keyFilter,
			Handler:     handleFilterKey,
			AvailableIn: []ViewState{ViewPRList, ViewPipelineList, ViewPipelineRuns},
			Navigation:  true,
		},
		{
			Keys:        []string{keyEsc},
			Description: "cancel/back",
			ShortHelp:   keyEsc,
			Handler:     handleEscKey,
			AvailableIn: allViewStates,
			Navigation:  true,
		},
		{
			Keys:        []string{keyCommand},
			Description: "command",
			ShortHelp:   keyCommand,
			Handler:     handleColonKey,
			AvailableIn: allViewStates,
			Navigation:  true,
		},
		{
			Keys:        []string{keyAdd},
			Description: "Add PAT",
			ShortHelp:   keyAdd,
			Handler:     handleAddKey,
			AvailableIn: []ViewState{ViewPATs},
		},
		{
			Keys:        []string{keyDelete},
			Description: "Delete PAT",
			ShortHelp:   keyDelete,
			Handler:     handleDeleteKey,
			AvailableIn: []ViewState{ViewPATs},
		},
		{
			Keys:        []string{keyEdit},
			Description: "Edit PAT",
			ShortHelp:   keyEdit,
			Handler:     handleEditKey,
			AvailableIn: []ViewState{ViewPATs},
		},
		{
			Keys:        []string{keyTogglePane},
			Description: "next pane",
			ShortHelp:   keyTogglePane,
			Handler:     handleDashboardNextPaneKey,
			AvailableIn: []ViewState{ViewDashboard},
			Navigation:  true,
		},
		{
			Keys:        []string{keyPrevPane},
			Description: "previous pane",
			ShortHelp:   keyPrevPane,
			Handler:     handleDashboardPrevPaneKey,
			AvailableIn: []ViewState{ViewDashboard},
			Navigation:  true,
		},
		{
			Keys:        []string{keyFocusPaneIn},
			Description: "pane beside",
			ShortHelp:   "→/←",
			Handler:     handleDashboardRightKey,
			AvailableIn: []ViewState{ViewDashboard},
			Navigation:  true,
		},
		{
			Keys:        []string{keyFocusPaneOut},
			Description: "pane beside",
			ShortHelp:   "→/←",
			Handler:     handleDashboardLeftKey,
			AvailableIn: []ViewState{ViewDashboard},
			Navigation:  true,
		},
		{
			Keys:        []string{keyDashboard},
			Description: "Dashboard",
			ShortHelp:   keyDashboard,
			Handler:     handleDashboardKey,
			AvailableIn: []ViewState{ViewPRList},
		},
		{
			Keys:        []string{keyTogglePane},
			Description: "Toggle details pane",
			ShortHelp:   keyTogglePane,
			Handler:     handleTogglePreviewKey,
			AvailableIn: splitViewStates,
		},
		{
			Keys:        []string{keyToggleTopBar},
			Description: "Toggle top bar",
			ShortHelp:   keyToggleTopBar,
			Handler:     handleToggleTopBarKey,
			AvailableIn: allViewStates,
			Navigation:  true,
		},
		{
			Keys:        []string{keyTogglePane},
			Description: "Work items pane",
			ShortHelp:   keyTogglePane,
			Handler:     handleTogglePreviewKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyLinkWorkItem},
			Description: "Link work item",
			ShortHelp:   keyLinkWorkItem,
			Handler:     handleLinkWorkItemKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyRefresh},
			Description: "Refresh",
			ShortHelp:   keyRefresh,
			Handler:     handleRefreshKey,
			AvailableIn: []ViewState{ViewPRList, ViewPipelineList, ViewPipelineRuns, ViewPipelineRun, ViewDashboard},
		},
		{
			Keys:        []string{keyOpenBrowser},
			Description: "Open in browser",
			ShortHelp:   keyOpenBrowser,
			Handler:     handleOpenBrowserKey,
			AvailableIn: []ViewState{ViewPRList, ViewPRInspect, ViewPipelineList, ViewPipelineRuns, ViewPipelineRun, ViewDashboard},
		},
		{
			Keys:        []string{keyNextFile, keyFocusPaneIn},
			Description: "Next/previous file",
			ShortHelp:   keyNextFile + "/" + keyPrevFile,
			Handler:     handleNextFileKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyPrevFile, keyFocusPaneOut},
			Description: "Next/previous file",
			ShortHelp:   keyNextFile + "/" + keyPrevFile,
			Handler:     handlePrevFileKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyViewDiff},
			Description: "View diff",
			ShortHelp:   keyViewDiff,
			Handler:     handleViewDiffKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyComments},
			Description: "View comments",
			ShortHelp:   keyComments,
			Handler:     handleViewCommentsKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyApprove},
			Description: "Approve PR",
			ShortHelp:   keyApprove,
			Handler:     handleApproveKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyRequestChange},
			Description: "Request changes",
			ShortHelp:   keyRequestChange,
			Handler:     handleRequestChangesKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyMerge},
			Description: "Merge PR",
			ShortHelp:   keyMerge,
			Handler:     handleMergeKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyInlineComment},
			Description: "Inline comment on line",
			ShortHelp:   keyInlineComment,
			Handler:     handleInlineCommentKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyToggleDiff},
			Description: "Toggle diff view mode",
			ShortHelp:   keyToggleDiff,
			Handler:     handleToggleDiffViewKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyCommits},
			Description: "Commits",
			ShortHelp:   keyCommits,
			Handler:     handleCommitsKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyWhitespace},
			Description: "Hide/show whitespace changes",
			ShortHelp:   keyWhitespace,
			Handler:     handleWhitespaceKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyFileFilter},
			Description: "Filter files by extension",
			ShortHelp:   keyFileFilter,
			Handler:     handleFileFilterKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyYankFile},
			Description: "Yank current file diff",
			ShortHelp:   keyYankFile,
			Handler:     handleYankCurrentFileKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyYankAll},
			Description: "Yank all files diff",
			ShortHelp:   keyYankAll,
			Handler:     handleYankAllFilesKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyEdit},
			Description: "Edit PR description",
			ShortHelp:   keyEdit,
			Handler:     handleEditDescriptionKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keySubmitReview},
			Description: "Submit review",
			ShortHelp:   keySubmitReview,
			Handler:     handleReviewSubmitKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{keyLinkRepo},
			Description: "Link to local repo",
			ShortHelp:   keyLinkRepo,
			Handler:     handleLinkKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:               []string{keyCheckout},
			Description:        "Checkout branch",
			ShortHelp:          keyCheckout,
			Handler:            handleCheckoutKey,
			AvailableIn:        []ViewState{ViewPRInspect},
			RequiresLinkedRepo: true,
		},
		{
			Keys:               []string{keyOpenFolder},
			Description:        "Open local repo folder",
			ShortHelp:          keyOpenFolder,
			Handler:            handleOpenFolderKey,
			AvailableIn:        []ViewState{ViewPRInspect},
			RequiresLinkedRepo: true,
		},
		{
			Keys:               []string{keyPull},
			Description:        "Pull latest changes",
			ShortHelp:          keyPull,
			Handler:            handlePullKey,
			AvailableIn:        []ViewState{ViewPRInspect},
			RequiresLinkedRepo: true,
		},
		{
			Keys:        []string{keyApprove},
			Description: "Approve stage",
			ShortHelp:   keyApprove,
			Handler:     handleApproveStageKey,
			AvailableIn: []ViewState{ViewPipelineRun},
		},
		{
			Keys:        []string{keyReject},
			Description: "Reject stage",
			ShortHelp:   keyReject,
			Handler:     handleRejectStageKey,
			AvailableIn: []ViewState{ViewPipelineRun},
		},
		{
			Keys:        []string{keyFold},
			Description: "Fold/unfold",
			ShortHelp:   keyFold,
			Handler:     handleFoldKey,
			AvailableIn: []ViewState{ViewPipelineList},
		},
		{
			Keys:        []string{keyFavoritesOnly},
			Description: "Favorites only",
			ShortHelp:   keyFavoritesOnly,
			Handler:     handleFavoritesOnlyKey,
			AvailableIn: []ViewState{ViewPipelineList},
		},
		{
			Keys:        []string{keyAutoRefresh},
			Description: "Toggle auto-refresh",
			ShortHelp:   keyAutoRefresh,
			Handler:     handleAutoRefreshKey,
			AvailableIn: pipelineViewStates,
		},
		{
			Keys:        []string{keyFoldAll},
			Description: "Fold/unfold all",
			ShortHelp:   keyFoldAll,
			Handler:     handleFoldAllKey,
			AvailableIn: []ViewState{ViewPipelineList},
		},
		{
			Keys:        []string{keyFavorite},
			Description: "Toggle favorite",
			ShortHelp:   keyFavorite,
			Handler:     handleFavoriteKey,
			AvailableIn: pipelineViewStates,
		},
	}
}

// registerPipelineSortBindings binds the number keys to sort modes. Only the first binding carries
// a ShortHelp so the top bar shows a single "<0-5> Sort" hint instead of one per mode.
func (cr *CommandRegistry) registerPipelineSortBindings() {
	lastIndex := len(domain.PipelineSortModes) - 1
	for i, mode := range domain.PipelineSortModes {
		binding := &KeyBinding{
			Keys:        []string{strconv.Itoa(i)},
			Description: "Sort " + views.PipelineSortModeLabel(mode),
			Handler:     sortPipelinesHandler(mode),
			AvailableIn: []ViewState{ViewPipelineList},
		}
		if i == 0 {
			binding.Description = "Sort (fav/proj/repo/name/run/status)"
			binding.ShortHelp = fmt.Sprintf("0-%d", lastIndex)
		}
		cr.keyBindings = append(cr.keyBindings, binding)
	}
}

func sortPipelinesHandler(mode domain.PipelineSortMode) KeyHandler {
	return func(m Model) (Model, tea.Cmd) {
		return m.setPipelineSortMode(mode)
	}
}

func (cr *CommandRegistry) ExecuteCommand(m Model, cmdName string, args []string) (Model, tea.Cmd) {
	cmdName = strings.TrimSpace(cmdName)
	if cmdName == "" {
		return m, nil
	}

	if cmdName == "help" || cmdName == "h" || cmdName == "?" {
		m.statusBar.SetMessage(cr.GenerateHelpText(), false)
		return m, nil
	}

	cmd, exists := cr.commands[cmdName]
	if !exists {
		m.statusBar.SetMessage(fmt.Sprintf("Unknown command: %s (try :help)", cmdName), true)
		return m, nil
	}

	if cmd.Handler == nil {
		m.statusBar.SetMessage(fmt.Sprintf("Command not implemented: %s", cmdName), false)
		return m, nil
	}

	return cmd.Handler(m, args)
}

// Resolve finds a command by its name or one of its aliases.
func (cr *CommandRegistry) Resolve(name string) (*Command, bool) {
	command, ok := cr.commands[name]
	return command, ok
}

func (cr *CommandRegistry) HandleKey(m Model, key string) (Model, tea.Cmd, bool) {
	for _, kb := range cr.keyBindings {
		if !isInViews(m.state, kb.AvailableIn) {
			continue
		}
		for _, k := range kb.Keys {
			if k == key {
				newModel, cmd := kb.Handler(m)
				return newModel, cmd, true
			}
		}
	}
	return m, nil, false
}

func (cr *CommandRegistry) GenerateHelpText() string {
	var parts []string
	seen := make(map[string]bool)

	for name, cmd := range cr.commands {
		if name != cmd.Name {
			continue
		}
		if seen[cmd.Name] {
			continue
		}
		seen[cmd.Name] = true

		nameWithAliases := ":" + cmd.Name
		if len(cmd.Aliases) > 0 {
			nameWithAliases += "/:" + strings.Join(cmd.Aliases, "/:")
		}
		if cmd.Description != "" {
			nameWithAliases += " (" + cmd.Description + ")"
		}
		parts = append(parts, nameWithAliases)
	}

	slices.Sort(parts)
	return "Commands: " + strings.Join(parts, " | ")
}

// GetContextualShortcuts lists the view's actions for the top bar.
func (cr *CommandRegistry) GetContextualShortcuts(state ViewState, hasLinkedRepo bool) []string {
	var shortcuts []string
	for _, kb := range cr.visibleBindings(state, hasLinkedRepo, false) {
		shortcuts = append(shortcuts, fmt.Sprintf("<%s> %s", kb.ShortHelp, kb.Description))
	}
	return shortcuts
}

// NavigationHelp lists the view's navigation keys for the status bar help line.
func (cr *CommandRegistry) NavigationHelp(state ViewState) string {
	var hints []string
	for _, kb := range cr.visibleBindings(state, false, true) {
		hints = append(hints, kb.ShortHelp+": "+kb.Description)
	}
	return strings.Join(hints, helpSeparator)
}

// visibleBindings returns one binding per hint, in registration order, that is either a
// navigation binding or an action binding as navigation asks.
func (cr *CommandRegistry) visibleBindings(state ViewState, hasLinkedRepo, navigation bool) []*KeyBinding {
	var bindings []*KeyBinding
	seen := make(map[string]bool)

	for _, kb := range cr.keyBindings {
		if kb.Navigation != navigation || !isInViews(state, kb.AvailableIn) {
			continue
		}
		if kb.RequiresLinkedRepo && !hasLinkedRepo {
			continue
		}
		if kb.Description == "" || kb.ShortHelp == "" || seen[kb.ShortHelp] {
			continue
		}
		seen[kb.ShortHelp] = true
		bindings = append(bindings, kb)
	}
	return bindings
}

func (cr *CommandRegistry) GetAutocompleteSuggestion(input string, state ViewState) string {
	input = strings.ToLower(input)
	var matches []string

	for name, cmd := range cr.commands {
		if name != cmd.Name {
			continue
		}
		if !isInViews(state, cmd.AvailableIn) {
			continue
		}

		if strings.HasPrefix(strings.ToLower(cmd.Name), input) {
			matches = append(matches, cmd.Name)
		}
	}

	if len(matches) == 1 {
		return matches[0]
	}

	return ""
}

const commentsMarkdownFilePattern = "lgtmfaster-pr-%d-comments-*.md"

func isInViews(state ViewState, states []ViewState) bool {
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}

func handlePATsCommand(m Model, args []string) (Model, tea.Cmd) {
	m.state = ViewPATs
	m.topBar.SetView(components.ViewNamePATs)
	m.topBar.SetContext("", "")
	m.topBar.SetStats(0, 0)
	m.topBar.SetPRBreakdown(0, 0, 0)
	m.updateShortcuts()
	m.statusBar.SetMessage("Showing PATs", false)
	return m, m.loadPATs()
}

// handlePRCommand lists the pull requests, or with a "<repository>:<branch>" argument opens the form
// for a new one from that branch.
func handlePRCommand(m Model, args []string) (Model, tea.Cmd) {
	if len(m.providers) == 0 && m.provider == nil {
		m.statusBar.SetMessage("No active PAT. Please select a PAT first.", true)
		return m, nil
	}
	if len(args) > 0 {
		return m.startNewPR(strings.Join(args, " "))
	}
	return m.showPRList(), m.loadPRsWithCache()
}

// inspectPR opens a pull request in the inspect view and loads its details, diff and comments.
func (m Model) inspectPR(pr domain.PullRequest) (Model, tea.Cmd) {
	if m.state != ViewPRInspect {
		m.prInspectOrigin = m.state
	}
	m.state = ViewPRInspect
	m.topBar.SetCommentThreads(nil)
	m.prInspect.ClosePreview()
	m.prInspect.SwitchToDescription()
	m.topBar.SetContext(pr.Repository.FullName, fmt.Sprintf("%d", pr.Number))
	m.topBar.SetView(components.ViewNamePRDescription)
	m.updateShortcuts()
	return m, tea.Batch(
		m.loadPRDetail(pr),
		m.loadDiff(pr),
		m.loadComments(pr),
	)
}

func handlePipelinesCommand(m Model, args []string) (Model, tea.Cmd) {
	return m.openPipelineList()
}

func handleLogsCommand(m Model, args []string) (Model, tea.Cmd) {
	m.logsView.Activate()
	return m, nil
}

func handleQuitCommand(m Model, args []string) (Model, tea.Cmd) {
	return m.quit()
}

func handleQuitKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPATs {
		return m.quit()
	}

	newModel, cmd := m.navigateBack()
	return newModel.(Model), cmd
}

func handleSpaceKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPATs {
		newModel, cmd := m.handlePATSpaceToggle()
		return newModel.(Model), cmd
	}
	return m, nil
}

func handleEnterKey(m Model) (Model, tea.Cmd) {
	switch m.state {
	case ViewPATs:
		newModel, cmd := m.handlePATEnter()
		return newModel.(Model), cmd
	case ViewPRList:
		if pr := m.prListView.GetSelectedPR(); pr != nil {
			return m.inspectPR(*pr)
		}
	case ViewDashboard:
		if pr := m.dashboardView.GetSelectedPR(); pr != nil {
			return m.inspectPR(*pr)
		}
	case ViewPRInspect:
		switch m.prInspect.GetMode() {
		case views.PRInspectModeCommits:
			return m.openSelectedCommit()
		case views.PRInspectModeDiff:
			if m.prInspect.ViewingCommit() != nil {
				m.statusBar.SetMessage(commitCommentsBlockedMessage, true)
				return m, nil
			}
			m.reviewView.Activate(views.ReviewModeComment)
		}
		return m, nil
	case ViewPipelineList:
		if run := m.pipelineListView.GetSelectedRun(); run != nil {
			return m.enterPipelineRun(*run)
		}
		if pipeline := m.pipelineListView.GetSelectedPipeline(); pipeline != nil {
			return m.enterPipelineRuns(*pipeline)
		}
		m.pipelineListView.ToggleSelected()
		return m, m.loadPipelinePreview()
	case ViewPipelineRuns:
		if run := m.pipelineRunsView.GetSelectedRun(); run != nil {
			return m.enterPipelineRun(*run)
		}
	}
	return m, nil
}

func handleBackKey(m Model) (Model, tea.Cmd) {
	if m.patsView.Mode == views.PATModeAdd || m.patsView.Mode == views.PATModeEdit {
		m.patsView.ExitEditMode()
		return m, nil
	}
	newModel, cmd := m.navigateBack()
	return newModel.(Model), cmd
}

func handleUpKey(m Model) (Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.scrollFocusedPane(-paneScrollStep) {
		return m, nil
	}
	switch m.state {
	case ViewPATs:
		cmd = m.patsView.Update(tea.KeyMsg{Type: tea.KeyUp})
	case ViewPRList:
		m.prListView.MoveUp()
		cmd = m.loadPRListComments()
	case ViewDashboard:
		m.dashboardView.MoveUp()
	case ViewPRInspect:
		if m.prInspect.GetMode() == views.PRInspectModeCommits {
			m.prInspect.MoveCommitUp()
			return m, nil
		}
		if m.prInspect.GetMode() == views.PRInspectModeDiff {
			m.prInspect.PrevLine()
			return m, nil
		}
		cmd = m.prInspect.Update(tea.KeyMsg{Type: tea.KeyUp})
	case ViewPipelineList:
		m.pipelineListView.MoveUp()
		cmd = m.loadPipelinePreview()
	case ViewPipelineRuns:
		m.pipelineRunsView.MoveUp()
		cmd = m.loadPipelinePreview()
	case ViewPipelineRun:
		m.pipelineRunView.MoveUp()
	}
	return m, cmd
}

func handleDownKey(m Model) (Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.scrollFocusedPane(paneScrollStep) {
		return m, nil
	}
	switch m.state {
	case ViewPATs:
		cmd = m.patsView.Update(tea.KeyMsg{Type: tea.KeyDown})
	case ViewPRList:
		m.prListView.MoveDown()
		cmd = m.loadPRListComments()
	case ViewDashboard:
		m.dashboardView.MoveDown()
	case ViewPRInspect:
		if m.prInspect.GetMode() == views.PRInspectModeCommits {
			m.prInspect.MoveCommitDown()
			return m, nil
		}
		if m.prInspect.GetMode() == views.PRInspectModeDiff {
			m.prInspect.NextLine()
			return m, nil
		}
		cmd = m.prInspect.Update(tea.KeyMsg{Type: tea.KeyDown})
	case ViewPipelineList:
		m.pipelineListView.MoveDown()
		cmd = m.loadPipelinePreview()
	case ViewPipelineRuns:
		m.pipelineRunsView.MoveDown()
		cmd = m.loadPipelinePreview()
	case ViewPipelineRun:
		m.pipelineRunView.MoveDown()
	}
	return m, cmd
}

func handleAddKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPATs {
		m.patsView.EnterAddMode()
		return m, nil
	}
	return m, nil
}

func handleEditKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPATs {
		pat := m.patsView.GetSelectedPAT()
		if pat != nil {
			m.patsView.EnterEditMode(*pat)
			return m, nil
		}
		m.statusBar.SetMessage("No PAT selected to edit", true)
		return m, nil
	}
	return m, nil
}

func handleDeleteKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPATs {
		newModel, cmd := m.handleDeletePAT()
		return newModel.(Model), cmd
	}
	return m, nil
}

func handleRefreshKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRList {
		m.loadingState = LoadingState{}
		m.prCache = nil
		return m, m.loadPRsStreaming()
	}
	if isPipelineView(m.state) {
		return m.refreshPipelineView()
	}
	if m.state == ViewDashboard {
		return m.refreshDashboard()
	}
	return m, nil
}

func handleFilterKey(m Model) (Model, tea.Cmd) {
	switch m.state {
	case ViewPRList:
		m.prListView.ActivateFilter()
	case ViewPipelineList:
		m.pipelineListView.ActivateFilter()
	case ViewPipelineRuns:
		m.pipelineRunsView.ActivateFilter()
	}
	return m, nil
}

func handleViewDiffKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRInspect {
		m.prInspect.SwitchToDiff()
		m.topBar.SetView(components.ViewNamePRDiff)
		m.updateShortcuts()
	}
	return m, nil
}

// handleNextFileKey also serves → in PR inspect, which beside the description focuses the work
// items pane instead.
func handleNextFileKey(m Model) (Model, tea.Cmd) {
	if m.state != ViewPRInspect {
		return m, nil
	}
	switch m.prInspect.GetMode() {
	case views.PRInspectModeDiff:
		m.prInspect.NextFile()
	case views.PRInspectModeDescription:
		return handleFocusPaneInKey(m)
	}
	return m, nil
}

func handlePrevFileKey(m Model) (Model, tea.Cmd) {
	if m.state != ViewPRInspect {
		return m, nil
	}
	switch m.prInspect.GetMode() {
	case views.PRInspectModeDiff:
		m.prInspect.PrevFile()
	case views.PRInspectModeDescription:
		return handleFocusPaneOutKey(m)
	}
	return m, nil
}

func handleViewCommentsKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRInspect {
		comments := m.prInspect.GetComments()
		diff := m.prInspect.GetDiff()
		m.commentDetailView.Activate(comments, diff)
		return m, nil
	}
	return m, nil
}

func handleApproveKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRInspect {
		m.reviewView.Activate(views.ReviewModeApprove)
		return m, nil
	}
	return m, nil
}

func handleRequestChangesKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRInspect {
		m.reviewView.Activate(views.ReviewModeRequestChanges)
		return m, nil
	}
	return m, nil
}

func handleMergeCommand(m Model, args []string) (Model, tea.Cmd) {
	return handleMergeKey(m)
}

func handleMergeKey(m Model) (Model, tea.Cmd) {
	if m.state != ViewPRInspect {
		return m, nil
	}

	pr := m.prInspect.GetPR()
	if pr == nil {
		m.statusBar.SetMessage("No PR selected", true)
		return m, nil
	}

	if pr.Status == domain.PRStatusMerged {
		m.statusBar.SetMessage("PR is already merged", true)
		return m, nil
	}

	if pr.Status != domain.PRStatusOpen {
		m.statusBar.SetMessage("Can only merge open PRs", true)
		return m, nil
	}

	provider := m.getProviderForPR(*pr)
	if provider == nil {
		m.statusBar.SetMessage("No provider available", true)
		return m, nil
	}

	m.mergeView.Activate(pr, provider.GetType())
	return m, nil
}

func handleColonKey(m Model) (Model, tea.Cmd) {
	m.commandBar.Activate()
	return m, nil
}

func handleReviewSubmitKey(m Model) (Model, tea.Cmd) {
	if m.reviewView.IsActive() {
		return m, m.submitReview()
	}
	return m, nil
}

func handleEscKey(m Model) (Model, tea.Cmd) {
	if m.isPaneFocused() {
		return handleFocusPaneOutKey(m)
	}
	if m.state == ViewPATs && (m.patsView.Mode == views.PATModeAdd || m.patsView.Mode == views.PATModeEdit) {
		m.patsView.ExitEditMode()
		return m, nil
	}
	if m.reviewView.IsActive() {
		m.reviewView.Deactivate()
		return m, nil
	}
	if m.state == ViewPRList && m.prListView.GetFilterText() != "" {
		m.prListView.ClearFilter()
		return m, nil
	}
	if m.state == ViewPipelineList && m.pipelineListView.GetFilterText() != "" {
		m.pipelineListView.ClearFilter()
		return m, nil
	}
	if m.state == ViewPipelineRuns && m.pipelineRunsView.GetFilterText() != "" {
		m.pipelineRunsView.ClearFilter()
		return m, nil
	}
	newModel, cmd := m.navigateBack()
	return newModel.(Model), cmd
}

func handleOpenBrowserKey(m Model) (Model, tea.Cmd) {
	var url string
	target := "PR"

	switch m.state {
	case ViewPRList:
		pr := m.prListView.GetSelectedPR()
		if pr != nil {
			url = pr.URL
		}
	case ViewPRInspect:
		pr := m.prInspect.GetPR()
		if pr != nil {
			url = pr.URL
		}
	case ViewDashboard:
		if pr := m.dashboardView.GetSelectedPR(); pr != nil {
			url = pr.URL
		}
	case ViewPipelineList, ViewPipelineRuns, ViewPipelineRun:
		url, target = m.pipelineBrowserTarget()
	}

	if url == "" {
		m.statusBar.SetMessage(fmt.Sprintf("No %s URL available", target), true)
		return m, nil
	}

	if err := openExternal(url); err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to open browser: %v", err), true)
		return m, nil
	}

	m.statusBar.SetMessage(fmt.Sprintf("Opening %s in browser...", target), false)
	return m, nil
}

func handleFoldKey(m Model) (Model, tea.Cmd) {
	if m.state != ViewPipelineList {
		return m, nil
	}
	m.pipelineListView.ToggleSelected()
	return m, m.loadPipelinePreview()
}

func handleFoldAllKey(m Model) (Model, tea.Cmd) {
	if m.state != ViewPipelineList {
		return m, nil
	}
	m.pipelineListView.ToggleAll()
	return m, m.loadPipelinePreview()
}

func handleAutoRefreshKey(m Model) (Model, tea.Cmd) {
	return m.toggleAutoRefresh()
}

func handleFavoritesOnlyKey(m Model) (Model, tea.Cmd) {
	return m.toggleFavoritesOnly()
}

func handleFavoriteKey(m Model) (Model, tea.Cmd) {
	if !isPipelineView(m.state) {
		return m, nil
	}
	return m.togglePipelineFavorite()
}

func handleApproveStageKey(m Model) (Model, tea.Cmd) {
	return m.beginStageApproval(domain.ApprovalDecisionApprove)
}

func handleRejectStageKey(m Model) (Model, tea.Cmd) {
	return m.beginStageApproval(domain.ApprovalDecisionReject)
}

func handleInlineCommentKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRInspect && m.prInspect.GetMode() == views.PRInspectModeDiff {
		if m.prInspect.ViewingCommit() != nil {
			m.statusBar.SetMessage(commitCommentsBlockedMessage, true)
			return m, nil
		}
		if target, ok := m.prInspect.CurrentCommentTarget(); ok {
			m.inlineCommentView.Activate(target)
		}
	}
	return m, nil
}

func handleToggleDiffViewKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRInspect && m.prInspect.GetMode() == views.PRInspectModeDiff {
		m.prInspect.ToggleDiffViewMode()
		mode := "full"
		if m.prInspect.GetDiffViewMode() == views.DiffViewModeCompact {
			mode = "compact"
		}
		m.statusBar.SetMessage(fmt.Sprintf("Diff view: %s", mode), false)
	}
	return m, nil
}

func handleYankCurrentFileKey(m Model) (Model, tea.Cmd) {
	if m.state != ViewPRInspect || m.prInspect.GetMode() != views.PRInspectModeDiff {
		return m, nil
	}

	diffText := m.prInspect.GetCurrentFileDiffText()
	if diffText == "" {
		m.statusBar.SetMessage("No diff to copy", true)
		return m, nil
	}

	if err := clipboard.WriteAll(diffText); err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to copy: %v", err), true)
		return m, nil
	}

	m.statusBar.SetMessage("Copied current file diff to clipboard", false)
	return m, nil
}

func handleYankAllFilesKey(m Model) (Model, tea.Cmd) {
	if m.state != ViewPRInspect || m.prInspect.GetMode() != views.PRInspectModeDiff {
		return m, nil
	}

	diffText := m.prInspect.GetAllFilesDiffText()
	if diffText == "" {
		m.statusBar.SetMessage("No diff to copy", true)
		return m, nil
	}

	if err := clipboard.WriteAll(diffText); err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to copy: %v", err), true)
		return m, nil
	}

	diff := m.prInspect.GetDiff()
	fileCount := 0
	if diff != nil {
		fileCount = len(diff.Files)
	}
	m.statusBar.SetMessage(fmt.Sprintf("Copied diff from %d files to clipboard", fileCount), false)
	return m, nil
}

func handleEditDescriptionKey(m Model) (Model, tea.Cmd) {
	if m.state != ViewPRInspect {
		return m, nil
	}

	pr := m.prInspect.GetPR()
	if pr == nil {
		m.statusBar.SetMessage("No PR selected", true)
		return m, nil
	}

	if pr.Category != domain.PRCategoryAuthored {
		m.statusBar.SetMessage("Can only edit description on your own PRs", true)
		return m, nil
	}

	m.descriptionEditView.Activate(pr.Description)
	return m, nil
}

// openExternal and openFolder hand a URL, file or folder to the system. They are variables so the
// tests can replace them; otherwise every test of an "open" key launches a real application.
var (
	openExternal = openWithSystem
	openFolder   = git.OpenFolder
)

// openWithSystem opens a URL or a file with the system's default application.
func openWithSystem(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

func handleLinkCommand(m Model, args []string) (Model, tea.Cmd) {
	var pr *domain.PullRequest

	if m.state == ViewPRInspect {
		pr = m.prInspect.GetPR()
	} else if m.state == ViewPRList {
		pr = m.prListView.GetSelectedPR()
	}

	if pr == nil {
		m.statusBar.SetMessage("No PR selected", true)
		return m, nil
	}

	if len(args) > 0 {
		path := strings.Join(args, " ")
		m.repoLinkView.ActivateWithPath(pr.Repository.FullName, path)
	} else {
		existingLink, _ := m.repository.GetRepositoryLinkByRepoFullName(pr.Repository.FullName)
		if existingLink != nil {
			m.repoLinkView.ActivateWithPath(pr.Repository.FullName, existingLink.LocalPath)
		} else {
			m.repoLinkView.Activate(pr.Repository.FullName)
		}
	}

	return m, nil
}

func handleReposCommand(m Model, args []string) (Model, tea.Cmd) {
	links, err := m.repository.ListRepositoryLinks()
	if err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to load repos: %v", err), true)
		return m, nil
	}

	m.reposView.SetLinks(links)
	m.reposView.Activate()
	return m, nil
}

func handleSettingsCommand(m Model, args []string) (Model, tea.Cmd) {
	return m, m.openSettingsEditor()
}

func handleCheckoutKey(m Model) (Model, tea.Cmd) {
	if m.state != ViewPRInspect {
		return m, nil
	}
	newModel, cmd := m.handleCheckoutBranch()
	return newModel.(Model), cmd
}

func handleOpenFolderKey(m Model) (Model, tea.Cmd) {
	if m.state != ViewPRInspect {
		return m, nil
	}
	newModel, cmd := m.handleOpenRepoFolder()
	return newModel.(Model), cmd
}

func handlePullKey(m Model) (Model, tea.Cmd) {
	if m.state != ViewPRInspect {
		return m, nil
	}
	newModel, cmd := m.handlePullChanges()
	return newModel.(Model), cmd
}

func handleLinkKey(m Model) (Model, tea.Cmd) {
	if m.state != ViewPRInspect {
		return m, nil
	}

	pr := m.prInspect.GetPR()
	if pr == nil {
		m.statusBar.SetMessage("No PR selected", true)
		return m, nil
	}

	existingLink, _ := m.repository.GetRepositoryLinkByRepoFullName(pr.Repository.FullName)
	if existingLink != nil {
		m.repoLinkView.ActivateWithPath(pr.Repository.FullName, existingLink.LocalPath)
	} else {
		m.repoLinkView.Activate(pr.Repository.FullName)
	}

	return m, nil
}

// openCommentsMarkdown writes every comment of the inspected pull request, with the code each is
// about, to a markdown file and opens it with the system's default application.
func (m Model) openCommentsMarkdown() (Model, tea.Cmd) {
	pr := m.prInspect.GetPR()
	number := 0
	if pr != nil {
		number = pr.Number
	}

	file, err := os.CreateTemp("", fmt.Sprintf(commentsMarkdownFilePattern, number))
	if err == nil {
		_, err = file.WriteString(m.commentDetailView.Markdown(pr))
		file.Close()
	}
	if err == nil {
		err = openExternal(file.Name())
	}
	if err != nil {
		logger.LogError("OPEN_COMMENTS_MARKDOWN", "temp file", err)
		m.statusBar.SetMessage(fmt.Sprintf("Failed to open comments as markdown: %v", err), true)
		return m, nil
	}

	logger.LogFileWrite(file.Name())
	m.statusBar.SetMessage("Opened comments as markdown: "+file.Name(), false)
	return m, clearStatusAfterDelay(4 * time.Second)
}

func handleWhitespaceKey(m Model) (Model, tea.Cmd) {
	if m.prInspect.GetMode() != views.PRInspectModeDiff {
		return m, nil
	}
	m.prInspect.ToggleIgnoreWhitespace()
	if m.prInspect.IgnoresWhitespace() {
		m.statusBar.SetMessage("Hiding whitespace-only changes", false)
	} else {
		m.statusBar.SetMessage("Showing whitespace-only changes", false)
	}
	return m, clearStatusAfterDelay(4 * time.Second)
}

func handleFileFilterKey(m Model) (Model, tea.Cmd) {
	if m.prInspect.GetMode() != views.PRInspectModeDiff || len(m.prInspect.Extensions()) == 0 {
		return m, nil
	}
	m.extensionFilterView.Activate(m.prInspect.Extensions(), m.prInspect.HiddenExtensions())
	return m, nil
}

// handleExtensionFilterKey routes keys to the open extension filter; the diff, and with it what
// y and Y copy, changes when the filter is applied.
func (m Model) handleExtensionFilterKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.extensionFilterView.HandleKey(msg) {
	case views.ExtensionFilterActionApply:
		m.prInspect.SetHiddenExtensions(m.extensionFilterView.Hidden())
		m.extensionFilterView.Deactivate()
	case views.ExtensionFilterActionCancel:
		m.extensionFilterView.Deactivate()
	}
	return m, nil
}
