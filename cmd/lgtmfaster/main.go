package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/editor"
	"github.com/johanforsgren/lgtmfaster/internal/headless"
	"github.com/johanforsgren/lgtmfaster/internal/listing"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/factory"
	"github.com/johanforsgren/lgtmfaster/internal/storage"
	"github.com/johanforsgren/lgtmfaster/internal/ui"
	"github.com/johanforsgren/lgtmfaster/internal/usage"
)

// currentRepository is where -pr looks for the branch to open a pull request from.
const currentRepository = "."

var errListingWithPR = errors.New("-prs and -pl cannot be combined with -pr")

func main() {
	debugLog, prFlags, listFlags := registerFlags(flag.CommandLine)
	flag.Usage = func() { usage.Write(flag.CommandLine.Output(), helpPage()) }
	flag.Parse()
	if err := validateFlags(prFlags, listFlags); err != nil {
		exitWithError("Error: %v\n", err)
	}

	if *debugLog != "" {
		if err := logger.Init(*debugLog); err != nil {
			exitWithError("Error initializing logger: %v\n", err)
		}
		defer logger.Close()
		logger.Log("Application starting with debug logging enabled")
	} else {
		logger.EnsureInit()
		logger.Log("Application starting (in-memory logging only)")
	}

	// Enable global HTTP request/response logging
	// This wraps http.DefaultTransport to log all HTTP traffic
	// http.DefaultTransport = common.NewLoggingTransport(http.DefaultTransport)
	// logger.Log("HTTP request logging enabled globally")

	repo, err := storage.NewLocalRepository()
	if err != nil {
		logger.LogError("INIT", "storage", err)
		exitWithError("Error initializing storage: %v\n", err)
	}
	logger.Log("Storage initialized successfully")

	if listFlags.Requested() {
		printListing(listFlags, repo)
		return
	}

	model := ui.NewModel(repo)
	if prFlags.PR {
		var done bool
		if model, done = openPullRequest(prFlags, repo, model); done {
			return
		}
	}
	runTUI(model)
}

func registerFlags(fs *flag.FlagSet) (*string, *headless.Flags, *listing.Flags) {
	debugLog := fs.String(optionDebug.Name, "", optionDebug.Help)
	return debugLog, headless.Register(fs), listing.Register(fs)
}

func validateFlags(prFlags *headless.Flags, listFlags *listing.Flags) error {
	if err := prFlags.Validate(flag.CommandLine); err != nil {
		return err
	}
	if err := listFlags.Validate(); err != nil {
		return err
	}
	if prFlags.PR && listFlags.Requested() {
		return errListingWithPR
	}
	return nil
}

// printListing prints the pull requests or pipelines the flags ask for; ctrl+c cancels the requests.
func printListing(listFlags *listing.Flags, repo domain.Repository) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	pats, err := repo.ListPATs()
	if err != nil {
		exitWithError("Error reading PATs: %v\n", err)
	}

	if listFlags.Pipelines {
		err = listing.PrintPipelines(ctx, pats, factory.New, os.Stdout)
	} else {
		settings, settingsErr := repo.GetSettings()
		if settingsErr != nil {
			exitWithError("Error reading settings: %v\n", settingsErr)
		}
		err = listing.PrintPRs(ctx, pats, factory.New, settings, os.Stdout)
	}
	if err != nil {
		logger.LogError("LISTING", "accounts", err)
		exitWithError("Error: %v\n", err)
	}
}

// openPullRequest creates the pull request the flags describe, or with -i returns the model that
// opens the form for it; done means there is no TUI to run.
func openPullRequest(prFlags *headless.Flags, repo domain.Repository, model ui.Model) (ui.Model, bool) {
	draft, err := headless.Draft(currentRepository, prFlags, editor.Edit)
	if err != nil {
		exitWithError("Error: %v\n", err)
	}
	pats, err := repo.ListPATs()
	if err != nil {
		exitWithError("Error reading PATs: %v\n", err)
	}

	if prFlags.Interactive {
		pat, err := headless.SelectedPAT(pats, draft)
		if err != nil {
			exitWithError("Error: %v\n", err)
		}
		return model.WithNewPullRequest(draft, pat.ID), false
	}

	if err := headless.Create(context.Background(), pats, factory.New, draft, os.Stdout); err != nil {
		logger.LogError("HEADLESS_PR", draft.SourceBranch, err)
		exitWithError("Error: %v\n", err)
	}
	return model, true
}

func runTUI(model ui.Model) {
	p := tea.NewProgram(model, tea.WithAltScreen())

	logger.Log("Starting TUI application")
	if _, err := p.Run(); err != nil {
		logger.LogError("RUN", "application", err)
		exitWithError("Error running program: %v\n", err)
	}

	logger.Log("Application exiting normally")
}

func exitWithError(format string, err error) {
	fmt.Fprintf(os.Stderr, format, err)
	os.Exit(1)
}
