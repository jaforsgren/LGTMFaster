package views

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	minDiffGutterDigits = 3
	diffGutterSeparator = " │ "
	noCursorRow         = -1
	pendingSpaceLabel   = "␣"
	decimalBase         = 10
)

var diffGutterStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

// diffMotion is a vim motion being typed in the diff: a count, then g for gg, or the space leader.
type diffMotion struct {
	count        int
	pendingG     bool
	pendingSpace bool
}

func (d diffMotion) String() string {
	pending := ""
	if d.count > 0 {
		pending = strconv.Itoa(d.count)
	}
	if d.pendingG {
		pending += "g"
	}
	if d.pendingSpace {
		pending += pendingSpaceLabel
	}
	return pending
}

// PushMotionDigit adds a digit to the count. A leading 0 is not a count, as in vim.
func (m *PRInspectViewModel) PushMotionDigit(digit int) {
	if digit == 0 && m.motion.count == 0 {
		return
	}
	m.motion.pendingG = false
	m.motion.pendingSpace = false
	m.motion.count = m.motion.count*decimalBase + digit
}

// PressG completes gg on the second g: to the counted line, else the first line.
func (m *PRInspectViewModel) PressG() {
	m.motion.pendingSpace = false
	if !m.motion.pendingG {
		m.motion.pendingG = true
		return
	}
	m.jumpToMotionTarget(m.FirstLine)
}

// PressShiftG is G: to the counted line, else the last line.
func (m *PRInspectViewModel) PressShiftG() {
	m.jumpToMotionTarget(m.LastLine)
}

// PressSpace reports the second space of the <space><space> leader, which opens the file finder.
func (m *PRInspectViewModel) PressSpace() bool {
	if m.motion.pendingSpace {
		m.ResetMotion()
		return true
	}
	m.motion = diffMotion{pendingSpace: true}
	return false
}

func (m *PRInspectViewModel) jumpToMotionTarget(withoutCount func()) {
	count := m.motion.count
	m.ResetMotion()
	if count > 0 {
		m.GoToLine(count)
		return
	}
	withoutCount()
}

func (m *PRInspectViewModel) ResetMotion() {
	m.motion = diffMotion{}
}

// PendingMotion is what has been typed of a motion so far, like vim shows it.
func (m *PRInspectViewModel) PendingMotion() string {
	return m.motion.String()
}

func (m *PRInspectViewModel) FirstLine() {
	if lines := m.visibleLineIndices(); len(lines) > 0 {
		m.moveCursorTo(lines[0])
	}
}

func (m *PRInspectViewModel) LastLine() {
	if lines := m.visibleLineIndices(); len(lines) > 0 {
		m.moveCursorTo(lines[len(lines)-1])
	}
}

// GoToLine moves to the shown line with the given number in the new file, or the next one after
// it when the diff does not show that line, else to the last line.
func (m *PRInspectViewModel) GoToLine(number int) {
	indices := m.visibleLineIndices()
	if len(indices) == 0 {
		return
	}
	lines := fileDiffLines(m.diff.Files[m.currentFile])
	for _, index := range indices {
		line := lines[index]
		if line.Type != diffLineTypeDelete && line.NewLine >= number {
			m.moveCursorTo(index)
			return
		}
	}
	m.moveCursorTo(indices[len(indices)-1])
}

// visibleLineIndices are the indices of the current file's diff lines the view mode shows.
func (m *PRInspectViewModel) visibleLineIndices() []int {
	if m.diff == nil || len(m.diff.Files) == 0 {
		return nil
	}
	var indices []int
	for i, line := range fileDiffLines(m.diff.Files[m.currentFile]) {
		if m.isLineShown(line) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m *PRInspectViewModel) isLineShown(line domain.DiffLine) bool {
	return m.diffViewMode == DiffViewModeFull || line.Type != diffLineTypeCtx
}

func (m *PRInspectViewModel) moveCursorTo(index int) {
	m.currentLineIdx = index
	m.updateViewport()
	m.ensureLineVisible()
}

// NextChange and PrevChange move to the start of the next or previous run of added and deleted
// lines, going on into the other files without wrapping, and report whether there was one. Inside
// a run, PrevChange goes to its start first, like vim's [c.
func (m *PRInspectViewModel) NextChange() bool {
	return m.stepChange(true)
}

func (m *PRInspectViewModel) PrevChange() bool {
	return m.stepChange(false)
}

func (m *PRInspectViewModel) stepChange(forward bool) bool {
	target, found := positionFrom(m.changeStarts(), m.cursorPosition(), forward, false)
	if found {
		m.goToPosition(target)
	}
	return found
}

// changeStarts are the first lines of every run of added and deleted lines, in file order.
func (m *PRInspectViewModel) changeStarts() []diffPosition {
	if m.diff == nil {
		return nil
	}
	var starts []diffPosition
	for fileIndex, file := range m.diff.Files {
		inChange := false
		for lineIndex, line := range fileDiffLines(file) {
			isChange := line.Type == diffLineTypeAdd || line.Type == diffLineTypeDelete
			if isChange && !inChange {
				starts = append(starts, diffPosition{file: fileIndex, line: lineIndex})
			}
			inChange = isChange
		}
	}
	return starts
}

// ensureLineVisible scrolls the cursor's row, as the last render placed it, into view.
func (m *PRInspectViewModel) ensureLineVisible() {
	if m.cursorRow == noCursorRow || m.viewport.Height <= 0 {
		return
	}
	if m.cursorRow >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.YOffset = m.cursorRow - m.viewport.Height + 1
	}
	if m.cursorRow < m.viewport.YOffset {
		m.viewport.YOffset = m.cursorRow
	}
	m.clampViewportOffset()
}

// diffGutterDigits fits the file's largest line number, on either side.
func diffGutterDigits(file domain.FileDiff) int {
	largest := 0
	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			largest = max(largest, line.OldLine, line.NewLine)
		}
	}
	return max(len(strconv.Itoa(largest)), minDiffGutterDigits)
}

// renderDiffGutter shows the line's number in the old and the new file, leaving out the side a
// deleted or added line does not exist on.
func renderDiffGutter(line domain.DiffLine, digits int) string {
	oldNumber, newNumber := strconv.Itoa(line.OldLine), strconv.Itoa(line.NewLine)
	switch line.Type {
	case diffLineTypeAdd:
		oldNumber = ""
	case diffLineTypeDelete:
		newNumber = ""
	}
	return diffGutterStyle.Render(fmt.Sprintf("%*s %*s", digits, oldNumber, digits, newNumber) + diffGutterSeparator)
}
