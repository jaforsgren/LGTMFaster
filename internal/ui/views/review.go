package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type ReviewMode int

const (
	ReviewModeComment ReviewMode = iota
	ReviewModeApprove
	ReviewModeRequestChanges
)

type ReviewViewModel struct {
	mode     ReviewMode
	textarea textarea.Model
	width    int
	height   int
	active   bool
}

func NewReviewView() *ReviewViewModel {
	ta := textarea.New()
	ta.Placeholder = "Enter your comment or review..."
	ta.CharLimit = 10000
	ta.ShowLineNumbers = false

	return &ReviewViewModel{
		mode:     ReviewModeComment,
		textarea: ta,
		active:   false,
	}
}

func (m *ReviewViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textarea.SetWidth(width - 4)
	m.textarea.SetHeight(height - 12)
}

func (m *ReviewViewModel) Activate(mode ReviewMode) {
	m.active = true
	m.mode = mode
	m.textarea.Focus()
	m.textarea.SetValue("")
}

func (m *ReviewViewModel) Deactivate() {
	m.active = false
	m.textarea.Blur()
	m.textarea.SetValue("")
}

func (m *ReviewViewModel) IsActive() bool {
	return m.active
}

func (m *ReviewViewModel) GetReview() domain.Review {
	action := domain.ReviewActionComment
	switch m.mode {
	case ReviewModeApprove:
		action = domain.ReviewActionApprove
	case ReviewModeRequestChanges:
		action = domain.ReviewActionRequestChanges
	}

	return domain.Review{
		Action:   action,
		Body:     m.textarea.Value(),
		Comments: []domain.Comment{},
	}
}

func (m *ReviewViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return cmd
}

func (m *ReviewViewModel) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	title := ""
	switch m.mode {
	case ReviewModeApprove:
		title = "Approve Pull Request"
	case ReviewModeRequestChanges:
		title = "Request Changes"
	default:
		title = "Add Comment"
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Padding(1, 0)

	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")
	b.WriteString(m.textarea.View())
	b.WriteString("\n\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)

	help := "Ctrl+S: Submit | Esc: Cancel"
	b.WriteString(helpStyle.Render(help))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(1, 2).
		Width(m.width - 4)

	return boxStyle.Render(b.String())
}
