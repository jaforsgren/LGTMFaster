package markdown

import (
	"strings"
	"testing"
)

func TestRenderer_EmptyText(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	result := r.Render("")

	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestRenderer_PlainText(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	result := r.Render("Hello world")

	if !strings.Contains(result, "Hello world") {
		t.Error("expected result to contain 'Hello world'")
	}
}

func TestRenderer_H1(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	result := r.Render("# Heading 1")

	if !strings.Contains(result, "Heading 1") {
		t.Error("expected result to contain 'Heading 1'")
	}
	if strings.Contains(result, "#") {
		t.Error("expected # to be stripped from heading")
	}
}

func TestRenderer_H2(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	result := r.Render("## Heading 2")

	if !strings.Contains(result, "Heading 2") {
		t.Error("expected result to contain 'Heading 2'")
	}
}

func TestRenderer_H3(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	result := r.Render("### Heading 3")

	if !strings.Contains(result, "Heading 3") {
		t.Error("expected result to contain 'Heading 3'")
	}
}

func TestRenderer_BulletList(t *testing.T) {
	r := NewRenderer(DefaultStyles())

	tests := []struct {
		input    string
		contains string
	}{
		{"- Item one", "Item one"},
		{"* Item two", "Item two"},
		{"+ Item three", "Item three"},
	}

	for _, tt := range tests {
		result := r.Render(tt.input)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("expected result to contain %q for input %q", tt.contains, tt.input)
		}
		if !strings.Contains(result, "•") {
			t.Errorf("expected bullet character for input %q", tt.input)
		}
	}
}

func TestRenderer_NumberedList(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	result := r.Render("1. First item\n2. Second item\n3. Third item")

	if !strings.Contains(result, "First item") {
		t.Error("expected result to contain 'First item'")
	}
	if !strings.Contains(result, "1.") {
		t.Error("expected result to contain '1.'")
	}
}

func TestRenderer_HorizontalRule(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	r.SetWidth(40)

	tests := []string{"---", "***", "___", "----", "- - -"}

	for _, input := range tests {
		result := r.Render(input)
		if !strings.Contains(result, "─") {
			t.Errorf("expected horizontal rule for input %q, got %q", input, result)
		}
	}
}

func TestRenderer_Bold(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	result := r.Render("This is **bold** text")

	if !strings.Contains(result, "bold") {
		t.Error("expected result to contain 'bold'")
	}
	if strings.Contains(result, "**") {
		t.Error("expected ** to be stripped")
	}
}

func TestRenderer_Italic(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	result := r.Render("This is *italic* text")

	if !strings.Contains(result, "italic") {
		t.Error("expected result to contain 'italic'")
	}
	if strings.Contains(result, "*italic*") {
		t.Error("expected * to be stripped")
	}
}

func TestRenderer_BoldItalic(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	result := r.Render("This is ***bold italic*** text")

	if !strings.Contains(result, "bold italic") {
		t.Error("expected result to contain 'bold italic'")
	}
	if strings.Contains(result, "***") {
		t.Error("expected *** to be stripped")
	}
}

func TestRenderer_InlineCode(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	result := r.Render("Use `code` here")

	if !strings.Contains(result, "code") {
		t.Error("expected result to contain 'code'")
	}
	if strings.Contains(result, "`code`") {
		t.Error("expected backticks to be stripped")
	}
}

func TestRenderer_CodeBlock(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	input := "```\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"
	result := r.Render(input)

	if !strings.Contains(result, "func main()") {
		t.Error("expected result to contain code block content")
	}
	if strings.Contains(result, "```") {
		t.Error("expected ``` to be stripped")
	}
}

func TestRenderer_Link(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	result := r.Render("Check out [this link](https://example.com)")

	if !strings.Contains(result, "this link") {
		t.Error("expected result to contain link text")
	}
	if !strings.Contains(result, "example.com") {
		t.Error("expected result to contain URL")
	}
}

func TestRenderer_Blockquote(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	result := r.Render("> This is a quote")

	if !strings.Contains(result, "This is a quote") {
		t.Error("expected result to contain quote text")
	}
}

func TestRenderer_MixedContent(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	input := `# Title

This is a **description** with some *emphasis*.

## Features

- Feature one
- Feature two
- Feature three

---

Check the [docs](https://docs.example.com) for more info.`

	result := r.Render(input)

	checks := []string{
		"Title",
		"description",
		"emphasis",
		"Features",
		"Feature one",
		"•",
		"─",
		"docs",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("expected result to contain %q", check)
		}
	}
}

func TestRenderer_SetWidth(t *testing.T) {
	r := NewRenderer(DefaultStyles())
	r.SetWidth(50)

	result := r.Render("---")

	runeCount := strings.Count(result, "─")
	if runeCount < 40 {
		t.Errorf("expected horizontal rule to be at least 40 chars, got %d", runeCount)
	}
}

func TestDefaultStyles_ReturnsValidStyles(t *testing.T) {
	styles := DefaultStyles()

	testText := "test"
	h1Rendered := styles.H1.Render(testText)
	if h1Rendered == "" {
		t.Error("expected H1 style to render text")
	}

	textRendered := styles.Text.Render(testText)
	if textRendered == "" {
		t.Error("expected Text style to render text")
	}
}
