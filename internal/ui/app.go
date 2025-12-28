package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
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
	state      ViewState
	width      int
	height     int
	topBar     *components.TopBarModel
	statusBar  *components.StatusBarModel
	commandBar *components.CommandBarModel
	patsView   *views.PATsViewModel
	prListView *views.PRListViewModel
	prInspect  *views.PRInspectViewModel
	reviewView *views.ReviewViewModel
	repository domain.Repository
	provider   domain.Provider
	ctx        context.Context
}

func NewModel(repository domain.Repository) Model {
	return Model{
		state:      ViewPATs,
		topBar:     components.NewTopBar(),
		statusBar:  components.NewStatusBar(),
		commandBar: components.NewCommandBar(),
		patsView:   views.NewPATsView(),
		prListView: views.NewPRListView(),
		prInspect:  views.NewPRInspectView(),
		reviewView: views.NewReviewView(),
		repository: repository,
		ctx:        context.Background(),
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadPATs()
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

	case tea.KeyMsg:
		if m.reviewView.IsActive() {
			switch msg.String() {
			case "esc":
				m.reviewView.Deactivate()
				return m, nil
			case "ctrl+enter":
				return m, m.submitReview()
			default:
				cmd = m.reviewView.Update(msg)
				return m, cmd
			}
		}

		if m.commandBar.IsActive() {
			switch msg.String() {
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

		switch msg.String() {
		case "ctrl+c", "q":
			if m.state == ViewPATs {
				return m, tea.Quit
			}
			return m.navigateBack()

		case ":":
			m.commandBar.Activate()
			cmd = m.commandBar.Update(msg)
			return m, cmd

		case "enter":
			return m.handleEnter()

		case "a":
			if m.state == ViewPRInspect {
				m.reviewView.Activate(views.ReviewModeApprove)
				return m, nil
			} else if m.state == ViewPATs {
				m.patsView.EnterAddMode()
				return m, nil
			}

		case "r":
			if m.state == ViewPRInspect {
				m.reviewView.Activate(views.ReviewModeRequestChanges)
				return m, nil
			} else if m.state == ViewPRList {
				return m, m.loadPRs()
			}

		case "d":
			if m.state == ViewPATs {
				return m.handleDeletePAT()
			}

		case "esc":
			if m.state == ViewPATs && m.patsView.Mode == views.PATModeAdd {
				m.patsView.ExitAddMode()
				return m, nil
			}
		}

	case PATsLoadedMsg:
		m.patsView.SetPATs(msg.pats)
		if len(msg.pats) > 0 {
			for _, pat := range msg.pats {
				if pat.IsActive {
					m.topBar.SetActivePAT(pat.Name, string(pat.Provider))
					m.provider = m.createProvider(pat)
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

	if m.reviewView.IsActive() {
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
	command := ParseCommand(m.commandBar.Value())
	m.commandBar.Deactivate()

	switch command.Type {
	case CommandQuit:
		return m, tea.Quit
	case CommandPATs:
		m.state = ViewPATs
		return m, m.loadPATs()
	case CommandPR:
		if m.provider == nil {
			m.statusBar.SetMessage("No active PAT. Please select a PAT first.", true)
			return m, nil
		}
		return m, m.loadPRs()
	default:
		m.statusBar.SetMessage("Unknown command", true)
		return m, nil
	}
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
		newPAT := m.patsView.GetNewPAT()
		newPAT.ID = uuid.New().String()

		if err := m.repository.SavePAT(newPAT); err != nil {
			return m, func() tea.Msg {
				return ErrorMsg{err: err}
			}
		}

		m.patsView.ExitAddMode()
		m.statusBar.SetMessage("PAT added successfully", false)
		return m, m.loadPATs()
	}

	if m.patsView.GetSelectedPAT() != nil {
		pat := m.patsView.GetSelectedPAT()
		if err := m.repository.SetActivePAT(pat.ID); err != nil {
			return m, func() tea.Msg {
				return ErrorMsg{err: err}
			}
		}
		m.provider = m.createProvider(*pat)
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
		m.state = ViewPATs
		m.topBar.SetContext("", "")
		m.topBar.SetStats(0, 0)
		m.topBar.SetPRBreakdown(0, 0, 0)
		m.topBar.SetView("PATs")
		m.updateShortcuts()
		return m, nil
	case ViewPRInspect:
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
		return func() tea.Msg {
			return ErrorMsg{err: fmt.Errorf("no PR selected")}
		}
	}

	review.PRIdentifier = fmt.Sprintf("%s/%d", pr.Repository.FullName, pr.Number)

	return func() tea.Msg {
		if err := m.provider.SubmitReview(m.ctx, review); err != nil {
			return ErrorMsg{err: err}
		}
		return SuccessMsg{message: "Review submitted successfully"}
	}
}

func (m Model) createProvider(pat domain.PAT) domain.Provider {
	switch pat.Provider {
	case domain.ProviderGitHub:
		return github.NewProvider(pat.Token, pat.Username)
	default:
		return nil
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
	var shortcuts []string

	switch m.state {
	case ViewPATs:
		shortcuts = []string{
			"<enter> Select PAT",
			"<a> Add PAT",
			"<d> Delete PAT",
			"<:> Command",
			"<q> Quit",
		}
	case ViewPRList:
		shortcuts = []string{
			"<enter> Inspect PR",
			"<r> Refresh",
			"</> Filter",
			"<j/k> Navigate",
			"<q> Back",
			"<:> Command",
		}
	case ViewPRInspect:
		shortcuts = []string{
			"<n/p> Next/Prev File",
			"<c> Toggle Comments",
			"<a> Approve",
			"<r> Request Changes",
			"<enter> Add Comment",
			"<j/k> Scroll",
			"<q> Back",
		}
	}

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
