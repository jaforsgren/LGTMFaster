package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type StatusBarModel struct {
	width   int
	message string
	isError bool
}

func NewStatusBar() *StatusBarModel {
	return &StatusBarModel{}
}

func (m *StatusBarModel) SetWidth(width int) {
	m.width = width
}

func (m *StatusBarModel) SetMessage(message string, isError bool) {
	m.message = message
	m.isError = isError
}

func (m *StatusBarModel) ClearMessage() {
	m.message = ""
	m.isError = false
}

func (m *StatusBarModel) View() string {
	content := " " + m.message

	if lipgloss.Width(content) > m.width {
		content = content[:m.width-3] + "..."
	} else if lipgloss.Width(content) < m.width {
		padding := m.width - lipgloss.Width(content)
		content += strings.Repeat(" ", padding)
	}

	bgColor := lipgloss.Color("#374151")
	if m.isError {
		bgColor = lipgloss.Color("#991B1B")
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F9FAFB")).
		Background(bgColor).
		Width(m.width)

	return style.Render(content)
}
