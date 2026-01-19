package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DescriptionEditViewModel struct {
	textarea textarea.Model
	width    int
	height   int
	active   bool
}

func NewDescriptionEditView() *DescriptionEditViewModel {
	ta := textarea.New()
	ta.Placeholder = "Enter PR description..."
	ta.CharLimit = 65535
	ta.ShowLineNumbers = false

	return &DescriptionEditViewModel{
		textarea: ta,
		active:   false,
	}
}

func (m *DescriptionEditViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textarea.SetWidth(width - 4)
	m.textarea.SetHeight(height - 12)
}

func (m *DescriptionEditViewModel) Activate(currentDescription string) {
	m.active = true
	m.textarea.Focus()
	m.textarea.SetValue(currentDescription)
}

func (m *DescriptionEditViewModel) Deactivate() {
	m.active = false
	m.textarea.Blur()
	m.textarea.SetValue("")
}

func (m *DescriptionEditViewModel) IsActive() bool {
	return m.active
}

func (m *DescriptionEditViewModel) GetDescription() string {
	return m.textarea.Value()
}

func (m *DescriptionEditViewModel) GetValue() string {
	return m.textarea.Value()
}

func (m *DescriptionEditViewModel) SetValue(value string) {
	m.textarea.SetValue(value)
}

func (m *DescriptionEditViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return cmd
}

func (m *DescriptionEditViewModel) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Padding(1, 0)

	b.WriteString(titleStyle.Render("Edit PR Description"))
	b.WriteString("\n\n")
	b.WriteString(m.textarea.View())
	b.WriteString("\n\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)

	help := "Ctrl+S: Save | Ctrl+G: Open in editor | Esc: Cancel"
	b.WriteString(helpStyle.Render(help))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(1, 2).
		Width(m.width - 4)

	return boxStyle.Render(b.String())
}
