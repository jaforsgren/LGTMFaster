package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

// diffPosition is a line of the diff: a file and the index of the line among its diff lines.
type diffPosition struct {
	file int
	line int
}

func (p diffPosition) before(other diffPosition) bool {
	return p.file < other.file || p.file == other.file && p.line < other.line
}

func (m *PRInspectViewModel) cursorPosition() diffPosition {
	return diffPosition{file: m.currentFile, line: m.currentLineIdx}
}

func (m *PRInspectViewModel) ActivateSearch() {
	m.search.Activate()
}

func (m *PRInspectViewModel) DeactivateSearch() {
	m.search.Deactivate()
	m.updateViewport()
}

func (m *PRInspectViewModel) IsSearching() bool {
	return m.search.IsActive()
}

func (m *PRInspectViewModel) SearchQuery() string {
	return m.search.Query()
}

func (m *PRInspectViewModel) ClearSearch() {
	m.search.Clear()
	m.updateViewport()
}

// UpdateSearch edits the query and moves to its first match from the cursor on, in any file.
func (m *PRInspectViewModel) UpdateSearch(msg tea.Msg) tea.Cmd {
	cmd := m.search.Update(msg)
	if !m.stepMatch(true, true) {
		m.updateViewport()
	}
	return cmd
}

// NextMatch and PrevMatch move to the next or previous line containing the query, going on into
// the other files and wrapping around, and report whether there is a match at all.
func (m *PRInspectViewModel) NextMatch() bool {
	return m.stepMatch(true, false)
}

func (m *PRInspectViewModel) PrevMatch() bool {
	return m.stepMatch(false, false)
}

func (m *PRInspectViewModel) stepMatch(forward, includeCursor bool) bool {
	matches := m.diffMatches()
	if len(matches) == 0 {
		return false
	}
	cursor := m.cursorPosition()
	target := matches[0]
	if !forward {
		target = matches[len(matches)-1]
	}
	if forward {
		for _, match := range matches {
			if cursor.before(match) || includeCursor && match == cursor {
				target = match
				break
			}
		}
	} else {
		for i := len(matches) - 1; i >= 0; i-- {
			if matches[i].before(cursor) {
				target = matches[i]
				break
			}
		}
	}
	m.goToPosition(target)
	return true
}

// diffMatches are the shown lines containing the query, in file order.
func (m *PRInspectViewModel) diffMatches() []diffPosition {
	if m.diff == nil || m.search.Query() == "" {
		return nil
	}
	var matches []diffPosition
	for fileIndex, file := range m.diff.Files {
		for lineIndex, line := range fileDiffLines(file) {
			if m.isLineShown(line) && m.search.Matches(line.Content) {
				matches = append(matches, diffPosition{file: fileIndex, line: lineIndex})
			}
		}
	}
	return matches
}

// searchPosition is the cursor's position among the matches (0 when it is not on one), and their
// count.
func (m *PRInspectViewModel) searchPosition() (int, int) {
	matches := m.diffMatches()
	cursor := m.cursorPosition()
	for i, match := range matches {
		if match == cursor {
			return i + 1, len(matches)
		}
	}
	return 0, len(matches)
}

func (m *PRInspectViewModel) goToPosition(position diffPosition) {
	m.currentFile = position.file
	m.moveCursorTo(position.line)
}

// GoToFile shows a file of the diff from its top.
func (m *PRInspectViewModel) GoToFile(index int) {
	if m.diff == nil || index < 0 || index >= len(m.diff.Files) {
		return
	}
	m.currentFile = index
	m.currentLineIdx = 0
	m.updateViewport()
	m.viewport.GotoTop()
}

// DiffFiles are the files the diff shows, which the file finder searches.
func (m *PRInspectViewModel) DiffFiles() ([]domain.FileDiff, int) {
	if m.diff == nil {
		return nil, 0
	}
	return m.diff.Files, m.currentFile
}
