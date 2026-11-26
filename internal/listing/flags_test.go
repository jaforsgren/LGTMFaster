package listing

import (
	"errors"
	"flag"
	"io"
	"testing"
)

func parseFlags(t *testing.T, args ...string) *Flags {
	t.Helper()
	fs := flag.NewFlagSet("lgtmfaster", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	f := Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestFlags_ShortAndLongPipelinesNamesSetTheSameField(t *testing.T) {
	for _, arg := range []string{"-pl", "--pipelines"} {
		if f := parseFlags(t, arg); !f.Pipelines || !f.Requested() {
			t.Errorf("%s: expected pipelines requested, got %+v", arg, f)
		}
	}
}

func TestFlags_NothingRequestedByDefault(t *testing.T) {
	if f := parseFlags(t); f.Requested() {
		t.Errorf("expected no listing requested, got %+v", f)
	}
}

func TestFlags_OneListingAtATime(t *testing.T) {
	if err := parseFlags(t, "-prs", "-pl").Validate(); !errors.Is(err, errBothListings) {
		t.Errorf("expected both listings refused, got %v", err)
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
