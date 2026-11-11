package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	commentBoxInset     = 4
	systemCommentMarker = "● "
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

// commentDetailHelpRows is the blank line and help line under the viewport.
const commentDetailHelpRows = 3

func (m *CommentDetailViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = max(height-commentDetailHelpRows, minPaneSize)
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

	help := helpStyle.Render("\nq/Esc: Back to Diff | ctrl+o: Open as markdown")

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

	grouped := groupComments(m.comments)
	generalComments := grouped.general

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
	if len(grouped.filePaths) > 0 {
		sectionHeaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true).
			Underline(true).
			Padding(0, 0, 1, 0)

		b.WriteString(sectionHeaderStyle.Render("Inline Comments"))
		b.WriteString("\n\n")

		for _, filePath := range grouped.filePaths {
			fileComments := grouped.byFile[filePath]
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

	if comment.IsSystem {
		b.WriteString(metaStyle.Render(fmt.Sprintf("%s%s · %s", systemCommentMarker, comment.Body, FormatAge(comment.CreatedAt))))
		b.WriteString("\n")
		return
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Padding(1, 2).
		Width(m.width - commentBoxInset)

	var content strings.Builder

	header := authorStyle.Render(comment.Author.Username)
	if comment.Line > 0 {
		lineDescription := fmt.Sprintf(" on line %d", comment.Line)
		if comment.Side == domain.CommentSideLeft {
			lineDescription += " (deleted line)"
		}
		header += metaStyle.Render(lineDescription)
	}
	header += metaStyle.Render(" · " + FormatAge(comment.CreatedAt))
	content.WriteString(header)
	content.WriteString("\n\n")

	if snippet, found := snippetForComment(m.diff, comment); found {
		content.WriteString(renderCodeSnippet(snippet, m.width-commentBoxInset*2))
		content.WriteString("\n\n")
	}

	content.WriteString(commentStyle.Render(comment.Body))

	b.WriteString(boxStyle.Render(content.String()))
}
