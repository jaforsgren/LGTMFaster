package common

import (
	"html"
	"regexp"
	"strings"
)

const (
	listItemPrefix   = "- "
	nonBreakingSpace = " "
)

var (
	lineBreakTags      = regexp.MustCompile(`(?i)<br\s*/?>|</(p|div|li|tr|h[1-6]|pre|blockquote)>`)
	listItemTags       = regexp.MustCompile(`(?i)<li[^>]*>`)
	anyTag             = regexp.MustCompile(`<[^>]*>`)
	repeatedBlankLines = regexp.MustCompile(`\n{3,}`)
)

// HTMLToText turns rich text, like Azure DevOps work item descriptions, into plain text: block
// elements become line breaks, list items "- " lines, and the remaining tags are dropped.
func HTMLToText(rich string) string {
	text := lineBreakTags.ReplaceAllString(rich, "\n")
	text = listItemTags.ReplaceAllString(text, listItemPrefix)
	text = anyTag.ReplaceAllString(text, "")
	text = strings.ReplaceAll(html.UnescapeString(text), nonBreakingSpace, " ")

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}
	text = repeatedBlankLines.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")
	return strings.TrimSpace(text)
}
