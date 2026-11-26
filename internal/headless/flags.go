// Package headless opens pull requests from the command line, for the branch checked out in the
// current repository: directly, or through the TUI's new pull request form.
package headless

import (
	"errors"
	"flag"
	"fmt"
	"strings"
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
	descriptionSourceSeparator = ", "
)

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
	fs.BoolVar(&f.PR, flagPR, false, flagHelpPR)
	for _, name := range []string{flagInteractive, flagInteractiveLong} {
		fs.BoolVar(&f.Interactive, name, false, flagHelpInteractive)
	}
	for _, name := range []string{flagWorkItems, flagWorkItemsLong} {
		fs.StringVar(&f.WorkItems, name, "", flagHelpWorkItems)
	}
	for _, name := range []string{flagDescription, flagDescriptionLong} {
		fs.StringVar(&f.Description, name, "", flagHelpDescription)
	}
	for _, name := range []string{flagDescriptionFile, flagDescriptionFileLong} {
		fs.StringVar(&f.DescriptionFile, name, "", flagHelpDescriptionFile)
	}
	for _, name := range []string{flagDescriptionEditor, flagDescriptionEditorLong} {
		fs.BoolVar(&f.DescriptionEditor, name, false, flagHelpDescriptionEditor)
	}
	for _, name := range []string{flagTarget, flagTargetLong} {
		fs.StringVar(&f.Target, name, "", flagHelpTarget)
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
	for _, names := range [][]string{
		{flagDescription, flagDescriptionLong},
		{flagDescriptionFile, flagDescriptionFileLong},
		{flagDescriptionEditor, flagDescriptionEditorLong},
	} {
		if f.set[names[0]] || f.set[names[1]] {
			sources = append(sources, "-"+names[0])
		}
	}
	if len(sources) > 1 {
		return fmt.Errorf("%w (got %s)", errDescriptionSources, strings.Join(sources, descriptionSourceSeparator))
	}
	return nil
}

func (f *Flags) prOnlyFlags() []string {
	return []string{
		flagInteractive, flagInteractiveLong, flagWorkItems, flagWorkItemsLong,
		flagDescription, flagDescriptionLong, flagDescriptionFile, flagDescriptionFileLong,
		flagDescriptionEditor, flagDescriptionEditorLong, flagTarget, flagTargetLong,
	}
}
