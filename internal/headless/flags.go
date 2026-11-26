// Package headless opens pull requests from the command line, for the branch checked out in the
// current repository: directly, or through the TUI's new pull request form.
package headless

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/johanforsgren/lgtmfaster/internal/usage"
)

const (
	flagPR                     = "pr"
	flagInteractive            = "i"
	flagInteractiveLong        = "interactive"
	flagWorkItems              = "a"
	flagWorkItemsLong          = "ado"
	flagDescription            = "d"
	flagDescriptionLong        = "description"
	flagDescriptionFile        = "df"
	flagDescriptionFileLong    = "description-file"
	flagDescriptionEditor      = "de"
	flagDescriptionEditorLong  = "description-editor"
	flagTarget                 = "t"
	flagTargetLong             = "target"
	flagHelpInteractive        = "open the pull request in the TUI's new pull request form instead of creating it"
	flagHelpWorkItems          = "work item numbers to link, e.g. 1234 or \"1234,5678\""
	flagHelpDescription        = "pull request description"
	flagHelpDescriptionFile    = "file holding the pull request description"
	flagHelpDescriptionEditor  = "write the pull request description in $EDITOR; save and quit to submit"
	flagHelpTarget             = "branch to merge into (default " + DefaultTargetBranch + ")"
	flagHelpPR                 = "open a pull request from the current branch"
	valueWorkItems             = "ids"
	valueDescription           = "text"
	valueDescriptionFile       = "file"
	valueTarget                = "branch"
	descriptionSourceSeparator = ", "
)

var (
	optionPR                = usage.Option{Name: flagPR, Help: flagHelpPR}
	optionInteractive       = usage.Option{Name: flagInteractive, Long: flagInteractiveLong, Help: flagHelpInteractive}
	optionWorkItems         = usage.Option{Name: flagWorkItems, Long: flagWorkItemsLong, Value: valueWorkItems, Help: flagHelpWorkItems}
	optionDescription       = usage.Option{Name: flagDescription, Long: flagDescriptionLong, Value: valueDescription, Help: flagHelpDescription}
	optionDescriptionFile   = usage.Option{Name: flagDescriptionFile, Long: flagDescriptionFileLong, Value: valueDescriptionFile, Help: flagHelpDescriptionFile}
	optionDescriptionEditor = usage.Option{Name: flagDescriptionEditor, Long: flagDescriptionEditorLong, Help: flagHelpDescriptionEditor}
	optionTarget            = usage.Option{Name: flagTarget, Long: flagTargetLong, Value: valueTarget, Help: flagHelpTarget}
)

// Options are the pull request flags as the help lists them, -pr first.
func Options() []usage.Option {
	return []usage.Option{
		optionPR, optionInteractive, optionWorkItems, optionDescription,
		optionDescriptionFile, optionDescriptionEditor, optionTarget,
	}
}

// DefaultTargetBranch is what a pull request created without -t merges into.
const DefaultTargetBranch = "main"

var errDescriptionSources = errors.New("give the description one way only: -d, -df or -de")

// Flags are the pull request flags as given on the command line.
type Flags struct {
	PR                bool
	Interactive       bool
	WorkItems         string
	Description       string
	DescriptionFile   string
	DescriptionEditor bool
	Target            string

	set map[string]bool
}

// Register adds the pull request flags to fs, each under its short and its long name.
func Register(fs *flag.FlagSet) *Flags {
	f := &Flags{}
	fs.BoolVar(&f.PR, optionPR.Name, false, optionPR.Help)
	for _, name := range optionInteractive.Names() {
		fs.BoolVar(&f.Interactive, name, false, optionInteractive.Help)
	}
	for _, name := range optionWorkItems.Names() {
		fs.StringVar(&f.WorkItems, name, "", optionWorkItems.Help)
	}
	for _, name := range optionDescription.Names() {
		fs.StringVar(&f.Description, name, "", optionDescription.Help)
	}
	for _, name := range optionDescriptionFile.Names() {
		fs.StringVar(&f.DescriptionFile, name, "", optionDescriptionFile.Help)
	}
	for _, name := range optionDescriptionEditor.Names() {
		fs.BoolVar(&f.DescriptionEditor, name, false, optionDescriptionEditor.Help)
	}
	for _, name := range optionTarget.Names() {
		fs.StringVar(&f.Target, name, "", optionTarget.Help)
	}
	return f
}

// Validate checks the flags once fs has parsed them: the others only go with -pr, and the
// description comes from one place.
func (f *Flags) Validate(fs *flag.FlagSet) error {
	f.set = make(map[string]bool)
	fs.Visit(func(given *flag.Flag) { f.set[given.Name] = true })

	if !f.PR {
		for _, name := range f.prOnlyFlags() {
			if f.set[name] {
				return fmt.Errorf("-%s only goes with -%s", name, flagPR)
			}
		}
		return nil
	}

	var sources []string
	for _, option := range []usage.Option{optionDescription, optionDescriptionFile, optionDescriptionEditor} {
		if f.set[option.Name] || f.set[option.Long] {
			sources = append(sources, "-"+option.Name)
		}
	}
	if len(sources) > 1 {
		return fmt.Errorf("%w (got %s)", errDescriptionSources, strings.Join(sources, descriptionSourceSeparator))
	}
	return nil
}

func (f *Flags) prOnlyFlags() []string {
	var names []string
	for _, option := range Options() {
		if option != optionPR {
			names = append(names, option.Names()...)
		}
	}
	return names
}
