package views

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/sahilm/fuzzy"
)

// FileFinderAction is what a key press in the file finder asks the app to do.
type FileFinderAction int

const (
	FileFinderActionNone FileFinderAction = iota
	FileFinderActionOpen
	FileFinderActionCancel
)

const (
	fileFinderTitle    = "Find file"
	fileFinderPrompt   = "> "
	fileFinderHelp     = "↑/↓ ctrl+n/p: move | enter: open | esc: close"
	fileFinderNoMatch  = "No file matches."
	fileFinderCountFmt = "%d/%d"
	fileFinderMaxWidth = 160
	// fileFinderWidthPercent is the share of the screen the finder takes, up to its max width.
	fileFinderWidthPercent = 80
	fileFinderMaxRows      = 12
	fileFinderCharLimit    = 200
	fileFinderStatsWidth   = 12
	// fileFinderChromeRows is the box's border and padding, title, prompt, divider, blank and help.
	fileFinderChromeRows = 9
	fileFinderBoxInset   = 6
)

var (
	fileFinderBoxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(focusedPaneBorderColor).Padding(1, 2)
	fileFinderMatchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	fileFinderAddStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	fileFinderDelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
)

// fileFinderEntry is a file of the diff with its index there and how many lines it changes.
type fileFinderEntry struct {
	index   int
	path    string
	added   int
	deleted int
}

// fileFinderResult is an entry the query matches, with the positions of the matched characters.
type fileFinderResult struct {
	entry   fileFinderEntry
	matched []int
}

// FileFinderViewModel fuzzy-finds a file of the diff, like telescope's find_files.
type FileFinderViewModel struct {
	entries []fileFinderEntry
	results []fileFinderResult
	input   textinput.Model
	cursor  listCursor
	active  bool
	width   int
	height  int
}

func NewFileFinderView() *FileFinderViewModel {
	input := textinput.New()
	input.Prompt = fileFinderPrompt
	input.CharLimit = fileFinderCharLimit
	return &FileFinderViewModel{input: input, cursor: newListCursor()}
}

func (m *FileFinderViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.input.Width = max(m.boxWidth()-len(fileFinderPrompt)-fileFinderBoxInset, minPaneSize)
	m.cursor.SetVisibleRows(m.resultRows())
}

func (m *FileFinderViewModel) boxWidth() int {
	return max(min(m.width*fileFinderWidthPercent/percentScale, fileFinderMaxWidth), minPaneSize)
}

func (m *FileFinderViewModel) resultRows() int {
	return max(min(m.height-fileFinderChromeRows, fileFinderMaxRows), minPaneSize)
}

// Activate lists the diff's files with the current one selected.
func (m *FileFinderViewModel) Activate(files []domain.FileDiff, currentFile int) {
	m.entries = make([]fileFinderEntry, len(files))
	for i, file := range files {
		added, deleted := countChangedLines(file)
		m.entries[i] = fileFinderEntry{index: i, path: getFilePath(file), added: added, deleted: deleted}
	}
	m.input.SetValue("")
	m.input.Focus()
	m.active = true
	m.rebuild()
	m.cursor.SetIndex(currentFile, len(m.results))
}

func (m *FileFinderViewModel) Deactivate() {
	m.active = false
	m.input.Blur()
}

func (m *FileFinderViewModel) IsActive() bool {
	return m.active
}

// SelectedFile is the diff index of the selected file, false when nothing matches.
func (m *FileFinderViewModel) SelectedFile() (int, bool) {
	index := m.cursor.Index()
	if index < 0 || index >= len(m.results) {
		return 0, false
	}
	return m.results[index].entry.index, true
}

func (m *FileFinderViewModel) HandleKey(msg tea.KeyMsg) (FileFinderAction, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if _, ok := m.SelectedFile(); !ok {
			return FileFinderActionNone, nil
		}
		return FileFinderActionOpen, nil
	case "esc":
		return FileFinderActionCancel, nil
	case "down", "ctrl+n", "ctrl+j":
		m.cursor.MoveDown(len(m.results))
		return FileFinderActionNone, nil
	case "up", "ctrl+p", "ctrl+k":
		m.cursor.MoveUp()
		return FileFinderActionNone, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.rebuild()
	m.cursor.Reset()
	return FileFinderActionNone, cmd
}

// rebuild lists every file in diff order without a query, else the fuzzy matches, best first.
func (m *FileFinderViewModel) rebuild() {
	query := strings.TrimSpace(m.input.Value())
	m.results = m.results[:0]
	if query == "" {
		for _, entry := range m.entries {
			m.results = append(m.results, fileFinderResult{entry: entry})
		}
	} else {
		paths := make([]string, len(m.entries))
		for i, entry := range m.entries {
			paths[i] = entry.path
		}
		for _, match := range fuzzy.Find(query, paths) {
			m.results = append(m.results, fileFinderResult{entry: m.entries[match.Index], matched: match.MatchedIndexes})
		}
	}
	m.cursor.Clamp(len(m.results))
}

func countChangedLines(file domain.FileDiff) (int, int) {
	added, deleted := 0, 0
	for _, line := range fileDiffLines(file) {
		switch line.Type {
		case diffLineTypeAdd:
			added++
		case diffLineTypeDelete:
			deleted++
		}
	}
	return added, deleted
}

func (m *FileFinderViewModel) View() string {
	if !m.active {
		return ""
	}
	innerWidth := m.boxWidth() - fileFinderBoxInset

	var b strings.Builder
	count := listMutedStyle.Render(fmt.Sprintf(fileFinderCountFmt, len(m.results), len(m.entries)))
	title := listTitleStyle.Render(fileFinderTitle)
	b.WriteString(title + strings.Repeat(" ", max(innerWidth-lipgloss.Width(title)-lipgloss.Width(count), 1)) + count)
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n")
	b.WriteString(listDividerStyle.Render(strings.Repeat(listDividerGlyph, innerWidth)))
	b.WriteString("\n")

	rows := make([]string, m.resultRows())
	if len(m.results) == 0 {
		rows[0] = listMutedStyle.Render(fileFinderNoMatch)
	}
	start, end := m.cursor.VisibleRange(len(m.results))
	for i := start; i < end; i++ {
		rows[i-start] = m.renderResult(m.results[i], i == m.cursor.Index(), innerWidth)
	}
	b.WriteString(strings.Join(rows, "\n"))
	b.WriteString("\n\n")
	b.WriteString(listMutedStyle.Render(truncateString(fileFinderHelp, innerWidth)))

	// lipgloss widths leave out the border, so the box is boxWidth wide with it.
	box := fileFinderBoxStyle.Width(m.boxWidth() - paneBorderSize).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *FileFinderViewModel) renderResult(result fileFinderResult, selected bool, width int) string {
	marker, style := unselectedRowMarker, listRowStyle
	if selected {
		marker, style = selectedRowMarker, listSelectedRowStyle
	}
	stats := fileFinderAddStyle.Render(fmt.Sprintf("+%d", result.entry.added)) + " " +
		fileFinderDelStyle.Render(fmt.Sprintf("-%d", result.entry.deleted))
	pathWidth := max(width-len(marker)-fileFinderStatsWidth, minPaneSize)
	shown, cut := truncateStart(result.entry.path, pathWidth)
	path := highlightMatches(shown, shiftMatches(result.matched, cut), style)
	padding := strings.Repeat(" ", max(pathWidth-lipgloss.Width(path), 0))
	return style.Render(marker) + path + padding + " " + stats
}

// shiftMatches maps matched byte positions of a path onto the path with its first cut bytes
// replaced by the truncation marker, dropping the ones that were cut.
func shiftMatches(matched []int, cut int) []int {
	if cut == 0 {
		return matched
	}
	shifted := make([]int, 0, len(matched))
	for _, position := range matched {
		if position >= cut {
			shifted = append(shifted, position-cut+len(truncationMarker))
		}
	}
	return shifted
}

// highlightMatches styles the matched characters of s; positions are byte offsets, as fuzzy
// reports them.
func highlightMatches(s string, matched []int, style lipgloss.Style) string {
	if len(matched) == 0 {
		return style.Render(s)
	}
	var b strings.Builder
	for i, r := range s {
		if slices.Contains(matched, i) {
			b.WriteString(fileFinderMatchStyle.Render(string(r)))
		} else {
			b.WriteString(style.Render(string(r)))
		}
	}
	return b.String()
}
