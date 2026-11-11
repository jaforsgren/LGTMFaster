package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	stageApprovalCommentCharLimit = 1000
	approveColor                  = lipgloss.Color("#10B981")
	rejectColor                   = lipgloss.Color("#EF4444")
)

type StageApprovalViewModel struct {
	input    textinput.Model
	project  string
	stage    domain.PipelineStage
	decision domain.ApprovalDecision
	active   bool
	width    int
	height   int
}

func NewStageApprovalView() *StageApprovalViewModel {
	ti := textinput.New()
	ti.Placeholder = "Optional comment..."
	ti.CharLimit = stageApprovalCommentCharLimit

	return &StageApprovalViewModel{input: ti}
}

func (m *StageApprovalViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.input.Width = max(1, width-12)
}

func (m *StageApprovalViewModel) Activate(project string, stage domain.PipelineStage, decision domain.ApprovalDecision) {
	m.project = project
	m.stage = stage
	m.decision = decision
	m.active = true
	m.input.SetValue("")
	m.input.Focus()
}

func (m *StageApprovalViewModel) Deactivate() {
	m.active = false
	m.input.Blur()
	m.input.SetValue("")
}

func (m *StageApprovalViewModel) IsActive() bool {
	return m.active
}

func (m *StageApprovalViewModel) GetStage() domain.PipelineStage {
	return m.stage
}

func (m *StageApprovalViewModel) GetApproval() domain.StageApproval {
	return domain.StageApproval{
		Project:    m.project,
		ApprovalID: m.stage.PendingApprovalID,
		Decision:   m.decision,
		Comment:    strings.TrimSpace(m.input.Value()),
	}
}

func (m *StageApprovalViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

func (m *StageApprovalViewModel) View() string {
	if !m.active {
		return ""
	}

	title := "Approve stage: " + m.stage.Name
	color := approveColor
	if m.decision == domain.ApprovalDecisionReject {
		title = "Reject stage: " + m.stage.Name
		color = rejectColor
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(color).Bold(true).Padding(1, 0).Render(title))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(listMutedStyle.Render("Enter: Submit | Esc: Cancel"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(1, 2).
		Width(max(1, m.width-4))

	return boxStyle.Render(b.String())
}
