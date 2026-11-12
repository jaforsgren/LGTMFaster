package views

import (
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	branchDropdownRows     = 6
	loadingBranchesText    = "loading branches..."
	noMatchingBranchesText = "no matching branch"
)

// branchPicker is a text field that completes to one of a repository's branches, listing the
// matches in a dropdown under it.
type branchPicker struct {
	input       textinput.Model
	branches    domain.RepositoryBranches
	loaded      bool
	excluded    string
	highlighted int
	// replaceOnType makes the first character typed replace the preselected branch instead of
	// being appended to it.
	replaceOnType bool
}

func newBranchPicker() branchPicker {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = loadingBranchesText
	return branchPicker{input: input}
}

// Reset empties the picker until branches are set; excluded is left out of the matches.
func (p *branchPicker) Reset(excluded string) {
	p.branches = domain.RepositoryBranches{}
	p.loaded = false
	p.excluded = excluded
	p.highlighted = 0
	p.input.Placeholder = loadingBranchesText
	p.input.SetValue("")
}

// SetBranches fills the dropdown, preselecting the default branch unless one was typed. Typing
// replaces the preselected branch even when the field had focus before the branches arrived.
func (p *branchPicker) SetBranches(branches domain.RepositoryBranches) {
	p.branches = branches
	p.loaded = true
	p.input.Placeholder = ""
	if p.input.Value() == "" && branches.Default != "" {
		p.input.SetValue(branches.Default)
		p.replaceOnType = p.input.Focused()
	}
}

func (p *branchPicker) SetWidth(width int) {
	p.input.Width = width
}

func (p *branchPicker) Focus() {
	p.input.Focus()
	p.replaceOnType = p.input.Value() != ""
}

func (p *branchPicker) Blur() {
	p.input.Blur()
}

func (p *branchPicker) Value() string {
	return strings.TrimSpace(p.input.Value())
}

func (p *branchPicker) SetValue(value string) {
	p.input.SetValue(value)
}

// IsUnknown reports a branch that is not one of the loaded branches.
func (p *branchPicker) IsUnknown(branch string) bool {
	return p.loaded && !slices.Contains(p.branches.Names, branch)
}

// HandleKey moves through the dropdown or edits the text, and reports enter, which accepts the
// highlighted branch.
func (p *branchPicker) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	matches := p.Matches()
	switch msg.String() {
	case "up":
		p.highlighted = max(p.highlighted-1, 0)
		return false, nil
	case "down":
		p.highlighted = min(p.highlighted+1, max(len(matches)-1, 0))
		return false, nil
	case "enter":
		p.AcceptHighlighted()
		return true, nil
	}
	if p.replaceOnType && msg.Type == tea.KeyRunes {
		p.input.SetValue("")
	}
	p.replaceOnType = false
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	p.highlighted = 0
	return false, cmd
}

// AcceptHighlighted completes a partly typed branch to the highlighted dropdown entry.
func (p *branchPicker) AcceptHighlighted() {
	matches := p.Matches()
	if len(matches) == 0 || slices.Contains(matches, p.input.Value()) && p.highlighted == 0 {
		return
	}
	p.input.SetValue(matches[min(p.highlighted, len(matches)-1)])
	p.input.CursorEnd()
}

// Matches are the branches containing the typed text, prefix matches first, without the excluded
// branch.
func (p *branchPicker) Matches() []string {
	query := strings.ToLower(strings.TrimSpace(p.input.Value()))
	var prefixed, contained []string
	for _, name := range p.branches.Names {
		lower := strings.ToLower(name)
		switch {
		case name == p.excluded:
		case strings.HasPrefix(lower, query):
			prefixed = append(prefixed, name)
		case strings.Contains(lower, query):
			contained = append(contained, name)
		}
	}
	return append(prefixed, contained...)
}

func (p *branchPicker) View() string {
	return p.input.View()
}

// Dropdown lists the matching branches indented under the field, always taking the same rows so
// the form does not jump while typing.
func (p *branchPicker) Dropdown(indentWidth int) string {
	rows := make([]string, branchDropdownRows)
	if p.loaded {
		matches := p.Matches()
		if len(matches) == 0 {
			rows[0] = listMutedStyle.Render(noMatchingBranchesText)
		}
		start := max(0, p.highlighted-branchDropdownRows+1)
		for i := 0; i < branchDropdownRows && start+i < len(matches); i++ {
			marker, style := unselectedRowMarker, listRowStyle
			if start+i == p.highlighted {
				marker, style = selectedRowMarker, listSelectedRowStyle
			}
			rows[i] = style.Render(marker + matches[start+i])
		}
	}

	indent := strings.Repeat(" ", indentWidth)
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(indent + row + "\n")
	}
	return b.String()
}
