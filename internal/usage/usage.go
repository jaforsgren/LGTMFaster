// Package usage renders the command line help: usage lines, option sections and examples.
package usage

import (
	"fmt"
	"io"
	"strings"
)

const (
	shortPrefix     = "-"
	longPrefix      = "--"
	nameSeparator   = ", "
	indent          = "  "
	columnGap       = "   "
	headingUsage    = "Usage:"
	headingExamples = "Examples:"
)

// Option is one command line flag as the help lists it. Long, when set, is an alias shown with
// "--"; Value names the argument the flag takes, if any.
type Option struct {
	Name  string
	Long  string
	Value string
	Help  string
}

// Names are the flag names the option is registered under.
func (o Option) Names() []string {
	if o.Long == "" {
		return []string{o.Name}
	}
	return []string{o.Name, o.Long}
}

func (o Option) label() string {
	label := shortPrefix + o.Name
	if o.Long != "" {
		label += nameSeparator + longPrefix + o.Long
	}
	if o.Value != "" {
		label += " <" + o.Value + ">"
	}
	return label
}

// Section is a titled group of options.
type Section struct {
	Title   string
	Options []Option
}

// Example is a command line with what it does.
type Example struct {
	Command string
	Help    string
}

// Page is the whole help text.
type Page struct {
	Summary  string
	Usages   []string
	Sections []Section
	Examples []Example
}

// Write renders the page, aligning option help and example help in their own columns.
func Write(w io.Writer, page Page) {
	var b strings.Builder
	b.WriteString(page.Summary + "\n\n" + headingUsage + "\n")
	for _, line := range page.Usages {
		b.WriteString(indent + line + "\n")
	}

	labelWidth := 0
	for _, section := range page.Sections {
		for _, option := range section.Options {
			labelWidth = max(labelWidth, len(option.label()))
		}
	}
	for _, section := range page.Sections {
		b.WriteString("\n" + section.Title + "\n")
		for _, option := range section.Options {
			writeRow(&b, option.label(), option.Help, labelWidth)
		}
	}

	if len(page.Examples) > 0 {
		commandWidth := 0
		for _, example := range page.Examples {
			commandWidth = max(commandWidth, len(example.Command))
		}
		b.WriteString("\n" + headingExamples + "\n")
		for _, example := range page.Examples {
			writeRow(&b, example.Command, example.Help, commandWidth)
		}
	}
	fmt.Fprint(w, b.String())
}

func writeRow(b *strings.Builder, left, right string, width int) {
	b.WriteString(indent + left + strings.Repeat(" ", width-len(left)) + columnGap + right + "\n")
}
