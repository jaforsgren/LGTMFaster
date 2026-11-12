package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	searchFieldCharLimit = 200
	searchFieldLabel     = "Search: "
	searchMatchCountFmt  = " %d/%d"
)

var (
	searchLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true)
	searchMatchStyle = lipgloss.NewStyle().Background(lipgloss.Color("236"))
)

// searchField is the vim-style "/" search of a text view: typed while active, kept afterwards so
// n/N can move between its matches.
type searchField struct {
	input  textinput.Model
	active bool
}

func newSearchField(placeholder string) searchField {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.CharLimit = searchFieldCharLimit
	return searchField{input: input}
}

func (f *searchField) Activate() {
	f.active = true
	f.input.Focus()
}

func (f *searchField) Deactivate() {
	f.active = false
	f.input.Blur()
}

func (f *searchField) Clear() {
	f.input.SetValue("")
	f.Deactivate()
}

func (f *searchField) IsActive() bool {
	return f.active
}

func (f *searchField) Query() string {
	return strings.TrimSpace(f.input.Value())
}

func (f *searchField) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return cmd
}

// Matches reports a case-insensitive match of the query; an empty query matches nothing.
func (f *searchField) Matches(text string) bool {
	query := f.Query()
	return query != "" && containsFold(text, query)
}

// View is the input while typing, else the kept query with the match position, else nothing.
func (f *searchField) View(position, count int) string {
	if f.active {
		return searchLabelStyle.Render(searchFieldLabel) + f.input.View()
	}
	query := f.Query()
	if query == "" {
		return ""
	}
	return searchLabelStyle.Render(searchFieldLabel) + listRowStyle.Render(query) +
		listMutedStyle.Render(fmt.Sprintf(searchMatchCountFmt, position, count))
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
