package main

import (
	"github.com/johanforsgren/lgtmfaster/internal/headless"
	"github.com/johanforsgren/lgtmfaster/internal/listing"
	"github.com/johanforsgren/lgtmfaster/internal/usage"
	"github.com/johanforsgren/lgtmfaster/internal/version"
)

const programName = "lgtmfaster"

var (
	optionDebug = usage.Option{Name: "debug", Value: "file", Help: "write debug logs to file (session logs are always in :logs)"}
	// The flag package handles -h and --help itself; the option is only listed.
	optionHelp = usage.Option{Name: "h", Long: "help", Help: "show this help"}
)

func helpPage() usage.Page {
	return usage.Page{
		Summary: programName + " " + version.Version + " - review pull requests and pipelines from GitHub and Azure DevOps",
		Usages: []string{
			programName + " [-debug <file>]",
			programName + " -prs | -pl",
			programName + " -pr [pull request options]",
		},
		Sections: []usage.Section{
			{Title: "Options:", Options: []usage.Option{optionDebug, optionHelp}},
			{Title: "Listing options (print and exit):", Options: listing.Options()},
			{Title: "Pull request options (from the branch checked out here, pushed to origin):", Options: headless.Options()},
		},
		Examples: []usage.Example{
			{Command: programName, Help: "open the dashboard"},
			{Command: programName + " -prs", Help: "print your open pull requests"},
			{Command: programName + " -pl | grep failed", Help: "find pipelines whose latest run failed"},
			{Command: programName + " -pr", Help: "create a PR from the current branch into " + headless.DefaultTargetBranch},
			{Command: programName + " -pr -t develop -a 1234", Help: "into develop, linked to work item 1234"},
			{Command: programName + " -pr -de", Help: "write the description in $EDITOR first"},
			{Command: programName + ` -pr -i -a "1234,5678"`, Help: "open the TUI's new PR form, prefilled"},
		},
	}
}
