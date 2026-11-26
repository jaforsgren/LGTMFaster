package main

import (
	"flag"
	"strings"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/usage"
)

func TestHelpPage_ListsEveryRegisteredFlag(t *testing.T) {
	fs := flag.NewFlagSet(programName, flag.ContinueOnError)
	registerFlags(fs)
	var help strings.Builder
	usage.Write(&help, helpPage())

	fs.VisitAll(func(registered *flag.Flag) {
		if !strings.Contains(help.String(), "-"+registered.Name) {
			t.Errorf("-%s is registered but missing from the help", registered.Name)
		}
	})
}
