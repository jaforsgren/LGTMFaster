package views

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type newPRField int

const (
	newPRFieldTarget newPRField = iota
	newPRFieldTitle
	newPRFieldDraft
	newPRFieldDescription
	newPRFieldCount
)

// NewPRAction is what a key press in the new pull request form asks the app to do.
type NewPRAction int

const (
	NewPRActionNone NewPRAction = iota
	NewPRActionSubmit
	NewPRActionCancel
	NewPRActionOpenEditor
)

const (
	newPRDropdownRows      = 6
	newPRLabelWidth        = 13
	newPRBoxChromeRows     = 4
	newPRFixedRows         = 14
	newPRMinTextareaRows   = 3
	newPRBoxInset          = 4
	newPRInputInset        = newPRLabelWidth + 8
	newPRCharLimit         = 65535
	newPRTitleCharLimit    = 400
	branchPathSeparator    = "/"
	draftLabel             = "[x] draft"
	readyForReviewLabel    = "[ ] ready for review"
	newPRHelp              = "tab/shift+tab: field | ↑↓: pick branch | space: draft | ctrl+g: editor | ctrl+s: create | esc: cancel"
	loadingBranchesText    = "loading branches..."
	branchWordSeparators   = "-_"
	newPRTitleText         = "New Pull Request"
	newPRSourceFormat      = "%s : %s"
	newPRPushedAgoFormat   = "  (pushed %s)"
	noMatchingBranchesText = "no matching branch"
)

var (
	errNewPRNoTitle        = errors.New("a pull request needs a title")
	errNewPRNoTarget       = errors.New("choose a branch to merge into")
	errNewPRSameBranch     = errors.New("the target branch is the source branch")
	errNewPRUnknownTarget  = errors.New("the target branch does not exist")
	newPRBoxStyle          = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(mainWindowBorderColor).Padding(1, 2)
	newPRFocusedLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	newPRLabelStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	newPRDraftStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

type NewPRViewModel struct {
	branch         domain.PushedBranch
	branches       domain.RepositoryBranches
	branchesLoaded bool
	target         textinput.Model
	title          textinput.Model
	description    textarea.Model
	isDraft        bool
	focus          newPRField
	highlighted    int
	active         bool
	width          int
	height         int
}

func NewNewPRView() *NewPRViewModel {
	target := textinput.New()
	target.Prompt = ""
	target.Placeholder = loadingBranchesText
	title := textinput.New()
	title.Prompt = ""
	title.CharLimit = newPRTitleCharLimit
	description := textarea.New()
	description.Placeholder = "Describe the change..."
	description.CharLimit = newPRCharLimit
	description.ShowLineNumbers = false

	return &NewPRViewModel{target: target, title: title, description: description}
}

func (m *NewPRViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	inputWidth := max(width-newPRInputInset, minPaneSize)
	m.target.Width = inputWidth
	m.title.Width = inputWidth
	m.description.SetWidth(max(width-newPRBoxInset*2, minPaneSize))
	m.description.SetHeight(max(height-newPRFixedRows-newPRDropdownRows, newPRMinTextareaRows))
}

// Activate opens the form for a pull request from branch, with a title made from its name.
func (m *NewPRViewModel) Activate(branch domain.PushedBranch) {
	m.branch = branch
	m.branches = domain.RepositoryBranches{}
	m.branchesLoaded = false
	m.target.SetValue("")
	m.title.SetValue(TitleFromBranch(branch.Branch))
	m.description.SetValue("")
	m.isDraft = false
	m.highlighted = 0
	m.active = true
	m.focusField(newPRFieldTitle)
}

func (m *NewPRViewModel) Deactivate() {
	m.active = false
	m.target.Blur()
	m.title.Blur()
	m.description.Blur()
}

func (m *NewPRViewModel) IsActive() bool {
	return m.active
}

func (m *NewPRViewModel) Branch() domain.PushedBranch {
	return m.branch
}

// SetBranches fills the target dropdown, preselecting the default branch unless one was typed.
func (m *NewPRViewModel) SetBranches(branches domain.RepositoryBranches) {
	m.branches = branches
	m.branchesLoaded = true
	m.target.Placeholder = ""
	if m.target.Value() == "" {
		m.target.SetValue(branches.Default)
	}
}

func (m *NewPRViewModel) Description() string {
	return m.description.Value()
}

func (m *NewPRViewModel) SetDescription(description string) {
	m.description.SetValue(description)
}

// Request validates the form and returns the pull request it describes.
func (m *NewPRViewModel) Request() (domain.NewPullRequest, error) {
	title := strings.TrimSpace(m.title.Value())
	target := strings.TrimSpace(m.target.Value())
	switch {
	case title == "":
		return domain.NewPullRequest{}, errNewPRNoTitle
	case target == "":
		return domain.NewPullRequest{}, errNewPRNoTarget
	case target == m.branch.Branch:
		return domain.NewPullRequest{}, errNewPRSameBranch
	case m.branchesLoaded && !slices.Contains(m.branches.Names, target):
		return domain.NewPullRequest{}, errNewPRUnknownTarget
	}
	return domain.NewPullRequest{
		RepositoryFullName: m.branch.Repository.FullName,
		SourceBranch:       m.branch.Branch,
		TargetBranch:       target,
		Title:              title,
		Description:        m.description.Value(),
		IsDraft:            m.isDraft,
	}, nil
}

// HandleKey moves between and edits the fields, and reports what the app should do.
func (m *NewPRViewModel) HandleKey(msg tea.KeyMsg) (NewPRAction, tea.Cmd) {
	switch msg.String() {
	case "ctrl+s":
		return NewPRActionSubmit, nil
	case "esc":
		return NewPRActionCancel, nil
	case "ctrl+g":
		return NewPRActionOpenEditor, nil
	case "tab":
		m.acceptHighlightedTarget()
		m.focusField((m.focus + 1) % newPRFieldCount)
		return NewPRActionNone, nil
	case "shift+tab":
		m.acceptHighlightedTarget()
		m.focusField((m.focus + newPRFieldCount - 1) % newPRFieldCount)
		return NewPRActionNone, nil
	}

	switch m.focus {
	case newPRFieldTarget:
		return NewPRActionNone, m.handleTargetKey(msg)
	case newPRFieldTitle:
		if msg.String() == "enter" {
			m.focusField(newPRFieldDraft)
			return NewPRActionNone, nil
		}
		var cmd tea.Cmd
		m.title, cmd = m.title.Update(msg)
		return NewPRActionNone, cmd
	case newPRFieldDraft:
		if msg.String() == " " || msg.String() == "enter" {
			m.isDraft = !m.isDraft
		}
		return NewPRActionNone, nil
	default:
		var cmd tea.Cmd
		m.description, cmd = m.description.Update(msg)
		return NewPRActionNone, cmd
	}
}

func (m *NewPRViewModel) handleTargetKey(msg tea.KeyMsg) tea.Cmd {
	matches := m.targetMatches()
	switch msg.String() {
	case "up":
		m.highlighted = max(m.highlighted-1, 0)
		return nil
	case "down":
		m.highlighted = min(m.highlighted+1, max(len(matches)-1, 0))
		return nil
	case "enter":
		m.acceptHighlightedTarget()
		m.focusField(newPRFieldTitle)
		return nil
	}
	var cmd tea.Cmd
	m.target, cmd = m.target.Update(msg)
	m.highlighted = 0
	return cmd
}

// acceptHighlightedTarget completes a partly typed target to the highlighted dropdown entry.
func (m *NewPRViewModel) acceptHighlightedTarget() {
	if m.focus != newPRFieldTarget {
		return
	}
	matches := m.targetMatches()
	if len(matches) == 0 || slices.Contains(matches, m.target.Value()) && m.highlighted == 0 {
		return
	}
	m.target.SetValue(matches[min(m.highlighted, len(matches)-1)])
	m.target.CursorEnd()
}

// targetMatches are the branches containing the typed text, prefix matches first, without the
// source branch.
func (m *NewPRViewModel) targetMatches() []string {
	query := strings.ToLower(strings.TrimSpace(m.target.Value()))
	var prefixed, contained []string
	for _, name := range m.branches.Names {
		lower := strings.ToLower(name)
		switch {
		case name == m.branch.Branch:
		case strings.HasPrefix(lower, query):
			prefixed = append(prefixed, name)
		case strings.Contains(lower, query):
			contained = append(contained, name)
		}
	}
	return append(prefixed, contained...)
}

func (m *NewPRViewModel) focusField(field newPRField) {
	m.focus = field
	m.target.Blur()
	m.title.Blur()
	m.description.Blur()
	switch field {
	case newPRFieldTarget:
		m.target.Focus()
	case newPRFieldTitle:
		m.title.Focus()
	case newPRFieldDescription:
		m.description.Focus()
	}
}

func (m *NewPRViewModel) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder
	b.WriteString(listTitleStyle.Render(newPRTitleText))
	b.WriteString("\n\n")

	source := fmt.Sprintf(newPRSourceFormat, m.branch.Repository.FullName, m.branch.Branch)
	if !m.branch.PushedAt.IsZero() {
		source += listMutedStyle.Render(fmt.Sprintf(newPRPushedAgoFormat, FormatAge(m.branch.PushedAt)))
	}
	b.WriteString(m.label("From", false) + detailValueStyle.Render(source) + "\n")
	b.WriteString(m.label("Into", m.focus == newPRFieldTarget) + m.target.View() + "\n")
	b.WriteString(m.renderDropdown())
	b.WriteString(m.label("Title", m.focus == newPRFieldTitle) + m.title.View() + "\n")
	b.WriteString(m.label("Status", m.focus == newPRFieldDraft) + m.renderDraft() + "\n\n")
	b.WriteString(m.label("Description", m.focus == newPRFieldDescription) + "\n")
	b.WriteString(m.description.View())
	b.WriteString("\n\n")
	b.WriteString(listMutedStyle.Render(newPRHelp))

	return newPRBoxStyle.Width(max(m.width-newPRBoxInset, minPaneSize)).Render(b.String())
}

func (m *NewPRViewModel) label(text string, focused bool) string {
	style := newPRLabelStyle
	if focused {
		style = newPRFocusedLabelStyle
	}
	return style.Render(padToWidth(text+":", newPRLabelWidth))
}

func (m *NewPRViewModel) renderDraft() string {
	if m.isDraft {
		return newPRDraftStyle.Render(draftLabel)
	}
	return detailValueStyle.Render(readyForReviewLabel)
}

// renderDropdown lists the matching target branches under the field while it has focus, always
// taking the same rows then so the form does not jump while typing.
func (m *NewPRViewModel) renderDropdown() string {
	if m.focus != newPRFieldTarget {
		return ""
	}
	rows := make([]string, newPRDropdownRows)
	if m.branchesLoaded {
		matches := m.targetMatches()
		if len(matches) == 0 {
			rows[0] = listMutedStyle.Render(noMatchingBranchesText)
		}
		start := max(0, m.highlighted-newPRDropdownRows+1)
		for i := 0; i < newPRDropdownRows && start+i < len(matches); i++ {
			marker, style := unselectedRowMarker, listRowStyle
			if start+i == m.highlighted {
				marker, style = selectedRowMarker, listSelectedRowStyle
			}
			rows[i] = style.Render(marker + matches[start+i])
		}
	}

	indent := strings.Repeat(" ", newPRLabelWidth)
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(indent + row + "\n")
	}
	return b.String()
}

// TitleFromBranch turns the last segment of a branch name into a sentence: "feature/add-login"
// becomes "Add login".
func TitleFromBranch(branch string) string {
	segments := strings.Split(branch, branchPathSeparator)
	words := strings.FieldsFunc(segments[len(segments)-1], func(r rune) bool {
		return strings.ContainsRune(branchWordSeparators, r)
	})
	title := strings.Join(words, " ")
	if title == "" {
		return ""
	}
	runes := []rune(title)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
