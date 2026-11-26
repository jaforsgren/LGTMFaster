package usage

import (
	"strings"
	"testing"
)

func TestOption_NamesIncludeLongOnlyWhenSet(t *testing.T) {
	if got := (Option{Name: "pr"}).Names(); len(got) != 1 || got[0] != "pr" {
		t.Errorf("expected [pr], got %v", got)
	}
	if got := (Option{Name: "t", Long: "target"}).Names(); len(got) != 2 || got[1] != "target" {
		t.Errorf("expected [t target], got %v", got)
	}
}

func TestWrite_LabelsShowShortLongAndValue(t *testing.T) {
	var b strings.Builder
	Write(&b, Page{Sections: []Section{{Title: "Options:", Options: []Option{
		{Name: "t", Long: "target", Value: "branch", Help: "branch to merge into"},
	}}}})

	if !strings.Contains(b.String(), "  -t, --target <branch>   branch to merge into\n") {
		t.Errorf("unexpected option row in:\n%s", b.String())
	}
}

func TestWrite_AlignsHelpAcrossSections(t *testing.T) {
	var b strings.Builder
	Write(&b, Page{Sections: []Section{
		{Title: "A:", Options: []Option{{Name: "x", Help: "first"}}},
		{Title: "B:", Options: []Option{{Name: "longer", Help: "second"}}},
	}})

	out := b.String()
	if strings.Index(lineWith(out, "first"), "first") != strings.Index(lineWith(out, "second"), "second") {
		t.Errorf("help columns not aligned:\n%s", out)
	}
}

func TestWrite_ListsUsagesAndExamples(t *testing.T) {
	var b strings.Builder
	Write(&b, Page{
		Summary:  "tool - does things",
		Usages:   []string{"tool [options]"},
		Examples: []Example{{Command: "tool -pr", Help: "open a PR"}},
	})

	out := b.String()
	for _, want := range []string{"tool - does things\n", headingUsage + "\n  tool [options]\n", headingExamples + "\n  tool -pr   open a PR\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in:\n%s", want, out)
		}
	}
}

func TestWrite_LeavesOutExamplesHeadingWithoutExamples(t *testing.T) {
	var b strings.Builder
	Write(&b, Page{Summary: "tool"})

	if strings.Contains(b.String(), headingExamples) {
		t.Errorf("unexpected examples heading in:\n%s", b.String())
	}
}

func lineWith(text, needle string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}
