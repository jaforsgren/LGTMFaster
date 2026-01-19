package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type InlineCommentViewModel struct {
	textarea textarea.Model
	width    int
	height   int
	active   bool
	lineInfo string
}

func NewInlineCommentView() *InlineCommentViewModel {
	ta := textarea.New()
	ta.Placeholder = "Enter your inline comment..."
	ta.CharLimit = 10000
	ta.ShowLineNumbers = false

	return &InlineCommentViewModel{
		textarea: ta,
		active:   false,
	}
}

func (m *InlineCommentViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textarea.SetWidth(width - 4)
	m.textarea.SetHeight(8)
}

func (m *InlineCommentViewModel) Activate(lineInfo string) {
	m.active = true
	m.lineInfo = lineInfo
	m.textarea.Focus()
	m.textarea.SetValue("")
}

func (m *InlineCommentViewModel) Deactivate() {
	m.active = false
	m.textarea.Blur()
	m.textarea.SetValue("")
}

func (m *InlineCommentViewModel) IsActive() bool {
	return m.active
}

func (m *InlineCommentViewModel) GetComment() string {
	return m.textarea.Value()
}

func (m *InlineCommentViewModel) GetValue() string {
	return m.textarea.Value()
}

func (m *InlineCommentViewModel) SetValue(value string) {
	m.textarea.SetValue(value)
}

func (m *InlineCommentViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return cmd
}

func (m *InlineCommentViewModel) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Padding(1, 0)

	title := "Add Inline Comment"
	if m.lineInfo != "" {
		title += " - " + m.lineInfo
	}

	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")
	b.WriteString(m.textarea.View())
	b.WriteString("\n\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)

	help := "Ctrl+S: Add Comment | Ctrl+G: Open in editor | Esc: Cancel"
	b.WriteString(helpStyle.Render(help))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(1, 2).
		Width(m.width - 4)

	return boxStyle.Render(b.String())
}
