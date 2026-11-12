package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ExtensionFilterAction is what a key press in the extension filter asks the app to do.
type ExtensionFilterAction int

const (
	ExtensionFilterActionNone ExtensionFilterAction = iota
	ExtensionFilterActionApply
	ExtensionFilterActionCancel
)

const (
	extensionFilterTitle  = "Show files with extension"
	extensionFilterHelp   = "j/k: move | space: toggle | a: all/none | enter: apply | esc: cancel"
	extensionShownMarker  = "[x] "
	extensionHiddenMarker = "[ ] "
	extensionFilterInset  = 4
)

var extensionFilterBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(mainWindowBorderColor).Padding(1, 2)

// ExtensionFilterViewModel picks which file extensions the diff shows.
type ExtensionFilterViewModel struct {
	extensions []ExtensionCount
	hidden     map[string]bool
	cursor     int
	active     bool
	width      int
	height     int
}

func NewExtensionFilterView() *ExtensionFilterViewModel {
	return &ExtensionFilterViewModel{hidden: make(map[string]bool)}
}

func (m *ExtensionFilterViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *ExtensionFilterViewModel) Activate(extensions []ExtensionCount, hidden map[string]bool) {
	m.extensions = extensions
	m.hidden = hidden
	m.cursor = 0
	m.active = true
}

func (m *ExtensionFilterViewModel) Deactivate() {
	m.active = false
}

func (m *ExtensionFilterViewModel) IsActive() bool {
	return m.active
}

// Hidden is the extensions the user unticked.
func (m *ExtensionFilterViewModel) Hidden() map[string]bool {
	return m.hidden
}

func (m *ExtensionFilterViewModel) HandleKey(msg tea.KeyMsg) ExtensionFilterAction {
	switch msg.String() {
	case "enter":
		return ExtensionFilterActionApply
	case "esc", "q":
		return ExtensionFilterActionCancel
	case "j", "down":
		m.cursor = min(m.cursor+1, max(len(m.extensions)-1, 0))
	case "k", "up":
		m.cursor = max(m.cursor-1, 0)
	case " ":
		if m.cursor < len(m.extensions) {
			extension := m.extensions[m.cursor].Extension
			m.hidden[extension] = !m.hidden[extension]
		}
	case "a":
		m.toggleAll()
	}
	return ExtensionFilterActionNone
}

// toggleAll shows every extension, or hides them all when they are already all shown.
func (m *ExtensionFilterViewModel) toggleAll() {
	allShown := true
	for _, extension := range m.extensions {
		if m.hidden[extension.Extension] {
			allShown = false
		}
	}
	for _, extension := range m.extensions {
		m.hidden[extension.Extension] = allShown
	}
}

func (m *ExtensionFilterViewModel) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder
	b.WriteString(listTitleStyle.Render(extensionFilterTitle))
	b.WriteString("\n\n")
	for i, extension := range m.extensions {
		marker := extensionShownMarker
		if m.hidden[extension.Extension] {
			marker = extensionHiddenMarker
		}
		row := fmt.Sprintf("%s%-12s %d files", marker, extension.Extension, extension.Files)
		if i == m.cursor {
			b.WriteString(listSelectedRowStyle.Render(selectedRowMarker + row))
		} else {
			b.WriteString(listRowStyle.Render(unselectedRowMarker + row))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(listMutedStyle.Render(extensionFilterHelp))
	return extensionFilterBoxStyle.Width(max(m.width-extensionFilterInset, minPaneSize)).Render(b.String())
}
