package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type PATItem struct {
	pat domain.PAT
}

func (i PATItem) FilterValue() string { return i.pat.Name }
func (i PATItem) Title() string {
	indicator := " "
	if i.pat.IsActive {
		indicator = "●"
	}
	return fmt.Sprintf("%s %s (%s)", indicator, i.pat.Name, i.pat.Provider)
}
func (i PATItem) Description() string { return i.pat.Username }

type PATMode int

const (
	PATModeList PATMode = iota
	PATModeAdd
)

type PATsViewModel struct {
	list          list.Model
	Mode          PATMode
	nameInput     textinput.Model
	tokenInput    textinput.Model
	providerInput textinput.Model
	usernameInput textinput.Model
	inputFocus    int
	width         int
	height        int
}

func NewPATsView() *PATsViewModel {
	items := []list.Item{}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Personal Access Tokens"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	nameInput := textinput.New()
	nameInput.Placeholder = "PAT Name"
	nameInput.CharLimit = 50

	tokenInput := textinput.New()
	tokenInput.Placeholder = "Token"
	tokenInput.CharLimit = 256
	tokenInput.EchoMode = textinput.EchoPassword

	providerInput := textinput.New()
	providerInput.Placeholder = "Provider (github/azuredevops)"
	providerInput.CharLimit = 20

	usernameInput := textinput.New()
	usernameInput.Placeholder = "Username"
	usernameInput.CharLimit = 50

	return &PATsViewModel{
		list:          l,
		Mode:          PATModeList,
		nameInput:     nameInput,
		tokenInput:    tokenInput,
		providerInput: providerInput,
		usernameInput: usernameInput,
		inputFocus:    0,
	}
}

func (m *PATsViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height-5)
}

func (m *PATsViewModel) SetPATs(pats []domain.PAT) {
	items := make([]list.Item, len(pats))
	for i, pat := range pats {
		items[i] = PATItem{pat: pat}
	}
	m.list.SetItems(items)
}

func (m *PATsViewModel) EnterAddMode() {
	m.Mode = PATModeAdd
	m.inputFocus = 0
	m.nameInput.Focus()
	m.nameInput.SetValue("")
	m.tokenInput.SetValue("")
	m.providerInput.SetValue("")
	m.usernameInput.SetValue("")
}

func (m *PATsViewModel) ExitAddMode() {
	m.Mode = PATModeList
	m.nameInput.Blur()
	m.tokenInput.Blur()
	m.providerInput.Blur()
	m.usernameInput.Blur()
}

func (m *PATsViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	if m.Mode == PATModeAdd {
		return m.updateAddMode(msg)
	}

	m.list, cmd = m.list.Update(msg)
	return cmd
}

func (m *PATsViewModel) updateAddMode(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.nextInput()
			return nil
		case "shift+tab", "up":
			m.prevInput()
			return nil
		}
	}

	switch m.inputFocus {
	case 0:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case 1:
		m.tokenInput, cmd = m.tokenInput.Update(msg)
	case 2:
		m.providerInput, cmd = m.providerInput.Update(msg)
	case 3:
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	}

	return cmd
}

func (m *PATsViewModel) nextInput() {
	m.blurAll()
	m.inputFocus = (m.inputFocus + 1) % 4
	m.focusCurrent()
}

func (m *PATsViewModel) prevInput() {
	m.blurAll()
	m.inputFocus = (m.inputFocus - 1 + 4) % 4
	m.focusCurrent()
}

func (m *PATsViewModel) blurAll() {
	m.nameInput.Blur()
	m.tokenInput.Blur()
	m.providerInput.Blur()
	m.usernameInput.Blur()
}

func (m *PATsViewModel) focusCurrent() {
	switch m.inputFocus {
	case 0:
		m.nameInput.Focus()
	case 1:
		m.tokenInput.Focus()
	case 2:
		m.providerInput.Focus()
	case 3:
		m.usernameInput.Focus()
	}
}

func (m *PATsViewModel) GetNewPAT() domain.PAT {
	return domain.PAT{
		Name:     m.nameInput.Value(),
		Token:    m.tokenInput.Value(),
		Provider: domain.ProviderType(m.providerInput.Value()),
		Username: m.usernameInput.Value(),
	}
}

func (m *PATsViewModel) GetSelectedPAT() *domain.PAT {
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}

	patItem, ok := item.(PATItem)
	if !ok {
		return nil
	}

	return &patItem.pat
}

func (m *PATsViewModel) View() string {
	if m.Mode == PATModeAdd {
		return m.viewAddMode()
	}
	return m.viewListMode()
}

func (m *PATsViewModel) viewListMode() string {
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Render("\nEnter: Select | a: Add | d: Delete | q: Back")

	return m.list.View() + help
}

func (m *PATsViewModel) viewAddMode() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Render("Add New PAT\n\n")

	b.WriteString(title)
	b.WriteString("Name:\n")
	b.WriteString(m.nameInput.View() + "\n\n")
	b.WriteString("Token:\n")
	b.WriteString(m.tokenInput.View() + "\n\n")
	b.WriteString("Provider:\n")
	b.WriteString(m.providerInput.View() + "\n\n")
	b.WriteString("Username:\n")
	b.WriteString(m.usernameInput.View() + "\n\n")

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Render("Tab: Next | Shift+Tab: Previous | Enter: Save | Esc: Cancel")

	b.WriteString(help)

	return b.String()
}
