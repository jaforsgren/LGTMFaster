package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/ui/components"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

type ViewState int

const (
	ViewPATs ViewState = iota
	ViewPRList
	ViewPRInspect
)

type Model struct {
	state            ViewState
	width            int
	height           int
	topBar           *components.TopBarModel
	statusBar        *components.StatusBarModel
	commandBar       *components.CommandBarModel
	patsView         *views.PATsViewModel
	prListView       *views.PRListViewModel
	prInspect        *views.PRInspectViewModel
	reviewView       *views.ReviewViewModel
	logsView         *views.LogsViewModel
	repository       domain.Repository
	providerManager  *ProviderManager
	prMetadata       map[string]string // Maps PR identifier (repo/number) to PATID
	ctx              context.Context
	commandRegistry  *CommandRegistry
	isInitialStartup bool
}

func NewModel(repository domain.Repository) Model {
	return Model{
		state:            ViewPATs,
		topBar:           components.NewTopBar(),
		statusBar:        components.NewStatusBar(),
		commandBar:       components.NewCommandBar(),
		patsView:         views.NewPATsView(),
		prListView:       views.NewPRListView(),
		prInspect:        views.NewPRInspectView(),
		reviewView:       views.NewReviewView(),
		logsView:         views.NewLogsView(),
		repository:       repository,
		providerManager:  NewProviderManager(),
		prMetadata:       make(map[string]string),
		ctx:              context.Background(),
		commandRegistry:  NewCommandRegistry(),
		isInitialStartup: true,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadPATs()
}

func (m Model) isInInputMode() bool {
	if m.commandBar.IsActive() {
		return true
	}
	if m.reviewView.IsActive() {
		return true
	}
	if m.logsView.IsActive() {
		return true
	}
	if m.state == ViewPATs && (m.patsView.Mode == views.PATModeAdd || m.patsView.Mode == views.PATModeEdit) {
		return true
	}
	return false
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.topBar.SetWidth(msg.Width)
		m.statusBar.SetWidth(msg.Width)
		m.commandBar.SetWidth(msg.Width)
		m.patsView.SetSize(msg.Width, msg.Height)
		m.prListView.SetSize(msg.Width, msg.Height)
		m.prInspect.SetSize(msg.Width, msg.Height)
		m.reviewView.SetSize(msg.Width, msg.Height)
		m.logsView.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		key := msg.String()

		if m.isInInputMode() {
			if m.commandBar.IsActive() {
				switch key {
				case "enter":
					return m.handleCommand()
				case "esc":
					m.commandBar.Deactivate()
					return m, nil
				default:
					cmd = m.commandBar.Update(msg)
					return m, cmd
				}
			}

			if m.reviewView.IsActive() {
				switch key {
				case "ctrl+s":
					return m, m.submitReview()
				case "esc":
					m.reviewView.Deactivate()
					return m, nil
				default:
					cmd = m.reviewView.Update(msg)
					return m, cmd
				}
			}

			if m.logsView.IsActive() {
				switch key {
				case "esc", "q":
					m.logsView.Deactivate()
					return m, nil
				default:
					cmd = m.logsView.Update(msg)
					return m, cmd
				}
			}

			if m.state == ViewPATs && (m.patsView.Mode == views.PATModeAdd || m.patsView.Mode == views.PATModeEdit) {
				switch key {
				case "enter":
					return m.handlePATEnter()
				case "esc":
					m.patsView.ExitEditMode()
					return m, nil
				default:
					cmd = m.patsView.Update(msg)
					return m, cmd
				}
			}
		}

		newModel, cmd, handled := m.commandRegistry.HandleKey(m, key)
		if handled {
			return newModel, cmd
		}

	case PATsLoadedMsg:
		m.patsView.SetPATs(msg.pats)

		// Initialize providers using ProviderManager
		if err := m.providerManager.InitializeProviders(msg.pats); err != nil {
			m.statusBar.SetMessage(fmt.Sprintf("Failed to initialize providers: %v", err), true)
		}

		// Count selected PATs and set up top bar
		selectedCount := 0
		for _, pat := range msg.pats {
			if pat.IsSelected {
				selectedCount++
				if pat.IsPrimary {
					m.topBar.SetActivePAT(pat.Name, string(pat.Provider))
				}
			}
		}

		if selectedCount > 1 {
			m.topBar.SetSelectedPATCount(selectedCount)
		}

		if selectedCount > 0 && m.isInitialStartup {
			m.isInitialStartup = false
			m.state = ViewPRList
			m.topBar.SetView("PRs")
			m.updateShortcuts()
			logger.Log("UI: Starting in PR list view with %d selected PAT(s)", selectedCount)
			return m, m.loadPRs()
		}

		m.isInitialStartup = false
		m.topBar.SetView("PATs")
		m.updateShortcuts()
		return m, nil

	case PRsLoadedMsg:
		// Store PR metadata (PATID) in UI layer from groups
		if msg.groups != nil {
			for _, group := range msg.groups {
				for _, pr := range group.PRs {
					m.setPATIDForPR(pr, group.PATID)
				}
			}
		}

		if msg.groups != nil && len(msg.groups) > 0 {
			m.prListView.SetPRGroups(msg.groups)
		} else {
			m.prListView.SetPRs(msg.prs)
		}

		repoMap := make(map[string]bool)
		authored, assigned, other := 0, 0, 0
		for _, pr := range msg.prs {
			repoMap[pr.Repository.FullName] = true
			switch pr.Category {
			case domain.PRCategoryAuthored:
				authored++
			case domain.PRCategoryAssigned:
				assigned++
			default:
				other++
			}
		}
		m.topBar.SetStats(len(msg.prs), len(repoMap))
		m.topBar.SetPRBreakdown(authored, assigned, other)
		m.topBar.SetView("PR List")

		m.state = ViewPRList
		m.updateShortcuts()
		m.statusBar.SetMessage(fmt.Sprintf("Loaded %d pull requests", len(msg.prs)), false)
		return m, clearStatusAfterDelay(4 * time.Second)

	case PRDetailLoadedMsg:
		m.prInspect.SetPR(msg.pr)
		return m, nil

	case DiffLoadedMsg:
		logger.Log("UI: DiffLoadedMsg received - diff has %d files", len(msg.diff.Files))
		for i, file := range msg.diff.Files {
			filePath := file.NewPath
			if filePath == "" {
				filePath = file.OldPath
			}
			logger.Log("UI: DiffLoadedMsg - File %d: %s (%d hunks)", i+1, filePath, len(file.Hunks))
			for j, hunk := range file.Hunks {
				logger.Log("UI: DiffLoadedMsg - File %d Hunk %d: %s (%d lines)", i+1, j+1, hunk.Header, len(hunk.Lines))
			}
		}
		m.prInspect.SetDiff(msg.diff)
		logger.Log("UI: SetDiff called on prInspect view")
		return m, nil

	case CommentsLoadedMsg:
		m.prInspect.SetComments(msg.comments)
		return m, nil

	case ErrorMsg:
		m.statusBar.SetMessage(msg.err.Error(), true)
		return m, nil

	case SuccessMsg:
		m.statusBar.SetMessage(msg.message, false)
		if msg.reloadComments && msg.reloadCommentsPR != nil {
			return m, m.loadComments(*msg.reloadCommentsPR)
		}
		return m, nil

	case ClearStatusMsg:
		m.statusBar.ClearMessage()
		return m, nil
	}

	switch m.state {
	case ViewPATs:
		cmd = m.patsView.Update(msg)
	case ViewPRList:
		cmd = m.prListView.Update(msg)
	case ViewPRInspect:
		cmd = m.prInspect.Update(msg)
	}

	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string

	if m.logsView.IsActive() {
		content = m.logsView.View()
	} else if m.reviewView.IsActive() {
		content = m.reviewView.View()
	} else {
		switch m.state {
		case ViewPATs:
			content = m.patsView.View()
		case ViewPRList:
			content = m.prListView.View()
		case ViewPRInspect:
			content = m.prInspect.View()
		}
	}

	topBar := m.topBar.View()
	statusBar := m.statusBar.View()
	commandBar := m.commandBar.View()

	if commandBar != "" {
		return topBar + "\n" + content + "\n" + commandBar
	}

	return topBar + "\n" + content + "\n" + statusBar
}

func (m Model) handleCommand() (tea.Model, tea.Cmd) {
	input := m.commandBar.Value()
	m.commandBar.Deactivate()

	input = strings.TrimPrefix(input, ":")
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return m, nil
	}

	cmdName := parts[0]
	args := parts[1:]

	logger.Log("UI: Executing command: %s %v", cmdName, args)
	return m.commandRegistry.ExecuteCommand(m, cmdName, args)
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case ViewPATs:
		return m.handlePATEnter()
	case ViewPRList:
		pr := m.prListView.GetSelectedPR()
		if pr != nil {
			m.state = ViewPRInspect
			m.topBar.SetContext(pr.Repository.FullName, fmt.Sprintf("%d", pr.Number))
			m.topBar.SetView("PR Inspect")
			m.updateShortcuts()
			return m, tea.Batch(
				m.loadPRDetail(*pr),
				m.loadDiff(*pr),
				m.loadComments(*pr),
			)
		}
	}
	return m, nil
}

func (m Model) handlePATSpaceToggle() (tea.Model, tea.Cmd) {
	if m.patsView.Mode != views.PATModeList {
		return m, nil
	}

	pat := m.patsView.GetSelectedPAT()
	if pat == nil {
		return m, nil
	}

	if err := m.repository.TogglePATSelection(pat.ID); err != nil {
		return m, func() tea.Msg {
			return ErrorMsg{err: err}
		}
	}

	return m, m.loadPATs()
}

func (m Model) handlePATEnter() (tea.Model, tea.Cmd) {
	if m.patsView.Mode == views.PATModeAdd {
		newPAT := m.patsView.GetPATData()
		newPAT.ID = uuid.New().String()

		if err := m.repository.SavePAT(newPAT); err != nil {
			return m, func() tea.Msg {
				return ErrorMsg{err: err}
			}
		}

		m.patsView.ExitEditMode()
		m.statusBar.SetMessage("PAT added successfully", false)
		return m, m.loadPATs()
	}

	if m.patsView.Mode == views.PATModeEdit {
		updatedPAT := m.patsView.GetPATData()

		if err := m.repository.SavePAT(updatedPAT); err != nil {
			return m, func() tea.Msg {
				return ErrorMsg{err: err}
			}
		}

		m.patsView.ExitEditMode()
		m.topBar.SetActivePAT(updatedPAT.Name, string(updatedPAT.Provider))
		m.statusBar.SetMessage("PAT updated successfully", false)
		return m, m.loadPATs()
	}

	if m.patsView.Mode == views.PATModeList {
		if m.providerManager.ProviderCount() > 0 {
			m.statusBar.SetMessage("Loading pull requests...", false)
			return m, m.loadPRs()
		}

		pat := m.patsView.GetSelectedPAT()
		if pat != nil {
			if err := m.repository.SetActivePAT(pat.ID); err != nil {
				return m, func() tea.Msg {
					return ErrorMsg{err: err}
				}
			}
			m.topBar.SetActivePAT(pat.Name, string(pat.Provider))
			m.statusBar.SetMessage(fmt.Sprintf("Activated PAT: %s", pat.Name), false)
			return m, m.loadPATs()
		}
	}

	return m, nil
}

func (m Model) handleDeletePAT() (tea.Model, tea.Cmd) {
	pat := m.patsView.GetSelectedPAT()
	if pat == nil {
		return m, nil
	}

	if err := m.repository.DeletePAT(pat.ID); err != nil {
		return m, func() tea.Msg {
			return ErrorMsg{err: err}
		}
	}

	return m, m.loadPATs()
}

func (m Model) navigateBack() (tea.Model, tea.Cmd) {
	switch m.state {
	case ViewPRList:
		logger.Log("UI: Navigating back from PR List to PATs")
		m.state = ViewPATs
		m.topBar.SetContext("", "")
		m.topBar.SetStats(0, 0)
		m.topBar.SetPRBreakdown(0, 0, 0)
		m.topBar.SetView("PATs")
		m.updateShortcuts()
		return m, nil
	case ViewPRInspect:
		logger.Log("UI: Navigating back from PR Inspect to PR List")
		m.state = ViewPRList
		m.topBar.SetContext("", "")
		m.topBar.SetView("PR List")
		m.updateShortcuts()
		return m, nil
	}
	return m, nil
}

func (m Model) submitReview() tea.Cmd {
	review := m.reviewView.GetReview()
	m.reviewView.Deactivate()

	pr := m.prInspect.GetPR()
	if pr == nil {
		logger.LogError("SUBMIT_REVIEW", "UI", fmt.Errorf("no PR selected"))
		return func() tea.Msg {
			return ErrorMsg{err: fmt.Errorf("no PR selected")}
		}
	}

	provider := m.getProviderForPR(*pr)
	if provider == nil {
		logger.LogError("SUBMIT_REVIEW", "UI", fmt.Errorf("no provider available for PR"))
		return func() tea.Msg {
			return ErrorMsg{err: fmt.Errorf("no provider available for PR")}
		}
	}

	var authenticatedUser string
	patID := m.getPATIDForPR(*pr)
	if patID != "" {
		pat, err := m.repository.GetPAT(patID)
		if err == nil && pat != nil {
			authenticatedUser = pat.Username
		}
	}

	isOwnPR := authenticatedUser != "" && pr.Author.Username == authenticatedUser
	if isOwnPR && (review.Action == domain.ReviewActionApprove || review.Action == domain.ReviewActionRequestChanges) {
		logger.Log("UI: Cannot %s your own PR, converting to comment", review.Action)
		review.Action = domain.ReviewActionComment
	}

	review.PRIdentifier = fmt.Sprintf("%s/%d", pr.Repository.FullName, pr.Number)

	logger.Log("UI: Submitting review for %s using provider (PATID: %s, Action: %s)",
		review.PRIdentifier, patID, review.Action)
	return func() tea.Msg {
		if err := provider.SubmitReview(m.ctx, review); err != nil {
			return ErrorMsg{err: err}
		}
		return SuccessMsg{
			message:          "Review submitted successfully",
			reloadComments:   true,
			reloadCommentsPR: pr,
		}
	}
}

func (m Model) loadPATs() tea.Cmd {
	return func() tea.Msg {
		pats, err := m.repository.ListPATs()
		if err != nil {
			return ErrorMsg{err: err}
		}
		return PATsLoadedMsg{pats: pats}
	}
}

func (m Model) loadPRs() tea.Cmd {
	if !m.providerManager.HasProviders() {
		return func() tea.Msg {
			return ErrorMsg{err: fmt.Errorf("no PATs selected")}
		}
	}

	return func() tea.Msg {
		// Single provider mode (backwards compatibility)
		if m.providerManager.ProviderCount() == 0 && m.providerManager.GetSingleProvider() != nil {
			pat, err := m.repository.GetActivePAT()
			if err != nil {
				return ErrorMsg{err: err}
			}

			prs, err := m.providerManager.GetSingleProvider().ListPullRequests(m.ctx, pat.Username)
			if err != nil {
				return ErrorMsg{err: err}
			}
			return PRsLoadedMsg{prs: prs, groups: nil}
		}

		selectedPATs, err := m.repository.GetSelectedPATs()
		if err != nil {
			return ErrorMsg{err: err}
		}

		type prResult struct {
			prs []domain.PullRequest
			pat domain.PAT
			err error
		}

		results := make(chan prResult, len(selectedPATs))

		for _, pat := range selectedPATs {
			go func(p domain.PAT) {
				provider := m.providerManager.GetProviderByPATID(p.ID)
				if provider == nil {
					results <- prResult{prs: nil, pat: p, err: fmt.Errorf("provider not found for PAT %s", p.Name)}
					return
				}
				prs, err := provider.ListPullRequests(m.ctx, p.Username)
				results <- prResult{prs: prs, pat: p, err: err}
			}(pat)
		}

		var allGroups []domain.PRGroup
		var allPRs []domain.PullRequest

		for i := 0; i < len(selectedPATs); i++ {
			result := <-results
			if result.err != nil {
				logger.LogError("LOAD_PRS", result.pat.Name, result.err)
				continue
			}

			// PR metadata (PATID) will be extracted from groups in the handler
			allGroups = append(allGroups, domain.PRGroup{
				PATName:   result.pat.Name,
				PATID:     result.pat.ID,
				Provider:  result.pat.Provider,
				Username:  result.pat.Username,
				IsPrimary: result.pat.IsPrimary,
				PRs:       result.prs,
			})
			allPRs = append(allPRs, result.prs...)
		}

		return PRsLoadedMsg{prs: allPRs, groups: allGroups}
	}
}

func (m Model) loadPRDetail(pr domain.PullRequest) tea.Cmd {
	return func() tea.Msg {
		provider := m.getProviderForPR(pr)
		if provider == nil {
			return ErrorMsg{err: fmt.Errorf("no provider available for PR")}
		}

		identifier := domain.PRIdentifier{
			Provider:   provider.GetType(),
			Repository: pr.Repository.FullName,
			Number:     pr.Number,
		}

		prDetail, err := provider.GetPullRequest(m.ctx, identifier)
		if err != nil {
			return ErrorMsg{err: err}
		}

		// No need to copy metadata - it's already stored in prMetadata map by identifier

		return PRDetailLoadedMsg{pr: prDetail}
	}
}

func (m Model) loadDiff(pr domain.PullRequest) tea.Cmd {
	return func() tea.Msg {
		provider := m.getProviderForPR(pr)
		if provider == nil {
			return ErrorMsg{err: fmt.Errorf("no provider available for PR")}
		}

		patID := m.getPATIDForPR(pr)
		logger.Log("Loading diff for PR #%d using provider %s (PATID: %s)", pr.Number, provider.GetType(), patID)

		identifier := domain.PRIdentifier{
			Provider:   provider.GetType(),
			Repository: pr.Repository.FullName,
			Number:     pr.Number,
		}

		diff, err := provider.GetDiff(m.ctx, identifier)
		if err != nil {
			logger.LogError("LOAD_DIFF", fmt.Sprintf("PR #%d provider %s", pr.Number, provider.GetType()), err)
			return ErrorMsg{err: err}
		}
		return DiffLoadedMsg{diff: diff}
	}
}

func (m Model) loadComments(pr domain.PullRequest) tea.Cmd {
	return func() tea.Msg {
		provider := m.getProviderForPR(pr)
		if provider == nil {
			return ErrorMsg{err: fmt.Errorf("no provider available for PR")}
		}

		identifier := domain.PRIdentifier{
			Provider:   provider.GetType(),
			Repository: pr.Repository.FullName,
			Number:     pr.Number,
		}

		comments, err := provider.GetComments(m.ctx, identifier)
		if err != nil {
			return ErrorMsg{err: err}
		}
		return CommentsLoadedMsg{comments: comments}
	}
}

func (m Model) getPRIdentifier(pr domain.PullRequest) string {
	return fmt.Sprintf("%s/%d", pr.Repository.FullName, pr.Number)
}

func (m Model) getPATIDForPR(pr domain.PullRequest) string {
	return m.prMetadata[m.getPRIdentifier(pr)]
}

func (m Model) setPATIDForPR(pr domain.PullRequest, patID string) {
	m.prMetadata[m.getPRIdentifier(pr)] = patID
}

func (m Model) getProviderForPR(pr domain.PullRequest) domain.Provider {
	patID := m.getPATIDForPR(pr)

	// Try to get provider by PATID
	if patID != "" {
		if provider := m.providerManager.GetProviderByPATID(patID); provider != nil {
			return provider
		}
	}

	// Fallback to primary or single provider
	if primary, _ := m.providerManager.GetPrimaryProvider(); primary != nil {
		return primary
	}

	return m.providerManager.GetSingleProvider()
}

func (m Model) updateShortcuts() {
	shortcuts := m.commandRegistry.GetContextualShortcuts(m.state)
	m.topBar.SetShortcuts(shortcuts)
}

func clearStatusAfterDelay(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

type PATsLoadedMsg struct {
	pats []domain.PAT
}

type PRsLoadedMsg struct {
	prs    []domain.PullRequest
	groups []domain.PRGroup
}

type PRDetailLoadedMsg struct {
	pr *domain.PullRequest
}

type DiffLoadedMsg struct {
	diff *domain.Diff
}

type CommentsLoadedMsg struct {
	comments []domain.Comment
}

type ErrorMsg struct {
	err error
}

type SuccessMsg struct {
	message             string
	reloadComments      bool
	reloadCommentsPR    *domain.PullRequest
}

type ClearStatusMsg struct{}
