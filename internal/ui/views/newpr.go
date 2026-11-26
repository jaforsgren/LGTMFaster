package views

import (
	"errors"
	"fmt"
	"strings"

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
	newPRFieldWorkItems
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
	newPRLabelWidth      = 13
	newPRBoxChromeRows   = 4
	newPRFixedRows       = 15
	newPRMinTextareaRows = 3
	newPRBoxInset        = 4
	newPRInputInset      = newPRLabelWidth + 8
	newPRCharLimit       = 65535
	newPRTitleCharLimit  = 400
	draftLabel           = "[x] draft"
	readyForReviewLabel  = "[ ] ready for review"
	newPRHelp            = "tab/shift+tab: field | ↑↓: pick branch | space: draft | ctrl+g: editor | ctrl+s: create | esc: cancel"
	newPRTitleText       = "New Pull Request"
	newPRSourceFormat    = "%s : %s"
	newPRPushedAgoFormat = "  (pushed %s)"
	newPRWorkItemsHint   = "numbers to link, e.g. 1234, 5678"
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
	branch      domain.PushedBranch
	target      branchPicker
	title       textinput.Model
	workItems   textinput.Model
	description textarea.Model
	isDraft     bool
	focus       newPRField
	active      bool
	width       int
	height      int
}

func NewNewPRView() *NewPRViewModel {
	title := textinput.New()
	title.Prompt = ""
	title.CharLimit = newPRTitleCharLimit
	workItems := textinput.New()
	workItems.Prompt = ""
	workItems.Placeholder = newPRWorkItemsHint
	description := textarea.New()
	description.Placeholder = "Describe the change..."
	description.CharLimit = newPRCharLimit
	description.ShowLineNumbers = false

	return &NewPRViewModel{target: newBranchPicker(), title: title, workItems: workItems, description: description}
}

func (m *NewPRViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	inputWidth := max(width-newPRInputInset, minPaneSize)
	m.target.SetWidth(inputWidth)
	m.title.Width = inputWidth
	m.workItems.Width = inputWidth
	m.description.SetWidth(max(width-newPRBoxInset*2, minPaneSize))
	m.description.SetHeight(max(height-newPRFixedRows-branchDropdownRows, newPRMinTextareaRows))
}

// Activate opens the form for a pull request from branch, with a title made from its name.
func (m *NewPRViewModel) Activate(branch domain.PushedBranch) {
	m.branch = branch
	m.target.Reset(branch.Branch)
	m.title.SetValue(domain.TitleFromBranch(branch.Branch))
	m.workItems.SetValue("")
	m.description.SetValue("")
	m.isDraft = false
	m.active = true
	m.focusField(newPRFieldTitle)
}

func (m *NewPRViewModel) Deactivate() {
	m.active = false
	m.target.Blur()
	m.title.Blur()
	m.workItems.Blur()
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
	m.target.SetBranches(branches)
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
	target := m.target.Value()
	switch {
	case title == "":
		return domain.NewPullRequest{}, errNewPRNoTitle
	case target == "":
		return domain.NewPullRequest{}, errNewPRNoTarget
	case target == m.branch.Branch:
		return domain.NewPullRequest{}, errNewPRSameBranch
	case m.target.IsUnknown(target):
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

// Prefill fills in what the draft gives: the target branch, the description and the work items.
func (m *NewPRViewModel) Prefill(draft domain.NewPullRequestDraft) {
	if draft.TargetBranch != "" {
		m.target.SetValue(draft.TargetBranch)
	}
	m.description.SetValue(draft.Description)
	m.workItems.SetValue(domain.FormatWorkItemIDs(draft.WorkItemIDs))
}

// WorkItemIDs are the work items to link once the pull request is created.
func (m *NewPRViewModel) WorkItemIDs() ([]int, error) {
	return domain.ParseWorkItemIDs(m.workItems.Value())
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
	case newPRFieldWorkItems:
		if msg.String() == "enter" {
			m.focusField(newPRFieldDescription)
			return NewPRActionNone, nil
		}
		var cmd tea.Cmd
		m.workItems, cmd = m.workItems.Update(msg)
		return NewPRActionNone, cmd
	default:
		var cmd tea.Cmd
		m.description, cmd = m.description.Update(msg)
		return NewPRActionNone, cmd
	}
}

func (m *NewPRViewModel) handleTargetKey(msg tea.KeyMsg) tea.Cmd {
	accepted, cmd := m.target.HandleKey(msg)
	if accepted {
		m.focusField(newPRFieldTitle)
	}
	return cmd
}

func (m *NewPRViewModel) acceptHighlightedTarget() {
	if m.focus == newPRFieldTarget {
		m.target.AcceptHighlighted()
	}
}

func (m *NewPRViewModel) focusField(field newPRField) {
	m.focus = field
	m.target.Blur()
	m.title.Blur()
	m.workItems.Blur()
	m.description.Blur()
	switch field {
	case newPRFieldTarget:
		m.target.Focus()
	case newPRFieldTitle:
		m.title.Focus()
	case newPRFieldWorkItems:
		m.workItems.Focus()
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
	b.WriteString(m.label("Status", m.focus == newPRFieldDraft) + m.renderDraft() + "\n")
	b.WriteString(m.label("Work items", m.focus == newPRFieldWorkItems) + m.workItems.View() + "\n\n")
	b.WriteString(m.label("Description", m.focus == newPRFieldDescription) + "\n")
	b.WriteString(m.description.View())
	b.WriteString("\n\n")
	b.WriteString(listMutedStyle.Render(newPRHelp))

	return newPRBoxStyle.Width(max(m.width-newPRBoxInset, minPaneSize)).Render(b.String())
}

func (m *NewPRViewModel) label(text string, focused bool) string {
	return renderFormLabel(text, focused, newPRLabelWidth)
}

// renderFormLabel is a form field's label, highlighted while the field has focus.
func renderFormLabel(text string, focused bool, width int) string {
	style := newPRLabelStyle
	if focused {
		style = newPRFocusedLabelStyle
	}
	return style.Render(padToWidth(text+":", width))
}

func (m *NewPRViewModel) renderDraft() string {
	if m.isDraft {
		return newPRDraftStyle.Render(draftLabel)
	}
	return detailValueStyle.Render(readyForReviewLabel)
}

func (m *NewPRViewModel) renderDropdown() string {
	if m.focus != newPRFieldTarget {
		return ""
	}
	return m.target.Dropdown(newPRLabelWidth)
}
