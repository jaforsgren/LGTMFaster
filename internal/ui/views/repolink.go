package views

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/git"
)

type RepoLinkViewModel struct {
	pathInput     textinput.Model
	suggestions   []string
	selectedIdx   int
	width         int
	height        int
	active        bool
	repoFullName  string
	errorMessage  string
	validatedPath string
	isValidRepo   bool
}

func NewRepoLinkView() *RepoLinkViewModel {
	ti := textinput.New()
	ti.Placeholder = "Enter path to local repository (e.g., ~/projects/myrepo)"
	ti.CharLimit = 500
	ti.Width = 60

	return &RepoLinkViewModel{
		pathInput:   ti,
		suggestions: []string{},
		selectedIdx: -1,
		active:      false,
	}
}

func (m *RepoLinkViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.pathInput.Width = width - 8
}

func (m *RepoLinkViewModel) Activate(repoFullName string) {
	m.active = true
	m.repoFullName = repoFullName
	m.pathInput.Focus()
	m.pathInput.SetValue("")
	m.suggestions = []string{}
	m.selectedIdx = -1
	m.errorMessage = ""
	m.validatedPath = ""
	m.isValidRepo = false
}

func (m *RepoLinkViewModel) ActivateWithPath(repoFullName, existingPath string) {
	m.Activate(repoFullName)
	if existingPath != "" {
		m.pathInput.SetValue(existingPath)
		m.validateAndUpdateSuggestions()
	}
}

func (m *RepoLinkViewModel) Deactivate() {
	m.active = false
	m.pathInput.Blur()
	m.pathInput.SetValue("")
	m.suggestions = []string{}
	m.errorMessage = ""
}

func (m *RepoLinkViewModel) IsActive() bool {
	return m.active
}

func (m *RepoLinkViewModel) GetPath() string {
	return m.pathInput.Value()
}

func (m *RepoLinkViewModel) GetValidatedPath() string {
	return m.validatedPath
}

func (m *RepoLinkViewModel) IsValidRepo() bool {
	return m.isValidRepo
}

func (m *RepoLinkViewModel) GetRepoFullName() string {
	return m.repoFullName
}

func (m *RepoLinkViewModel) validateAndUpdateSuggestions() {
	path := m.pathInput.Value()
	if path == "" {
		m.suggestions = []string{}
		m.errorMessage = ""
		m.validatedPath = ""
		m.isValidRepo = false
		return
	}

	expandedPath, err := git.ExpandPath(path)
	if err != nil {
		m.errorMessage = err.Error()
		m.validatedPath = ""
		m.isValidRepo = false
		m.suggestions = []string{}
		return
	}

	if git.IsGitRepo(expandedPath) {
		m.validatedPath = expandedPath
		m.isValidRepo = true
		m.errorMessage = ""
		m.suggestions = []string{}
		return
	}

	m.validatedPath = expandedPath
	m.isValidRepo = false

	dir := expandedPath
	if !strings.HasSuffix(path, "/") {
		dir = filepath.Dir(expandedPath)
	}

	contents, err := git.ListDirectoryContents(dir)
	if err != nil {
		m.suggestions = []string{}
		m.errorMessage = "Directory not found"
		return
	}

	m.errorMessage = ""
	prefix := filepath.Base(expandedPath)
	if strings.HasSuffix(path, "/") {
		prefix = ""
	}

	var filtered []string
	for _, entry := range contents {
		if prefix == "" || strings.HasPrefix(strings.ToLower(entry), strings.ToLower(prefix)) {
			filtered = append(filtered, entry)
		}
	}

	if len(filtered) > 8 {
		filtered = filtered[:8]
	}
	m.suggestions = filtered
	m.selectedIdx = -1
}

func (m *RepoLinkViewModel) SelectSuggestion() {
	if m.selectedIdx >= 0 && m.selectedIdx < len(m.suggestions) {
		path := m.pathInput.Value()
		selected := m.suggestions[m.selectedIdx]

		var newPath string
		if strings.HasSuffix(path, "/") {
			newPath = path + selected
		} else {
			dir := filepath.Dir(path)
			if dir == "." {
				newPath = selected
			} else {
				newPath = dir + "/" + selected
			}
		}

		m.pathInput.SetValue(newPath)
		m.pathInput.SetCursor(len(newPath))
		m.selectedIdx = -1
		m.validateAndUpdateSuggestions()
	}
}

func (m *RepoLinkViewModel) NextSuggestion() {
	if len(m.suggestions) == 0 {
		return
	}
	m.selectedIdx++
	if m.selectedIdx >= len(m.suggestions) {
		m.selectedIdx = 0
	}
}

func (m *RepoLinkViewModel) PrevSuggestion() {
	if len(m.suggestions) == 0 {
		return
	}
	m.selectedIdx--
	if m.selectedIdx < 0 {
		m.selectedIdx = len(m.suggestions) - 1
	}
}

func (m *RepoLinkViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	m.validateAndUpdateSuggestions()
	return cmd
}

func (m *RepoLinkViewModel) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Padding(1, 0)

	b.WriteString(titleStyle.Render("Link Repository"))
	b.WriteString("\n\n")

	repoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Bold(true)

	b.WriteString("Remote: ")
	b.WriteString(repoStyle.Render(m.repoFullName))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF"))

	b.WriteString(labelStyle.Render("Local Path:"))
	b.WriteString("\n")
	b.WriteString(m.pathInput.View())
	b.WriteString("\n")

	if m.errorMessage != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444"))
		b.WriteString(errorStyle.Render("⚠ " + m.errorMessage))
		b.WriteString("\n")
	} else if m.isValidRepo {
		successStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981"))
		b.WriteString(successStyle.Render("✓ Valid git repository"))
		b.WriteString("\n")
	}

	if len(m.suggestions) > 0 {
		b.WriteString("\n")
		suggestLabelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true)
		b.WriteString(suggestLabelStyle.Render("Suggestions (Tab to select, Up/Down to navigate):"))
		b.WriteString("\n")

		for i, suggestion := range m.suggestions {
			if i == m.selectedIdx {
				selectedStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#7C3AED")).
					Bold(true)
				b.WriteString(selectedStyle.Render("▸ " + suggestion))
			} else {
				normalStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#9CA3AF"))
				b.WriteString(normalStyle.Render("  " + suggestion))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)

	help := "Enter: Confirm | Tab: Complete | ↑/↓: Navigate | Esc: Cancel"
	b.WriteString(helpStyle.Render(help))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(1, 2).
		Width(m.width - 4)

	return boxStyle.Render(b.String())
}
