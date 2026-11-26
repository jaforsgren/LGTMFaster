package headless

import (
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

func parseFlags(t *testing.T, args ...string) (*Flags, error) {
	t.Helper()
	fs := flag.NewFlagSet("lgtmfaster", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	return f, f.Validate(fs)
}

func TestFlags_ShortAndLongNamesSetTheSameField(t *testing.T) {
	short, err := parseFlags(t, "-pr", "-i", "-a", "1234", "-d", "text", "-t", "develop")
	if err != nil {
		t.Fatal(err)
	}
	long, err := parseFlags(t, "-pr", "--interactive", "--ado", "1234", "--description", "text", "--target", "develop")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []*Flags{short, long} {
		if !f.Interactive || f.WorkItems != "1234" || f.Description != "text" || f.Target != "develop" {
			t.Errorf("unexpected flags %+v", f)
		}
	}
}

func TestFlags_OthersNeedPR(t *testing.T) {
	_, err := parseFlags(t, "-t", "develop")

	if err == nil || !strings.Contains(err.Error(), "-t only goes with -pr") {
		t.Errorf("expected -t refused without -pr, got %v", err)
	}
}

func TestFlags_DescriptionComesFromOnePlace(t *testing.T) {
	_, err := parseFlags(t, "-pr", "-df", "notes.md", "--description-editor")

	if !errors.Is(err, errDescriptionSources) {
		t.Errorf("expected two description sources refused, got %v", err)
	}
}

func TestOptions_ListEveryRegisteredFlag(t *testing.T) {
	fs := flag.NewFlagSet("lgtmfaster", flag.ContinueOnError)
	Register(fs)
	listed := make(map[string]bool)
	for _, option := range Options() {
		for _, name := range option.Names() {
			listed[name] = true
		}
	}

	fs.VisitAll(func(registered *flag.Flag) {
		if !listed[registered.Name] {
			t.Errorf("-%s is registered but missing from Options", registered.Name)
		}
	})
}
