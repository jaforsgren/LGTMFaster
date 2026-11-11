package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	// StatusBarHeight is the message row and the help row.
	StatusBarHeight      = 2
	statusBarPaddingCols = 2
	statusTruncationTail = "…"
)

var (
	statusBarStyle     = lipgloss.NewStyle().Background(lipgloss.Color("#000000")).Padding(0, 1)
	statusMessageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	statusErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	statusHelpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
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

func (m *StatusBarModel) Message() string {
	return m.message
}

// View renders the message over the help hints in exactly StatusBarHeight rows; both are truncated
// because wrapping would push the layout past the terminal.
func (m *StatusBarModel) View(help string) string {
	maxWidth := max(m.width-statusBarPaddingCols, 0)

	messageStyle := statusMessageStyle
	if m.isError {
		messageStyle = statusErrorStyle
	}
	message := messageStyle.Render(ansi.Truncate(m.message, maxWidth, statusTruncationTail))
	hints := statusHelpStyle.Render(ansi.Truncate(help, maxWidth, statusTruncationTail))

	return statusBarStyle.Width(m.width).Render(message + "\n" + hints)
}
