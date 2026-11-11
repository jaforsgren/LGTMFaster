package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listFilterCharLimit = 100

type listFilter struct {
	input  textinput.Model
	active bool
	text   string
}

func newListFilter(placeholder string) listFilter {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = listFilterCharLimit
	return listFilter{input: ti}
}

func (f *listFilter) Activate() {
	f.active = true
	f.input.SetValue(f.text)
	f.input.Focus()
}

func (f *listFilter) Deactivate() {
	f.active = false
	f.input.Blur()
}

func (f *listFilter) Clear() {
	f.text = ""
	f.input.SetValue("")
	f.Deactivate()
}

func (f *listFilter) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	f.text = f.input.Value()
	return cmd
}

func (f *listFilter) IsActive() bool {
	return f.active
}

func (f *listFilter) Text() string {
	return f.text
}

func (f *listFilter) Matches(fields ...string) bool {
	if f.text == "" {
		return true
	}
	needle := strings.ToLower(f.text)
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), needle) {
			return true
		}
	}
	return false
}

func (f *listFilter) View() string {
	if !f.active {
		return ""
	}
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Bold(true)
	return labelStyle.Render("Filter: ") + f.input.View()
}
