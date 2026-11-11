package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	// CommandBarHeight is reserved even while the bar is inactive, so the layout never jumps.
	CommandBarHeight    = 1
	commandPrefix       = ":"
	commandCharLimit    = 256
	commandInputInset   = 10
	commandBarMinWidth  = 10
	commandBarPaddingLR = 2
)

var (
	commandBarStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Background(lipgloss.Color("#000000")).Padding(0, commandBarPaddingLR)
	commandSuggestionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

type CommandBarModel struct {
	textInput  textinput.Model
	width      int
	active     bool
	completion string
}

func NewCommandBar() *CommandBarModel {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "Enter command..."
	ti.CharLimit = commandCharLimit

	return &CommandBarModel{textInput: ti}
}

func (m *CommandBarModel) SetWidth(width int) {
	m.width = width
	if width > commandBarMinWidth {
		m.textInput.Width = width - commandInputInset
	}
}

func (m *CommandBarModel) Activate() {
	m.active = true
	m.completion = ""
	m.textInput.Focus()
	m.textInput.SetValue(commandPrefix)
}

func (m *CommandBarModel) Deactivate() {
	m.active = false
	m.completion = ""
	m.textInput.Blur()
	m.textInput.SetValue("")
}

func (m *CommandBarModel) IsActive() bool {
	return m.active
}

func (m *CommandBarModel) Value() string {
	return m.textInput.Value()
}

// Input is the typed text without the command prefix.
func (m *CommandBarModel) Input() string {
	return strings.TrimPrefix(m.textInput.Value(), commandPrefix)
}

// SetInput replaces the typed text, keeping the cursor at its end.
func (m *CommandBarModel) SetInput(input string) {
	m.textInput.SetValue(commandPrefix + input)
	m.textInput.CursorEnd()
}

// SetCompletion sets the whole input the typed text completes to; the part still to type is shown
// faded after the input.
func (m *CommandBarModel) SetCompletion(completion string) {
	m.completion = completion
}

// AcceptCompletion replaces the input with the completion and reports whether there was one.
func (m *CommandBarModel) AcceptCompletion() bool {
	if m.completion == "" || m.completion == m.Input() {
		return false
	}
	m.SetInput(m.completion)
	m.completion = ""
	return true
}

func (m *CommandBarModel) remainingCompletion() string {
	input := m.Input()
	if !strings.HasPrefix(m.completion, input) {
		return ""
	}
	return m.completion[len(input):]
}

func (m *CommandBarModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return cmd
}

func (m *CommandBarModel) View() string {
	if !m.active {
		return commandBarStyle.Width(m.width).Render("")
	}
	return commandBarStyle.Width(m.width).Render(m.textInput.View() + commandSuggestionStyle.Render(m.remainingCompletion()))
}
