package markdown

import (
	"regexp"
	"strings"
)

type Renderer struct {
	styles   Styles
	width    int
	hRuleStr string
}

func NewRenderer(styles Styles) *Renderer {
	return &Renderer{
		styles:   styles,
		width:    80,
		hRuleStr: "────────────────────────────────────────",
	}
}

func (r *Renderer) SetWidth(width int) {
	r.width = width
	if width > 10 {
		r.hRuleStr = strings.Repeat("─", width-4)
	}
}

func (r *Renderer) Render(text string) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	var result []string
	inCodeBlock := false
	codeBlockLines := []string{}

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				result = append(result, r.renderCodeBlock(codeBlockLines))
				codeBlockLines = []string{}
				inCodeBlock = false
			} else {
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			codeBlockLines = append(codeBlockLines, line)
			continue
		}

		rendered := r.renderLine(line)
		result = append(result, rendered)
	}

	if inCodeBlock && len(codeBlockLines) > 0 {
		result = append(result, r.renderCodeBlock(codeBlockLines))
	}

	return strings.Join(result, "\n")
}

func (r *Renderer) renderLine(line string) string {
	trimmed := strings.TrimSpace(line)

	if trimmed == "" {
		return ""
	}

	if r.isHorizontalRule(trimmed) {
		return r.styles.HRule.Render(r.hRuleStr)
	}

	if strings.HasPrefix(trimmed, "# ") {
		return r.styles.H1.Render(strings.TrimPrefix(trimmed, "# "))
	}
	if strings.HasPrefix(trimmed, "## ") {
		return r.styles.H2.Render(strings.TrimPrefix(trimmed, "## "))
	}
	if strings.HasPrefix(trimmed, "### ") {
		return r.styles.H3.Render(strings.TrimPrefix(trimmed, "### "))
	}
	if strings.HasPrefix(trimmed, "#### ") {
		return r.styles.H4.Render(strings.TrimPrefix(trimmed, "#### "))
	}

	if strings.HasPrefix(trimmed, "> ") {
		content := strings.TrimPrefix(trimmed, "> ")
		content = r.renderInlineStyles(content)
		return r.styles.Blockquote.Render(content)
	}

	if bullet, content, ok := r.parseBulletList(trimmed); ok {
		content = r.renderInlineStyles(content)
		return r.styles.ListBullet.Render(bullet) + " " + r.styles.ListItem.Render(content)
	}

	if number, content, ok := r.parseNumberedList(trimmed); ok {
		content = r.renderInlineStyles(content)
		return r.styles.ListNumber.Render(number) + " " + r.styles.ListItem.Render(content)
	}

	return r.styles.Text.Render(r.renderInlineStyles(trimmed))
}

func (r *Renderer) isHorizontalRule(line string) bool {
	if len(line) < 3 {
		return false
	}
	dashOnly := strings.Trim(line, "- ")
	if dashOnly == "" && strings.Count(line, "-") >= 3 {
		return true
	}
	asteriskOnly := strings.Trim(line, "* ")
	if asteriskOnly == "" && strings.Count(line, "*") >= 3 {
		return true
	}
	underscoreOnly := strings.Trim(line, "_ ")
	if underscoreOnly == "" && strings.Count(line, "_") >= 3 {
		return true
	}
	return false
}

func (r *Renderer) parseBulletList(line string) (bullet string, content string, ok bool) {
	if strings.HasPrefix(line, "- ") {
		return "•", strings.TrimPrefix(line, "- "), true
	}
	if strings.HasPrefix(line, "* ") {
		return "•", strings.TrimPrefix(line, "* "), true
	}
	if strings.HasPrefix(line, "+ ") {
		return "•", strings.TrimPrefix(line, "+ "), true
	}
	return "", "", false
}

var numberedListRegex = regexp.MustCompile(`^(\d+)\.\s+(.*)$`)

func (r *Renderer) parseNumberedList(line string) (number string, content string, ok bool) {
	matches := numberedListRegex.FindStringSubmatch(line)
	if len(matches) == 3 {
		return matches[1] + ".", matches[2], true
	}
	return "", "", false
}

var (
	boldItalicRegex = regexp.MustCompile(`\*\*\*([^*]+)\*\*\*`)
	boldRegex       = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	italicRegex     = regexp.MustCompile(`\*([^*]+)\*`)
	codeRegex       = regexp.MustCompile("`([^`]+)`")
	linkRegex       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

func (r *Renderer) renderInlineStyles(text string) string {
	text = boldItalicRegex.ReplaceAllStringFunc(text, func(match string) string {
		content := boldItalicRegex.FindStringSubmatch(match)[1]
		return r.styles.BoldItalic.Render(content)
	})

	text = boldRegex.ReplaceAllStringFunc(text, func(match string) string {
		content := boldRegex.FindStringSubmatch(match)[1]
		return r.styles.Bold.Render(content)
	})

	text = italicRegex.ReplaceAllStringFunc(text, func(match string) string {
		content := italicRegex.FindStringSubmatch(match)[1]
		return r.styles.Italic.Render(content)
	})

	text = codeRegex.ReplaceAllStringFunc(text, func(match string) string {
		content := codeRegex.FindStringSubmatch(match)[1]
		return r.styles.Code.Render(content)
	})

	text = linkRegex.ReplaceAllStringFunc(text, func(match string) string {
		matches := linkRegex.FindStringSubmatch(match)
		linkText := matches[1]
		url := matches[2]
		return r.styles.Link.Render(linkText) + " " + r.styles.LinkURL.Render("("+url+")")
	})

	return text
}

func (r *Renderer) renderCodeBlock(lines []string) string {
	content := strings.Join(lines, "\n")
	return r.styles.CodeBlock.Render(content)
}
