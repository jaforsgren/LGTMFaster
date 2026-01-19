package markdown

import "github.com/charmbracelet/lipgloss"

type Styles struct {
	H1         lipgloss.Style
	H2         lipgloss.Style
	H3         lipgloss.Style
	H4         lipgloss.Style
	Text       lipgloss.Style
	Bold       lipgloss.Style
	Italic     lipgloss.Style
	BoldItalic lipgloss.Style
	Code       lipgloss.Style
	CodeBlock  lipgloss.Style
	Link       lipgloss.Style
	LinkURL    lipgloss.Style
	ListBullet lipgloss.Style
	ListNumber lipgloss.Style
	ListItem   lipgloss.Style
	HRule      lipgloss.Style
	Blockquote lipgloss.Style
}

func DefaultStyles() Styles {
	purple := lipgloss.Color("#7C3AED")
	cyan := lipgloss.Color("#06B6D4")
	gray := lipgloss.Color("#6B7280")
	lightGray := lipgloss.Color("#F9FAFB")
	orange := lipgloss.Color("#F59E0B")
	green := lipgloss.Color("#10B981")

	return Styles{
		H1: lipgloss.NewStyle().
			Foreground(purple).
			Bold(true).
			MarginTop(1).
			MarginBottom(1),

		H2: lipgloss.NewStyle().
			Foreground(purple).
			Bold(true).
			MarginTop(1),

		H3: lipgloss.NewStyle().
			Foreground(purple).
			Bold(true),

		H4: lipgloss.NewStyle().
			Foreground(purple),

		Text: lipgloss.NewStyle().
			Foreground(lightGray),

		Bold: lipgloss.NewStyle().
			Foreground(lightGray).
			Bold(true),

		Italic: lipgloss.NewStyle().
			Foreground(lightGray).
			Italic(true),

		BoldItalic: lipgloss.NewStyle().
			Foreground(lightGray).
			Bold(true).
			Italic(true),

		Code: lipgloss.NewStyle().
			Foreground(orange).
			Background(lipgloss.Color("#1F2937")),

		CodeBlock: lipgloss.NewStyle().
			Foreground(orange).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 1),

		Link: lipgloss.NewStyle().
			Foreground(cyan).
			Underline(true),

		LinkURL: lipgloss.NewStyle().
			Foreground(gray),

		ListBullet: lipgloss.NewStyle().
			Foreground(green),

		ListNumber: lipgloss.NewStyle().
			Foreground(green),

		ListItem: lipgloss.NewStyle().
			Foreground(lightGray),

		HRule: lipgloss.NewStyle().
			Foreground(gray),

		Blockquote: lipgloss.NewStyle().
			Foreground(gray).
			Italic(true).
			PaddingLeft(2).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(gray),
	}
}
