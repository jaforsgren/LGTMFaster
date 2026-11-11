package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	snippetLinesBefore     = 3
	snippetLinesAfter      = 1
	snippetLineNumberWidth = 5
	snippetFocusMarker     = "▶ "
	snippetMarkerPadding   = "  "
	diffLineTypeAdd        = "add"
	diffLineTypeDelete     = "delete"
)

var (
	snippetAddStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	snippetDeleteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	snippetContextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	snippetNumberStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	snippetFocusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
)

// codeSnippet is a few diff lines around the one a comment is about.
type codeSnippet struct {
	lines []domain.DiffLine
	focus int
}

// CommentTarget is the diff line an inline comment is being written for.
type CommentTarget struct {
	FilePath string
	Line     int
	Side     string
	snippet  codeSnippet
}

// Describe names the line as "path · line N", saying when it is a deleted line.
func (t CommentTarget) Describe() string {
	description := fmt.Sprintf("%s · line %d", t.FilePath, t.Line)
	if t.Side == domain.CommentSideLeft {
		description += " (deleted line)"
	}
	return description
}

func fileDiffLines(file domain.FileDiff) []domain.DiffLine {
	var lines []domain.DiffLine
	for _, hunk := range file.Hunks {
		lines = append(lines, hunk.Lines...)
	}
	return lines
}

func snippetAround(lines []domain.DiffLine, index int) codeSnippet {
	start := max(index-snippetLinesBefore, 0)
	end := min(index+snippetLinesAfter+1, len(lines))
	return codeSnippet{lines: lines[start:end], focus: index - start}
}

// commentTargetAt describes the line at index of a file's diff lines.
func commentTargetAt(file domain.FileDiff, index int) (CommentTarget, bool) {
	lines := fileDiffLines(file)
	if index < 0 || index >= len(lines) {
		return CommentTarget{}, false
	}
	line := lines[index]
	target := CommentTarget{FilePath: getFilePath(file), Line: line.NewLine, Side: domain.CommentSideRight, snippet: snippetAround(lines, index)}
	if line.Type == diffLineTypeDelete {
		target.Line = line.OldLine
		target.Side = domain.CommentSideLeft
	}
	return target, true
}

// snippetForComment finds the diff lines around a comment's line, counting in the side it was made on.
func snippetForComment(diff *domain.Diff, comment domain.Comment) (codeSnippet, bool) {
	if diff == nil || comment.Line <= 0 {
		return codeSnippet{}, false
	}
	for _, file := range diff.Files {
		if getFilePath(file) != comment.FilePath {
			continue
		}
		lines := fileDiffLines(file)
		for i, line := range lines {
			if commentIsOnLine(comment, line) {
				return snippetAround(lines, i), true
			}
		}
	}
	return codeSnippet{}, false
}

// commentIsOnLine matches a comment to a diff line: left-side comments to the old file's numbers of
// deleted lines, all others to the new file's numbers.
func commentIsOnLine(comment domain.Comment, line domain.DiffLine) bool {
	if comment.Side == domain.CommentSideLeft {
		return line.Type == diffLineTypeDelete && line.OldLine == comment.Line
	}
	return line.Type != diffLineTypeDelete && line.NewLine == comment.Line
}

// renderCodeSnippet shows the lines with their numbers, coloured like the diff, with the focused one
// marked.
func renderCodeSnippet(snippet codeSnippet, width int) string {
	rows := make([]string, len(snippet.lines))
	for i, line := range snippet.lines {
		number := line.NewLine
		style := snippetContextStyle
		switch line.Type {
		case diffLineTypeAdd:
			style = snippetAddStyle
		case diffLineTypeDelete:
			number = line.OldLine
			style = snippetDeleteStyle
		}

		marker := snippetMarkerPadding
		if i == snippet.focus {
			marker = snippetFocusStyle.Render(snippetFocusMarker)
			style = style.Bold(true)
		}
		gutter := snippetNumberStyle.Render(fmt.Sprintf("%*d ", snippetLineNumberWidth, number))
		contentWidth := max(width-snippetLineNumberWidth-len(snippetMarkerPadding)-1, minPaneSize)
		rows[i] = marker + gutter + style.Render(truncateString(line.Content, contentWidth))
	}
	return strings.Join(rows, "\n")
}
