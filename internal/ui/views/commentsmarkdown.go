package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	markdownDiffFence   = "```diff"
	markdownFence       = "```"
	markdownTimeFormat  = "2006-01-02 15:04"
	deletedLineNote     = " (deleted line)"
	markdownEmptyNotice = "_No comments on this pull request._"
)

// groupedComments splits comments into the ones on the whole pull request and the ones on lines,
// with the files in name order.
type groupedComments struct {
	general   []domain.Comment
	filePaths []string
	byFile    map[string][]domain.Comment
}

func groupComments(comments []domain.Comment) groupedComments {
	grouped := groupedComments{byFile: make(map[string][]domain.Comment)}
	for _, comment := range comments {
		if comment.FilePath == "" {
			grouped.general = append(grouped.general, comment)
			continue
		}
		if _, seen := grouped.byFile[comment.FilePath]; !seen {
			grouped.filePaths = append(grouped.filePaths, comment.FilePath)
		}
		grouped.byFile[comment.FilePath] = append(grouped.byFile[comment.FilePath], comment)
	}
	sort.Strings(grouped.filePaths)
	return grouped
}

// Markdown renders every comment of the pull request as a document: votes and review comments
// first, then the inline comments per file, each under a diff block of the code it is about.
func (m *CommentDetailViewModel) Markdown(pr *domain.PullRequest) string {
	var b strings.Builder

	if pr != nil {
		fmt.Fprintf(&b, "# #%d %s\n\n", pr.Number, pr.Title)
		fmt.Fprintf(&b, "%s · `%s` → `%s` · %d comments\n\n", pr.Repository.FullName, pr.SourceBranch, pr.TargetBranch, len(m.comments))
		if pr.URL != "" {
			fmt.Fprintf(&b, "<%s>\n\n", pr.URL)
		}
	} else {
		fmt.Fprintf(&b, "# Comments (%d)\n\n", len(m.comments))
	}

	if len(m.comments) == 0 {
		b.WriteString(markdownEmptyNotice + "\n")
		return b.String()
	}

	grouped := groupComments(m.comments)
	if len(grouped.general) > 0 {
		b.WriteString("## Review comments\n\n")
		for _, comment := range grouped.general {
			writeMarkdownComment(&b, comment, "###")
		}
	}

	if len(grouped.filePaths) > 0 {
		b.WriteString("## Inline comments\n\n")
		for _, filePath := range grouped.filePaths {
			fmt.Fprintf(&b, "### `%s`\n\n", filePath)
			for _, comment := range grouped.byFile[filePath] {
				m.writeMarkdownInlineComment(&b, comment)
			}
		}
	}
	return b.String()
}

// writeMarkdownComment writes a vote as a list item and any other comment under its own heading.
func writeMarkdownComment(b *strings.Builder, comment domain.Comment, heading string) {
	if comment.IsSystem {
		fmt.Fprintf(b, "- **%s** · %s\n\n", comment.Body, comment.CreatedAt.Format(markdownTimeFormat))
		return
	}
	fmt.Fprintf(b, "%s %s · %s\n\n%s\n\n", heading, comment.Author.Username, comment.CreatedAt.Format(markdownTimeFormat), strings.TrimSpace(comment.Body))
}

func (m *CommentDetailViewModel) writeMarkdownInlineComment(b *strings.Builder, comment domain.Comment) {
	line := fmt.Sprintf("Line %d", comment.Line)
	if comment.Side == domain.CommentSideLeft {
		line += deletedLineNote
	}
	fmt.Fprintf(b, "#### %s · %s · %s\n\n", line, comment.Author.Username, comment.CreatedAt.Format(markdownTimeFormat))

	if snippet, found := snippetForComment(m.diff, comment); found {
		b.WriteString(markdownDiffFence + "\n")
		for _, snippetLine := range snippet.lines {
			b.WriteString(snippetLine.Content + "\n")
		}
		b.WriteString(markdownFence + "\n\n")
	}
	b.WriteString(strings.TrimSpace(comment.Body) + "\n\n")
}
