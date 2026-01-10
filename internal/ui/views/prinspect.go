package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
)

type PRInspectMode int

const (
	PRInspectModeDescription PRInspectMode = iota
	PRInspectModeDiff
)

type PRInspectViewModel struct {
	pr              *domain.PullRequest
	diff            *domain.Diff
	comments        []domain.Comment
	viewport        viewport.Model
	currentFile     int
	currentLineIdx  int
	width           int
	height          int
	showComments    bool
	mode            PRInspectMode
	pendingComments []domain.Comment
}

func NewPRInspectView() *PRInspectViewModel {
	vp := viewport.New(0, 0)

	return &PRInspectViewModel{
		viewport:     vp,
		currentFile:  0,
		showComments: false,
		mode:         PRInspectModeDescription,
	}
}

func (m *PRInspectViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height - 10
}

func (m *PRInspectViewModel) SetPR(pr *domain.PullRequest) {
	m.pr = pr
	m.mode = PRInspectModeDescription
	m.updateViewport()
}

func (m *PRInspectViewModel) SetDiff(diff *domain.Diff) {
	m.diff = diff
	m.currentFile = 0
	logger.Log("PRInspectView: SetDiff called with %d files", len(diff.Files))
	if len(diff.Files) > 0 {
		for i, file := range diff.Files {
			logger.Log("PRInspectView: File %d: %s -> %s (%d hunks)", i+1, file.OldPath, file.NewPath, len(file.Hunks))
			if len(file.Hunks) > 0 {
				logger.Log("PRInspectView: File %d has %d lines in first hunk", i+1, len(file.Hunks[0].Lines))
			}
		}
	}
	m.updateViewport()
}

func (m *PRInspectViewModel) SetComments(comments []domain.Comment) {
	m.comments = comments
	m.updateViewport()
}

func (m *PRInspectViewModel) GetPR() *domain.PullRequest {
	return m.pr
}

func (m *PRInspectViewModel) SwitchToDiff() {
	m.mode = PRInspectModeDiff
	m.updateViewport()
}

func (m *PRInspectViewModel) SwitchToDescription() {
	m.mode = PRInspectModeDescription
	m.updateViewport()
}

func (m *PRInspectViewModel) GetMode() PRInspectMode {
	return m.mode
}

func (m *PRInspectViewModel) NextFile() {
	if m.diff != nil && m.currentFile < len(m.diff.Files)-1 {
		m.currentFile++
		m.currentLineIdx = 0
		m.updateViewport()
	}
}

func (m *PRInspectViewModel) PrevFile() {
	if m.currentFile > 0 {
		m.currentFile--
		m.currentLineIdx = 0
		m.updateViewport()
	}
}

func (m *PRInspectViewModel) ToggleComments() {
	m.showComments = !m.showComments
	m.updateViewport()
}

func (m *PRInspectViewModel) NextLine() {
	if m.diff == nil || len(m.diff.Files) == 0 {
		return
	}
	file := m.diff.Files[m.currentFile]
	totalLines := m.countTotalLines(file)
	if m.currentLineIdx < totalLines-1 {
		m.currentLineIdx++
		m.updateViewport()
		m.ensureLineVisible()
	}
}

func (m *PRInspectViewModel) PrevLine() {
	if m.currentLineIdx > 0 {
		m.currentLineIdx--
		m.updateViewport()
		m.ensureLineVisible()
	}
}

func (m *PRInspectViewModel) ensureLineVisible() {
	if m.diff == nil || len(m.diff.Files) == 0 {
		return
	}

	// Calculate approximate line position in rendered output
	// Each hunk has a header line, and we need to account for file header too
	file := m.diff.Files[m.currentFile]

	// Start with file header (takes about 3 lines)
	linePosition := 3

	lineIdx := 0
	for _, hunk := range file.Hunks {
		// Add 2 lines for hunk header and spacing
		linePosition += 2

		for range hunk.Lines {
			if lineIdx == m.currentLineIdx {
				// Found our line, now ensure it's visible
				viewportHeight := m.viewport.Height

				// If line is below visible area, scroll down
				if linePosition > m.viewport.YOffset+viewportHeight-1 {
					m.viewport.YOffset = linePosition - viewportHeight + 1
				}
				// If line is above visible area, scroll up
				if linePosition < m.viewport.YOffset {
					m.viewport.YOffset = linePosition
				}
				return
			}
			lineIdx++
			linePosition++
		}
		// Add spacing after hunk
		linePosition++
	}
}

func (m *PRInspectViewModel) countTotalLines(file domain.FileDiff) int {
	count := 0
	for _, hunk := range file.Hunks {
		count += len(hunk.Lines)
	}
	return count
}

func (m *PRInspectViewModel) GetCurrentLineInfo() *domain.DiffLine {
	if m.diff == nil || len(m.diff.Files) == 0 {
		return nil
	}

	file := m.diff.Files[m.currentFile]
	lineIdx := 0
	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			if lineIdx == m.currentLineIdx {
				return &line
			}
			lineIdx++
		}
	}
	return nil
}

func (m *PRInspectViewModel) AddPendingComment(body string) {
	if m.diff == nil || len(m.diff.Files) == 0 {
		return
	}

	lineInfo := m.GetCurrentLineInfo()
	if lineInfo == nil {
		return
	}

	file := m.diff.Files[m.currentFile]
	filePath := getFilePath(file)

	lineNumber := lineInfo.NewLine
	side := "RIGHT"
	if lineInfo.Type == "delete" {
		lineNumber = lineInfo.OldLine
		side = "LEFT"
	}

	comment := domain.Comment{
		Body:     body,
		FilePath: filePath,
		Line:     lineNumber,
		Side:     side,
	}

	m.pendingComments = append(m.pendingComments, comment)
}

func (m *PRInspectViewModel) GetPendingComments() []domain.Comment {
	return m.pendingComments
}

func (m *PRInspectViewModel) ClearPendingComments() {
	m.pendingComments = []domain.Comment{}
}

func (m *PRInspectViewModel) GetPendingCommentCount() int {
	return len(m.pendingComments)
}

func (m *PRInspectViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

func (m *PRInspectViewModel) View() string {
	content := m.viewport.View()

	var helpText string
	switch m.mode {
	case PRInspectModeDescription:
		helpText = "\nd: View Diff | ctrl+o: Open in Browser | q: Back"
	case PRInspectModeDiff:
		pendingCount := m.GetPendingCommentCount()
		countInfo := ""
		if pendingCount > 0 {
			countInfo = fmt.Sprintf(" [%d pending comments]", pendingCount)
		}
		helpText = fmt.Sprintf("\nFiles: n/p/←/→ | Lines: j/k | i: Inline Comment%s | c: Toggle Comments | a: Approve | r: Request Changes | Enter: General Comment | ctrl+o: Browser | q: Back", countInfo)
	}

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Render(helpText)

	return content + "\n" + help
}

func (m *PRInspectViewModel) updateViewport() {
	var b strings.Builder

	switch m.mode {
	case PRInspectModeDescription:
		if m.pr != nil {
			b.WriteString(m.renderPRHeader())
		}
	case PRInspectModeDiff:
		if m.diff != nil && len(m.diff.Files) > 0 {
			b.WriteString(m.renderDiff())
		}
	}

	m.viewport.SetContent(b.String())
}

func (m *PRInspectViewModel) renderPRHeader() string {
	if m.pr == nil {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true)

	b.WriteString(titleStyle.Render(m.pr.Title))
	b.WriteString("\n")

	metaStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	meta := fmt.Sprintf("%s #%d | %s → %s | by %s",
		m.pr.Repository.FullName,
		m.pr.Number,
		m.pr.SourceBranch,
		m.pr.TargetBranch,
		m.pr.Author.Username,
	)
	b.WriteString(metaStyle.Render(meta))
	b.WriteString("\n")

	statusStyle := lipgloss.NewStyle()
	statusText := string(m.pr.Status)
	switch m.pr.Status {
	case domain.PRStatusOpen:
		statusStyle = statusStyle.Foreground(lipgloss.Color("#10B981"))
	case domain.PRStatusClosed:
		statusStyle = statusStyle.Foreground(lipgloss.Color("#EF4444"))
	case domain.PRStatusMerged:
		statusStyle = statusStyle.Foreground(lipgloss.Color("#7C3AED"))
	}

	if m.pr.IsDraft {
		statusText += " (draft)"
	}

	b.WriteString(statusStyle.Render(statusText))
	b.WriteString("\n")

	if m.pr.Description != "" {
		b.WriteString("\n")
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FAFB"))
		b.WriteString(descStyle.Render(m.pr.Description))
	}

	return b.String()
}

func (m *PRInspectViewModel) renderDiff() string {
	if m.diff == nil || len(m.diff.Files) == 0 {
		logger.Log("PRInspectView: renderDiff - No diff available (diff nil: %v, files: %d)", m.diff == nil, 0)
		return "No diff available"
	}

	logger.Log("PRInspectView: renderDiff - Rendering file %d of %d", m.currentFile+1, len(m.diff.Files))

	var b strings.Builder

	fileHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Background(lipgloss.Color("#1F2937")).
		Padding(0, 1)

	file := m.diff.Files[m.currentFile]

	header := fmt.Sprintf("File %d/%d: %s",
		m.currentFile+1,
		len(m.diff.Files),
		getFilePath(file),
	)

	b.WriteString(fileHeaderStyle.Render(header))
	b.WriteString("\n\n")

	logger.Log("PRInspectView: renderDiff - File has %d hunks", len(file.Hunks))

	lineIdx := 0
	for hunkIdx, hunk := range file.Hunks {
		hunkHeaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3B82F6"))
		b.WriteString(hunkHeaderStyle.Render(hunk.Header))
		b.WriteString("\n")

		logger.Log("PRInspectView: renderDiff - Hunk %d has %d lines", hunkIdx+1, len(hunk.Lines))

		for _, line := range hunk.Lines {
			b.WriteString(m.renderDiffLine(line, lineIdx))
			b.WriteString("\n")
			lineIdx++
		}

		b.WriteString("\n")
	}

	if m.showComments {
		b.WriteString(m.renderComments(getFilePath(file)))
	}

	result := b.String()
	logger.Log("PRInspectView: renderDiff - Generated %d bytes of content", len(result))
	return result
}

func (m *PRInspectViewModel) renderDiffLine(line domain.DiffLine, lineIdx int) string {
	style := lipgloss.NewStyle()

	switch line.Type {
	case "add":
		style = style.Foreground(lipgloss.Color("#10B981"))
	case "delete":
		style = style.Foreground(lipgloss.Color("#EF4444"))
	default:
		style = style.Foreground(lipgloss.Color("#6B7280"))
	}

	isCursor := lineIdx == m.currentLineIdx
	hasPendingComment := m.hasPendingCommentOnLine(line)
	hasSubmittedComment := m.hasSubmittedCommentOnLine(line)

	prefix := ""
	if isCursor {
		prefix = "► "
		style = style.Bold(true).Background(lipgloss.Color("#374151")).Underline(true)
	} else {
		prefix = "  "
	}

	if hasPendingComment {
		prefix += "💬 "
	} else if hasSubmittedComment {
		prefix += "💭 "
	}

	return style.Render(prefix + line.Content)
}

func (m *PRInspectViewModel) hasPendingCommentOnLine(line domain.DiffLine) bool {
	if m.diff == nil || len(m.diff.Files) == 0 {
		return false
	}

	file := m.diff.Files[m.currentFile]
	filePath := getFilePath(file)

	lineNumber := line.NewLine
	if line.Type == "delete" {
		lineNumber = line.OldLine
	}

	for _, comment := range m.pendingComments {
		if comment.FilePath == filePath && comment.Line == lineNumber {
			return true
		}
	}
	return false
}

func (m *PRInspectViewModel) hasSubmittedCommentOnLine(line domain.DiffLine) bool {
	if m.diff == nil || len(m.diff.Files) == 0 {
		return false
	}

	file := m.diff.Files[m.currentFile]
	filePath := getFilePath(file)

	lineNumber := line.NewLine
	if line.Type == "delete" {
		lineNumber = line.OldLine
	}

	for _, comment := range m.comments {
		if comment.FilePath == filePath && comment.Line == lineNumber {
			return true
		}
	}
	return false
}

func (m *PRInspectViewModel) renderComments(filePath string) string {
	var b strings.Builder

	b.WriteString("\n")
	commentHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Bold(true)
	b.WriteString(commentHeaderStyle.Render("Comments"))
	b.WriteString("\n\n")

	relevantComments := []domain.Comment{}
	for _, comment := range m.comments {
		if comment.FilePath == filePath || comment.FilePath == "" {
			relevantComments = append(relevantComments, comment)
		}
	}

	if len(relevantComments) == 0 {
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true).
			Render("No comments"))
		return b.String()
	}

	for _, comment := range relevantComments {
		authorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)

		b.WriteString(authorStyle.Render(comment.Author.Username))
		if comment.Line > 0 {
			b.WriteString(fmt.Sprintf(" on line %d", comment.Line))
		}
		b.WriteString(":\n")

		commentStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FAFB")).
			PaddingLeft(2)
		b.WriteString(commentStyle.Render(comment.Body))
		b.WriteString("\n\n")
	}

	return b.String()
}

func getFilePath(file domain.FileDiff) string {
	if file.NewPath != "" {
		return file.NewPath
	}
	return file.OldPath
}
