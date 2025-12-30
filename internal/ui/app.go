package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/azuredevops"
	"github.com/johanforsgren/lgtmfaster/internal/provider/github"
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
	state           ViewState
	width           int
	height          int
	topBar          *components.TopBarModel
	statusBar       *components.StatusBarModel
	commandBar      *components.CommandBarModel
	patsView        *views.PATsViewModel
	prListView      *views.PRListViewModel
	prInspect       *views.PRInspectViewModel
	reviewView      *views.ReviewViewModel
	logsView        *views.LogsViewModel
	repository      domain.Repository
	provider        domain.Provider
	ctx             context.Context
	commandRegistry *CommandRegistry
}

func NewModel(repository domain.Repository) Model {
	return Model{
		state:           ViewPATs,
		topBar:          components.NewTopBar(),
		statusBar:       components.NewStatusBar(),
		commandBar:      components.NewCommandBar(),
		patsView:        views.NewPATsView(),
		prListView:      views.NewPRListView(),
		prInspect:       views.NewPRInspectView(),
		reviewView:      views.NewReviewView(),
		logsView:        views.NewLogsView(),
		repository:      repository,
		ctx:             context.Background(),
		commandRegistry: NewCommandRegistry(),
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
				cmd = m.reviewView.Update(msg)
				return m, cmd
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
				cmd = m.patsView.Update(msg)
				return m, cmd
			}
		}

		newModel, cmd, handled := m.commandRegistry.HandleKey(m, key)
		if handled {
			return newModel, cmd
		}

	case PATsLoadedMsg:
		m.patsView.SetPATs(msg.pats)
		if len(msg.pats) > 0 {
			for _, pat := range msg.pats {
				if pat.IsActive {
					m.topBar.SetActivePAT(pat.Name, string(pat.Provider))
					provider, err := m.createProvider(pat)
					if err != nil {
						m.statusBar.SetMessage(fmt.Sprintf("Failed to create provider: %v", err), true)
					} else {
						m.provider = provider
					}
					break
				}
			}
		}
		m.topBar.SetView("PATs")
		m.updateShortcuts()
		return m, nil

	case PRsLoadedMsg:
		m.prListView.SetPRs(msg.prs)

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
		return m, nil

	case PRDetailLoadedMsg:
		m.prInspect.SetPR(msg.pr)
		return m, nil

	case DiffLoadedMsg:
		m.prInspect.SetDiff(msg.diff)
		return m, nil

	case CommentsLoadedMsg:
		m.prInspect.SetComments(msg.comments)
		return m, nil

	case ErrorMsg:
		m.statusBar.SetMessage(msg.err.Error(), true)
		return m, nil

	case SuccessMsg:
		m.statusBar.SetMessage(msg.message, false)
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

	if m.patsView.GetSelectedPAT() != nil {
		pat := m.patsView.GetSelectedPAT()
		if err := m.repository.SetActivePAT(pat.ID); err != nil {
			return m, func() tea.Msg {
				return ErrorMsg{err: err}
			}
		}
		provider, err := m.createProvider(*pat)
		if err != nil {
			return m, func() tea.Msg {
				return ErrorMsg{err: err}
			}
		}
		m.provider = provider
		m.topBar.SetActivePAT(pat.Name, string(pat.Provider))
		m.statusBar.SetMessage(fmt.Sprintf("Activated PAT: %s", pat.Name), false)
		return m, m.loadPATs()
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

	review.PRIdentifier = fmt.Sprintf("%s/%d", pr.Repository.FullName, pr.Number)

	logger.Log("UI: Submitting review for %s (Action: %s)", review.PRIdentifier, review.Action)
	return func() tea.Msg {
		if err := m.provider.SubmitReview(m.ctx, review); err != nil {
			return ErrorMsg{err: err}
		}
		return SuccessMsg{message: "Review submitted successfully"}
	}
}

func (m Model) createProvider(pat domain.PAT) (domain.Provider, error) {
	switch pat.Provider {
	case domain.ProviderGitHub:
		return github.NewProvider(pat.Token, pat.Username), nil
	case domain.ProviderAzureDevOps:
		provider, err := azuredevops.NewProvider(pat.Token, pat.Organization, pat.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure DevOps provider: %w", err)
		}
		return provider, nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", pat.Provider)
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
	if m.provider == nil {
		return func() tea.Msg {
			return ErrorMsg{err: fmt.Errorf("no provider configured")}
		}
	}

	return func() tea.Msg {
		pat, err := m.repository.GetActivePAT()
		if err != nil {
			return ErrorMsg{err: err}
		}

		prs, err := m.provider.ListPullRequests(m.ctx, pat.Username)
		if err != nil {
			return ErrorMsg{err: err}
		}
		return PRsLoadedMsg{prs: prs}
	}
}

func (m Model) loadPRDetail(pr domain.PullRequest) tea.Cmd {
	return func() tea.Msg {
		identifier := domain.PRIdentifier{
			Provider:   m.provider.GetType(),
			Repository: pr.Repository.FullName,
			Number:     pr.Number,
		}

		prDetail, err := m.provider.GetPullRequest(m.ctx, identifier)
		if err != nil {
			return ErrorMsg{err: err}
		}
		return PRDetailLoadedMsg{pr: prDetail}
	}
}

func (m Model) loadDiff(pr domain.PullRequest) tea.Cmd {
	return func() tea.Msg {
		identifier := domain.PRIdentifier{
			Provider:   m.provider.GetType(),
			Repository: pr.Repository.FullName,
			Number:     pr.Number,
		}

		diff, err := m.provider.GetDiff(m.ctx, identifier)
		if err != nil {
			return ErrorMsg{err: err}
		}
		return DiffLoadedMsg{diff: diff}
	}
}

func (m Model) loadComments(pr domain.PullRequest) tea.Cmd {
	return func() tea.Msg {
		identifier := domain.PRIdentifier{
			Provider:   m.provider.GetType(),
			Repository: pr.Repository.FullName,
			Number:     pr.Number,
		}

		comments, err := m.provider.GetComments(m.ctx, identifier)
		if err != nil {
			return ErrorMsg{err: err}
		}
		return CommentsLoadedMsg{comments: comments}
	}
}

func (m Model) updateShortcuts() {
	shortcuts := m.commandRegistry.GetContextualShortcuts(m.state)
	m.topBar.SetShortcuts(shortcuts)
}

type PATsLoadedMsg struct {
	pats []domain.PAT
}

type PRsLoadedMsg struct {
	prs []domain.PullRequest
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
	message string
}
