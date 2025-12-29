package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type PRInspectViewModel struct {
	pr           *domain.PullRequest
	diff         *domain.Diff
	comments     []domain.Comment
	viewport     viewport.Model
	currentFile  int
	width        int
	height       int
	showComments bool
}

func NewPRInspectView() *PRInspectViewModel {
	vp := viewport.New(0, 0)

	return &PRInspectViewModel{
		viewport:     vp,
		currentFile:  0,
		showComments: false,
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
	m.updateViewport()
}

func (m *PRInspectViewModel) SetDiff(diff *domain.Diff) {
	m.diff = diff
	m.currentFile = 0
	m.updateViewport()
}

func (m *PRInspectViewModel) SetComments(comments []domain.Comment) {
	m.comments = comments
	m.updateViewport()
}

func (m *PRInspectViewModel) GetPR() *domain.PullRequest {
	return m.pr
}

func (m *PRInspectViewModel) NextFile() {
	if m.diff != nil && m.currentFile < len(m.diff.Files)-1 {
		m.currentFile++
		m.updateViewport()
	}
}

func (m *PRInspectViewModel) PrevFile() {
	if m.currentFile > 0 {
		m.currentFile--
		m.updateViewport()
	}
}

func (m *PRInspectViewModel) ToggleComments() {
	m.showComments = !m.showComments
	m.updateViewport()
}

func (m *PRInspectViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "n":
			m.NextFile()
			return nil
		case "p":
			m.PrevFile()
			return nil
		case "c":
			m.ToggleComments()
			return nil
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

func (m *PRInspectViewModel) View() string {
	content := m.viewport.View()

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Render("\nn/p: Next/Prev File | c: Toggle Comments | a: Approve | r: Request Changes | Enter: Comment | q: Back")

	return content + "\n" + help
}

func (m *PRInspectViewModel) updateViewport() {
	var b strings.Builder

	if m.pr != nil {
		b.WriteString(m.renderPRHeader())
		b.WriteString("\n\n")
	}

	if m.diff != nil && len(m.diff.Files) > 0 {
		b.WriteString(m.renderDiff())
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
		return "No diff available"
	}

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

	for _, hunk := range file.Hunks {
		hunkHeaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3B82F6"))
		b.WriteString(hunkHeaderStyle.Render(hunk.Header))
		b.WriteString("\n")

		for _, line := range hunk.Lines {
			b.WriteString(m.renderDiffLine(line))
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	if m.showComments {
		b.WriteString(m.renderComments(getFilePath(file)))
	}

	return b.String()
}

func (m *PRInspectViewModel) renderDiffLine(line domain.DiffLine) string {
	style := lipgloss.NewStyle()

	switch line.Type {
	case "add":
		style = style.Foreground(lipgloss.Color("#10B981"))
	case "delete":
		style = style.Foreground(lipgloss.Color("#EF4444"))
	default:
		style = style.Foreground(lipgloss.Color("#6B7280"))
	}

	return style.Render(line.Content)
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
