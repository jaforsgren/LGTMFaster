package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type CommentDetailViewModel struct {
	viewport viewport.Model
	comments []domain.Comment
	diff     *domain.Diff
	width    int
	height   int
	active   bool
}

func NewCommentDetailView() *CommentDetailViewModel {
	vp := viewport.New(0, 0)

	return &CommentDetailViewModel{
		viewport: vp,
		active:   false,
	}
}

func (m *CommentDetailViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height - 10
}

func (m *CommentDetailViewModel) Activate(comments []domain.Comment, diff *domain.Diff) {
	m.active = true
	m.comments = comments
	m.diff = diff
	m.updateViewport()
}

func (m *CommentDetailViewModel) Deactivate() {
	m.active = false
}

func (m *CommentDetailViewModel) IsActive() bool {
	return m.active
}

func (m *CommentDetailViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

func (m *CommentDetailViewModel) View() string {
	content := m.viewport.View()

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)

	help := helpStyle.Render("\nq/Esc: Back to Diff")

	return content + "\n" + help
}

func (m *CommentDetailViewModel) updateViewport() {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Padding(1, 0)

	b.WriteString(titleStyle.Render(fmt.Sprintf("Comments (%d)", len(m.comments))))
	b.WriteString("\n\n")

	if len(m.comments) == 0 {
		noCommentsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true)
		b.WriteString(noCommentsStyle.Render("No comments on this PR"))
		m.viewport.SetContent(b.String())
		return
	}

	// Separate general and inline comments
	generalComments := []domain.Comment{}
	inlineComments := []domain.Comment{}

	for _, comment := range m.comments {
		if comment.FilePath == "" {
			generalComments = append(generalComments, comment)
		} else {
			inlineComments = append(inlineComments, comment)
		}
	}

	// Render general/review-level comments first
	if len(generalComments) > 0 {
		sectionHeaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true).
			Underline(true).
			Padding(0, 0, 1, 0)

		b.WriteString(sectionHeaderStyle.Render("Review Comments"))
		b.WriteString("\n\n")

		for _, comment := range generalComments {
			m.renderComment(&b, comment)
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	// Render inline comments grouped by file
	if len(inlineComments) > 0 {
		sectionHeaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true).
			Underline(true).
			Padding(0, 0, 1, 0)

		b.WriteString(sectionHeaderStyle.Render("Inline Comments"))
		b.WriteString("\n\n")

		commentsByFile := make(map[string][]domain.Comment)
		for _, comment := range inlineComments {
			commentsByFile[comment.FilePath] = append(commentsByFile[comment.FilePath], comment)
		}

		for filePath, fileComments := range commentsByFile {
			fileHeaderStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#3B82F6")).
				Bold(true).
				Background(lipgloss.Color("#1F2937")).
				Padding(0, 1)

			b.WriteString(fileHeaderStyle.Render(filePath))
			b.WriteString("\n\n")

			for _, comment := range fileComments {
				m.renderComment(&b, comment)
				b.WriteString("\n")
			}

			b.WriteString("\n")
		}
	}

	m.viewport.SetContent(b.String())
}

func (m *CommentDetailViewModel) renderComment(b *strings.Builder, comment domain.Comment) {
	metaStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)

	authorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true)

	commentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Padding(0, 2)

	codeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5563"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Padding(1, 2).
		Width(m.width - 4)

	var content strings.Builder

	header := authorStyle.Render(comment.Author.Username)
	if comment.Line > 0 {
		header += metaStyle.Render(fmt.Sprintf(" on line %d", comment.Line))
	}
	content.WriteString(header)
	content.WriteString("\n\n")

	if comment.Line > 0 && m.diff != nil {
		codeContext := m.getCodeContext(comment)
		if codeContext != "" {
			content.WriteString(codeStyle.Render(codeContext))
			content.WriteString("\n\n")
		}
	}

	content.WriteString(commentStyle.Render(comment.Body))

	b.WriteString(boxStyle.Render(content.String()))
}

func (m *CommentDetailViewModel) getCodeContext(comment domain.Comment) string {
	if m.diff == nil {
		return ""
	}

	for _, file := range m.diff.Files {
		filePath := file.NewPath
		if filePath == "" {
			filePath = file.OldPath
		}

		if filePath != comment.FilePath {
			continue
		}

		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				lineNum := line.NewLine
				if comment.Side == "LEFT" || line.Type == "delete" {
					lineNum = line.OldLine
				}

				if lineNum == comment.Line {
					return line.Content
				}
			}
		}
	}

	return ""
}
