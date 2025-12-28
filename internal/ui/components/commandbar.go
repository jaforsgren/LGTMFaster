package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CommandBarModel struct {
	textInput textinput.Model
	width     int
	active    bool
}

func NewCommandBar() *CommandBarModel {
	ti := textinput.New()
	ti.Placeholder = "Enter command..."
	ti.CharLimit = 256
	ti.Width = 50

	return &CommandBarModel{
		textInput: ti,
		active:    false,
	}
}

func (m *CommandBarModel) SetWidth(width int) {
	m.width = width
	if width > 10 {
		m.textInput.Width = width - 10
	}
}

func (m *CommandBarModel) Activate() {
	m.active = true
	m.textInput.Focus()
	m.textInput.SetValue(":")
}

func (m *CommandBarModel) Deactivate() {
	m.active = false
	m.textInput.Blur()
	m.textInput.SetValue("")
}

func (m *CommandBarModel) IsActive() bool {
	return m.active
}

func (m *CommandBarModel) Value() string {
	return m.textInput.Value()
}

func (m *CommandBarModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return cmd
}

func (m *CommandBarModel) View() string {
	if !m.active {
		return ""
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F9FAFB")).
		Background(lipgloss.Color("#1F2937")).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Width(m.width)

	return style.Render(" " + m.textInput.View())
}
