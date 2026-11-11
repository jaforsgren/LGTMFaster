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
	suggestion string
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
	m.suggestion = ""
	m.textInput.Focus()
	m.textInput.SetValue(commandPrefix)
}

func (m *CommandBarModel) Deactivate() {
	m.active = false
	m.suggestion = ""
	m.textInput.Blur()
	m.textInput.SetValue("")
}

func (m *CommandBarModel) IsActive() bool {
	return m.active
}

func (m *CommandBarModel) Value() string {
	return m.textInput.Value()
}

// CommandName is the command word typed so far, without the prefix or arguments.
func (m *CommandBarModel) CommandName() string {
	fields := strings.Fields(strings.TrimPrefix(m.textInput.Value(), commandPrefix))
	if len(fields) != 1 || strings.HasSuffix(m.textInput.Value(), " ") {
		return ""
	}
	return fields[0]
}

// SetSuggestion sets the command the typed name completes to; it is shown faded after the input.
func (m *CommandBarModel) SetSuggestion(suggestion string) {
	m.suggestion = suggestion
}

// AcceptSuggestion completes the typed name to the suggestion and reports whether there was one.
func (m *CommandBarModel) AcceptSuggestion() bool {
	if m.remainingSuggestion() == "" {
		return false
	}
	m.textInput.SetValue(commandPrefix + m.suggestion + " ")
	m.textInput.CursorEnd()
	m.suggestion = ""
	return true
}

func (m *CommandBarModel) remainingSuggestion() string {
	name := m.CommandName()
	if name == "" || !strings.HasPrefix(m.suggestion, name) {
		return ""
	}
	return m.suggestion[len(name):]
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
	return commandBarStyle.Width(m.width).Render(m.textInput.View() + commandSuggestionStyle.Render(m.remainingSuggestion()))
}
