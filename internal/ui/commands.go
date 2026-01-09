package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	Keys        []string
	Description string
	ShortHelp   string
	Handler     KeyHandler
	AvailableIn []ViewState
}

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
			AvailableIn: []ViewState{ViewPATs, ViewPRList, ViewPRInspect},
		},
		{
			Name:        "pr",
			Aliases:     []string{"prs", "pulls"},
			Description: "List pull requests",
			ShortHelp:   ":pr",
			Handler:     handlePRCommand,
			AvailableIn: []ViewState{ViewPATs, ViewPRList, ViewPRInspect},
		},
		{
			Name:        "logs",
			Aliases:     []string{"log"},
			Description: "View session logs",
			ShortHelp:   ":logs",
			Handler:     handleLogsCommand,
			AvailableIn: []ViewState{ViewPATs, ViewPRList, ViewPRInspect},
		},
		{
			Name:        "quit",
			Aliases:     []string{"q", "exit"},
			Description: "Quit application",
			ShortHelp:   ":q",
			Handler:     handleQuitCommand,
			AvailableIn: []ViewState{ViewPATs, ViewPRList, ViewPRInspect},
		},
		{
			Name:        "help",
			Aliases:     []string{"h", "?"},
			Description: "Show help",
			ShortHelp:   ":h",
			Handler:     nil,
			AvailableIn: []ViewState{ViewPATs, ViewPRList, ViewPRInspect},
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
			Keys:        []string{"q", "ctrl+c"},
			Description: "Quit/Back",
			ShortHelp:   "q",
			Handler:     handleQuitKey,
			AvailableIn: []ViewState{ViewPATs, ViewPRList, ViewPRInspect},
		},
		{
			Keys:        []string{"enter"},
			Description: "Select",
			ShortHelp:   "enter",
			Handler:     handleEnterKey,
			AvailableIn: []ViewState{ViewPATs, ViewPRList, ViewPRInspect},
		},
		{
			Keys:        []string{" "},
			Description: "Toggle selection (multi-select)",
			ShortHelp:   "space",
			Handler:     handleSpaceKey,
			AvailableIn: []ViewState{ViewPATs},
		},
		{
			Keys:        []string{"backspace", "h"},
			Description: "Back",
			ShortHelp:   "h",
			Handler:     handleBackKey,
			AvailableIn: []ViewState{ViewPRList, ViewPRInspect},
		},
		{
			Keys:        []string{"up", "k"},
			Description: "Navigate up",
			ShortHelp:   "j/k",
			Handler:     handleUpKey,
			AvailableIn: []ViewState{ViewPATs, ViewPRList, ViewPRInspect},
		},
		{
			Keys:        []string{"down", "j"},
			Description: "Navigate down",
			ShortHelp:   "j/k",
			Handler:     handleDownKey,
			AvailableIn: []ViewState{ViewPATs, ViewPRList, ViewPRInspect},
		},
		{
			Keys:        []string{"a"},
			Description: "Add PAT",
			ShortHelp:   "a",
			Handler:     handleAddKey,
			AvailableIn: []ViewState{ViewPATs},
		},
		{
			Keys:        []string{"d"},
			Description: "Delete PAT",
			ShortHelp:   "d",
			Handler:     handleDeleteKey,
			AvailableIn: []ViewState{ViewPATs},
		},
		{
			Keys:        []string{"e"},
			Description: "Edit PAT",
			ShortHelp:   "e",
			Handler:     handleEditKey,
			AvailableIn: []ViewState{ViewPATs},
		},
		{
			Keys:        []string{"r"},
			Description: "Refresh",
			ShortHelp:   "r",
			Handler:     handleRefreshKey,
			AvailableIn: []ViewState{ViewPRList},
		},
		{
			Keys:        []string{"/"},
			Description: "Filter",
			ShortHelp:   "/",
			Handler:     handleFilterKey,
			AvailableIn: []ViewState{ViewPRList},
		},
		{
			Keys:        []string{"n"},
			Description: "Next file",
			ShortHelp:   "n",
			Handler:     handleNextFileKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{"p"},
			Description: "Previous file",
			ShortHelp:   "p",
			Handler:     handlePrevFileKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{"c"},
			Description: "Toggle comments",
			ShortHelp:   "c",
			Handler:     handleToggleCommentsKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{"a"},
			Description: "Approve PR",
			ShortHelp:   "a",
			Handler:     handleApproveKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{"r"},
			Description: "Request changes",
			ShortHelp:   "r",
			Handler:     handleRequestChangesKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{"d"},
			Description: "View diff",
			ShortHelp:   "d",
			Handler:     handleViewDiffKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{"left"},
			Description: "Previous file",
			ShortHelp:   "left",
			Handler:     handlePrevFileKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{"right"},
			Description: "Next file",
			ShortHelp:   "right",
			Handler:     handleNextFileKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{":"},
			Description: "Command mode",
			ShortHelp:   ":",
			Handler:     handleColonKey,
			AvailableIn: []ViewState{ViewPATs, ViewPRList, ViewPRInspect},
		},
		{
			Keys:        []string{"ctrl+s"},
			Description: "Submit review",
			ShortHelp:   "ctrl+s",
			Handler:     handleReviewSubmitKey,
			AvailableIn: []ViewState{ViewPRInspect},
		},
		{
			Keys:        []string{"esc"},
			Description: "Cancel/Back",
			ShortHelp:   "esc",
			Handler:     handleEscKey,
			AvailableIn: []ViewState{ViewPATs, ViewPRList, ViewPRInspect},
		},
		{
			Keys:        []string{"ctrl+o"},
			Description: "Open PR in browser",
			ShortHelp:   "ctrl+o",
			Handler:     handleOpenBrowserKey,
			AvailableIn: []ViewState{ViewPRList, ViewPRInspect},
		},
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
		parts = append(parts, nameWithAliases)
	}

	return "Commands: " + strings.Join(parts, " | ")
}

func (cr *CommandRegistry) GetContextualShortcuts(state ViewState) []string {
	var shortcuts []string
	seen := make(map[string]bool)

	for _, kb := range cr.keyBindings {
		if !isInViews(state, kb.AvailableIn) {
			continue
		}
		if kb.Description == "" {
			continue
		}
		if seen[kb.ShortHelp] {
			continue
		}
		seen[kb.ShortHelp] = true
		shortcuts = append(shortcuts, fmt.Sprintf("<%s> %s", kb.ShortHelp, kb.Description))
	}

	return shortcuts
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
	m.topBar.SetView("PATs")
	m.topBar.SetContext("", "")
	m.topBar.SetStats(0, 0)
	m.topBar.SetPRBreakdown(0, 0, 0)
	m.updateShortcuts()
	m.statusBar.SetMessage("Showing PATs", false)
	return m, m.loadPATs()
}

func handlePRCommand(m Model, args []string) (Model, tea.Cmd) {
	if m.provider == nil {
		m.statusBar.SetMessage("No active PAT. Please select a PAT first.", true)
		return m, nil
	}
	m.statusBar.SetMessage("Loading pull requests...", false)
	return m, m.loadPRs()
}

func handleLogsCommand(m Model, args []string) (Model, tea.Cmd) {
	m.logsView.Activate()
	return m, nil
}

func handleQuitCommand(m Model, args []string) (Model, tea.Cmd) {
	return m, tea.Quit
}

func handleQuitKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPATs {
		return m, tea.Quit
	}

	if m.state == ViewPRInspect && m.prInspect.GetMode() == views.PRInspectModeDiff {
		m.prInspect.SwitchToDescription()
		m.topBar.SetView("PR Description")
		m.updateShortcuts()
		return m, nil
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
		pr := m.prListView.GetSelectedPR()
		if pr != nil {
			m.state = ViewPRInspect
			m.prInspect.SwitchToDescription()
			m.topBar.SetContext(pr.Repository.FullName, fmt.Sprintf("%d", pr.Number))
			m.topBar.SetView("PR Description")
			m.updateShortcuts()
			return m, tea.Batch(
				m.loadPRDetail(*pr),
				m.loadDiff(*pr),
				m.loadComments(*pr),
			)
		}
	case ViewPRInspect:
		if m.prInspect.GetMode() == views.PRInspectModeDiff {
			m.reviewView.Activate(views.ReviewModeComment)
		}
		return m, nil
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
	switch m.state {
	case ViewPATs:
		cmd = m.patsView.Update(tea.KeyMsg{Type: tea.KeyUp})
	case ViewPRList:
		cmd = m.prListView.Update(tea.KeyMsg{Type: tea.KeyUp})
	case ViewPRInspect:
		cmd = m.prInspect.Update(tea.KeyMsg{Type: tea.KeyUp})
	}
	return m, cmd
}

func handleDownKey(m Model) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.state {
	case ViewPATs:
		cmd = m.patsView.Update(tea.KeyMsg{Type: tea.KeyDown})
	case ViewPRList:
		cmd = m.prListView.Update(tea.KeyMsg{Type: tea.KeyDown})
	case ViewPRInspect:
		cmd = m.prInspect.Update(tea.KeyMsg{Type: tea.KeyDown})
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
		m.statusBar.SetMessage("Refreshing pull requests...", false)
		return m, m.loadPRs()
	}
	return m, nil
}

func handleFilterKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRList {
		return m, nil
	}
	return m, nil
}

func handleViewDiffKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRInspect {
		m.prInspect.SwitchToDiff()
		m.topBar.SetView("PR Diff")
		m.updateShortcuts()
	}
	return m, nil
}

func handleNextFileKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRInspect && m.prInspect.GetMode() == views.PRInspectModeDiff {
		m.prInspect.NextFile()
		return m, nil
	}
	return m, nil
}

func handlePrevFileKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRInspect && m.prInspect.GetMode() == views.PRInspectModeDiff {
		m.prInspect.PrevFile()
		return m, nil
	}
	return m, nil
}

func handleToggleCommentsKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRInspect && m.prInspect.GetMode() == views.PRInspectModeDiff {
		m.prInspect.ToggleComments()
		return m, nil
	}
	return m, nil
}

func handleApproveKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRInspect && m.prInspect.GetMode() == views.PRInspectModeDiff {
		m.reviewView.Activate(views.ReviewModeApprove)
		return m, nil
	}
	return m, nil
}

func handleRequestChangesKey(m Model) (Model, tea.Cmd) {
	if m.state == ViewPRInspect && m.prInspect.GetMode() == views.PRInspectModeDiff {
		m.reviewView.Activate(views.ReviewModeRequestChanges)
		return m, nil
	}
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
	if m.state == ViewPATs && (m.patsView.Mode == views.PATModeAdd || m.patsView.Mode == views.PATModeEdit) {
		m.patsView.ExitEditMode()
		return m, nil
	}
	if m.reviewView.IsActive() {
		m.reviewView.Deactivate()
		return m, nil
	}
	newModel, cmd := m.navigateBack()
	return newModel.(Model), cmd
}

func handleOpenBrowserKey(m Model) (Model, tea.Cmd) {
	var url string

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
	}

	if url == "" {
		m.statusBar.SetMessage("No PR URL available", true)
		return m, nil
	}

	if err := openBrowser(url); err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to open browser: %v", err), true)
		return m, nil
	}

	m.statusBar.SetMessage("Opening PR in browser...", false)
	return m, nil
}

func openBrowser(url string) error {
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
