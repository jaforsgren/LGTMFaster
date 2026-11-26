// Package listing prints the selected accounts' open pull requests or pipelines to stdout, for
// use outside the TUI.
package listing

import (
	"errors"
	"flag"

	"github.com/johanforsgren/lgtmfaster/internal/usage"
)

const (
	flagPRs           = "prs"
	flagPipelines     = "pl"
	flagPipelinesLong = "pipelines"
	flagHelpPRs       = "print the open pull requests of the selected PATs"
	flagHelpPipelines = "print the pipelines of the selected PATs with their latest run"
)

var (
	optionPRs       = usage.Option{Name: flagPRs, Help: flagHelpPRs}
	optionPipelines = usage.Option{Name: flagPipelines, Long: flagPipelinesLong, Help: flagHelpPipelines}
)

var errBothListings = errors.New("give one of -prs or -pl, not both")

// Flags are the listing flags as given on the command line.
type Flags struct {
	PRs       bool
	Pipelines bool
}

// Options are the listing flags as the help lists them.
func Options() []usage.Option {
	return []usage.Option{optionPRs, optionPipelines}
}

// Register adds the listing flags to fs, each under its short and its long name.
func Register(fs *flag.FlagSet) *Flags {
	f := &Flags{}
	fs.BoolVar(&f.PRs, optionPRs.Name, false, optionPRs.Help)
	for _, name := range optionPipelines.Names() {
		fs.BoolVar(&f.Pipelines, name, false, optionPipelines.Help)
	}
	return f
}

// Requested reports whether a listing was asked for.
func (f *Flags) Requested() bool {
	return f.PRs || f.Pipelines
}

// Validate checks that one listing at most was asked for.
func (f *Flags) Validate() error {
	if f.PRs && f.Pipelines {
		return errBothListings
	}
	return nil
}
