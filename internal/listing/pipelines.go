package listing

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"text/tabwriter"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	noPipelinesText   = "No pipelines."
	neverRunText      = "never run"
	pipelineRowFormat = "%s\t%s\t%s\t%s\t%s\t%s\t%s\n"
	pipelineHeader    = "PROJECT\tREPOSITORY\tPIPELINE\tLATEST RUN\tBRANCH\tWHEN\tURL\n"
	runTimeLayout     = "2006-01-02 15:04"
)

var (
	errPipelinesUnsupported = errors.New("the provider has no pipelines")
	errNoPipelinePATs       = errors.New("none of the selected PATs has pipelines (only Azure DevOps does)")
)

// PrintPipelines prints the pipelines of the selected PATs whose providers have pipelines, with
// their latest run, sorted by project, repository and name. What the other PATs listed is printed
// even when one fails.
func PrintPipelines(ctx context.Context, pats []domain.PAT, newProvider domain.ProviderFactory, out io.Writer) error {
	listings, err := listSelectedAccounts(ctx, pats, newProvider, func(ctx context.Context, _ domain.PAT, provider domain.Provider) ([]domain.Pipeline, error) {
		pipelineProvider, ok := provider.(domain.PipelineProvider)
		if !ok {
			return nil, errPipelinesUnsupported
		}
		return pipelineProvider.ListPipelines(ctx)
	})
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(listings, func(listing accountListing[domain.Pipeline]) bool {
		return !errors.Is(listing.err, errPipelinesUnsupported)
	}) {
		return errNoPipelinePATs
	}

	var pipelines []domain.Pipeline
	for _, listing := range listings {
		pipelines = append(pipelines, listing.items...)
	}
	writePipelines(out, pipelines)
	return accountErrors(listings, errPipelinesUnsupported)
}

func writePipelines(out io.Writer, pipelines []domain.Pipeline) {
	if len(pipelines) == 0 {
		fmt.Fprintln(out, noPipelinesText)
		return
	}
	slices.SortStableFunc(pipelines, func(a, b domain.Pipeline) int {
		return cmp.Or(cmp.Compare(a.Project, b.Project), cmp.Compare(a.Repository, b.Repository), cmp.Compare(a.DisplayName(), b.DisplayName()))
	})

	table := tabwriter.NewWriter(out, tabMinWidth, tabWidth, tabPadding, tabPadChar, 0)
	fmt.Fprint(table, pipelineHeader)
	for _, pipeline := range pipelines {
		status, branch, when, url := neverRunText, "", "", pipeline.URL
		if run := pipeline.LatestRun; run != nil {
			status, branch, when = run.Status.Label(), run.SourceBranch, runTime(*run)
			url = cmp.Or(run.URL, pipeline.URL)
		}
		fmt.Fprintf(table, pipelineRowFormat, pipeline.Project, pipeline.Repository, pipeline.DisplayName(), status, branch, when, url)
	}
	table.Flush()
}

// runTime is when the run finished, else started, else was queued.
func runTime(run domain.PipelineRun) string {
	for _, at := range []time.Time{run.FinishedAt, run.StartedAt, run.QueuedAt} {
		if !at.IsZero() {
			return at.Local().Format(runTimeLayout)
		}
	}
	return ""
}
