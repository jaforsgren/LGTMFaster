package views

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

// PipelineTriggerAction is what a key press in the trigger form asks the app to do.
type PipelineTriggerAction int

const (
	PipelineTriggerActionNone PipelineTriggerAction = iota
	PipelineTriggerActionSubmit
	PipelineTriggerActionCancel
)

const (
	pipelineTriggerBranchField = 0
	pipelineTriggerLabelWidth  = 22
	pipelineTriggerInputInset  = pipelineTriggerLabelWidth + 8
	pipelineTriggerCharLimit   = 1000
	pipelineTriggerHelp        = "tab/shift+tab: field | ↑↓: pick branch | ←/→ or space: change choice | ctrl+s: run | esc: cancel"
	loadingParametersText      = "loading parameters..."
	noParametersText           = "This pipeline has no run-time parameters."
	booleanTrue                = "true"
	booleanFalse               = "false"
)

var (
	errTriggerNoBranch      = errors.New("choose a branch to run on")
	errTriggerUnknownBranch = errors.New("the branch does not exist")
	triggerChoiceStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

// triggerParameter is a parameter and the value entered for it. A parameter with fixed values (or
// a boolean) is chosen by cycling through them; any other is typed in.
type triggerParameter struct {
	parameter domain.PipelineParameter
	choices   []string
	choice    int
	input     textinput.Model
}

func newTriggerParameter(parameter domain.PipelineParameter) triggerParameter {
	field := triggerParameter{parameter: parameter, choices: parameter.Values}
	if parameter.Type == domain.PipelineParameterBoolean {
		field.choices = []string{booleanTrue, booleanFalse}
	}
	if len(field.choices) > 0 {
		field.choice = max(slices.Index(field.choices, parameter.Default), 0)
		return field
	}
	field.input = textinput.New()
	field.input.Prompt = ""
	field.input.CharLimit = pipelineTriggerCharLimit
	field.input.SetValue(parameter.Default)
	return field
}

func (f *triggerParameter) isChoice() bool {
	return len(f.choices) > 0
}

func (f *triggerParameter) value() string {
	if f.isChoice() {
		return f.choices[f.choice]
	}
	return strings.TrimSpace(f.input.Value())
}

func (f *triggerParameter) cycle(delta int) {
	f.choice = (f.choice + delta + len(f.choices)) % len(f.choices)
}

// PipelineTriggerViewModel is the form that starts a pipeline on a branch with its parameters.
type PipelineTriggerViewModel struct {
	pipeline   domain.Pipeline
	branch     branchPicker
	parameters []triggerParameter
	loaded     bool
	focus      int
	active     bool
	width      int
	height     int
}

func NewPipelineTriggerView() *PipelineTriggerViewModel {
	return &PipelineTriggerViewModel{branch: newBranchPicker()}
}

func (m *PipelineTriggerViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	inputWidth := max(width-pipelineTriggerInputInset, minPaneSize)
	m.branch.SetWidth(inputWidth)
	for i := range m.parameters {
		m.parameters[i].input.Width = inputWidth
	}
}

func (m *PipelineTriggerViewModel) Activate(pipeline domain.Pipeline) {
	m.pipeline = pipeline
	m.branch.Reset("")
	m.parameters = nil
	m.loaded = false
	m.active = true
	m.focusField(pipelineTriggerBranchField)
}

func (m *PipelineTriggerViewModel) Deactivate() {
	m.active = false
	m.branch.Blur()
}

func (m *PipelineTriggerViewModel) IsActive() bool {
	return m.active
}

func (m *PipelineTriggerViewModel) GetPipeline() domain.Pipeline {
	return m.pipeline
}

func (m *PipelineTriggerViewModel) SetOptions(options domain.PipelineTriggerOptions) {
	m.branch.SetBranches(options.Branches)
	m.parameters = make([]triggerParameter, 0, len(options.Parameters))
	for _, parameter := range options.Parameters {
		m.parameters = append(m.parameters, newTriggerParameter(parameter))
	}
	m.loaded = true
	m.SetSize(m.width, m.height)
}

// Trigger validates the form and returns the run it describes. Numbers are checked here, since
// the pipeline would only reject them once it is queued.
func (m *PipelineTriggerViewModel) Trigger() (domain.PipelineTrigger, error) {
	branch := m.branch.Value()
	switch {
	case branch == "":
		return domain.PipelineTrigger{}, errTriggerNoBranch
	case m.branch.IsUnknown(branch):
		return domain.PipelineTrigger{}, errTriggerUnknownBranch
	}

	parameters := make(map[string]string, len(m.parameters))
	for _, field := range m.parameters {
		value := field.value()
		if field.parameter.Type == domain.PipelineParameterNumber {
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				return domain.PipelineTrigger{}, fmt.Errorf("%s must be a number", field.parameter.Label())
			}
		}
		parameters[field.parameter.Name] = value
	}
	return domain.PipelineTrigger{Pipeline: m.pipeline.Identifier(), Branch: branch, Parameters: parameters}, nil
}

func (m *PipelineTriggerViewModel) fieldCount() int {
	return 1 + len(m.parameters)
}

// HandleKey moves between and edits the fields, and reports what the app should do.
func (m *PipelineTriggerViewModel) HandleKey(msg tea.KeyMsg) (PipelineTriggerAction, tea.Cmd) {
	switch msg.String() {
	case "ctrl+s":
		m.branch.AcceptHighlighted()
		return PipelineTriggerActionSubmit, nil
	case "esc":
		return PipelineTriggerActionCancel, nil
	case "tab":
		m.focusField((m.focus + 1) % m.fieldCount())
		return PipelineTriggerActionNone, nil
	case "shift+tab":
		m.focusField((m.focus + m.fieldCount() - 1) % m.fieldCount())
		return PipelineTriggerActionNone, nil
	}

	if m.focus == pipelineTriggerBranchField {
		accepted, cmd := m.branch.HandleKey(msg)
		if accepted {
			m.focusField(min(m.focus+1, m.fieldCount()-1))
		}
		return PipelineTriggerActionNone, cmd
	}

	field := &m.parameters[m.focus-1]
	switch {
	case msg.String() == "enter":
		m.focusField((m.focus + 1) % m.fieldCount())
		return PipelineTriggerActionNone, nil
	case field.isChoice():
		switch msg.String() {
		case "left", "up":
			field.cycle(-1)
		case "right", "down", " ":
			field.cycle(1)
		}
		return PipelineTriggerActionNone, nil
	default:
		var cmd tea.Cmd
		field.input, cmd = field.input.Update(msg)
		return PipelineTriggerActionNone, cmd
	}
}

func (m *PipelineTriggerViewModel) focusField(index int) {
	if m.focus == pipelineTriggerBranchField {
		m.branch.AcceptHighlighted()
	}
	m.focus = index
	m.branch.Blur()
	for i := range m.parameters {
		m.parameters[i].input.Blur()
	}
	if index == pipelineTriggerBranchField {
		m.branch.Focus()
		return
	}
	if field := &m.parameters[index-1]; !field.isChoice() {
		field.input.Focus()
	}
}

func (m *PipelineTriggerViewModel) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder
	b.WriteString(renderListTitle(pipelinesTitle, PipelineDisplayName(m.pipeline), "Run pipeline"))
	b.WriteString("\n\n")
	b.WriteString(m.label("Branch", m.focus == pipelineTriggerBranchField) + m.branch.View() + "\n")
	if m.focus == pipelineTriggerBranchField {
		b.WriteString(m.branch.Dropdown(pipelineTriggerLabelWidth))
	}
	b.WriteString("\n")

	switch {
	case !m.loaded:
		b.WriteString(listMutedStyle.Render(loadingParametersText) + "\n")
	case len(m.parameters) == 0:
		b.WriteString(listMutedStyle.Render(noParametersText) + "\n")
	default:
		for i := range m.parameters {
			b.WriteString(m.renderParameter(i) + "\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(listMutedStyle.Render(pipelineTriggerHelp))

	return newPRBoxStyle.Width(max(m.width-newPRBoxInset, minPaneSize)).Render(b.String())
}

func (m *PipelineTriggerViewModel) renderParameter(index int) string {
	field := &m.parameters[index]
	label := m.label(truncateString(field.parameter.Label(), pipelineTriggerLabelWidth-2), m.focus == index+1)
	if !field.isChoice() {
		return label + field.input.View()
	}
	return label + triggerChoiceStyle.Render(fmt.Sprintf("‹ %s ›", field.value())) +
		listMutedStyle.Render(fmt.Sprintf("  %d/%d", field.choice+1, len(field.choices)))
}

func (m *PipelineTriggerViewModel) label(text string, focused bool) string {
	return renderFormLabel(text, focused, pipelineTriggerLabelWidth)
}
