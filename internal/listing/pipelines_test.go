package listing

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestPrintPipelines_SortsByProjectRepositoryAndName(t *testing.T) {
	pipelines := []domain.Pipeline{
		{Name: "deploy", Project: "b", Repository: "api", URL: "https://example.test/deploy"},
		{Name: "ci", Folder: `\team`, Project: "a", Repository: "web", LatestRun: &domain.PipelineRun{
			Status: domain.PipelineStatusWaitingForApproval, SourceBranch: "main", URL: "https://example.test/run/1",
		}},
		{Name: "build", Project: "a", Repository: "web"},
	}
	factory := factoryOf(map[string]domain.Provider{"ado": pipelineListingProvider{pipelines: pipelines}})
	var out bytes.Buffer

	if err := PrintPipelines(context.Background(), []domain.PAT{selectedPAT("ado")}, factory, &out); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 4 || !strings.HasPrefix(lines[0], "PROJECT") {
		t.Fatalf("expected a header and three rows, got:\n%s", out.String())
	}
	for i, want := range [][]string{
		{"build", neverRunText},
		{`team\ci`, "waiting for approval", "main", "https://example.test/run/1"},
		{"deploy", neverRunText, "https://example.test/deploy"},
	} {
		for _, field := range want {
			if !strings.Contains(lines[i+1], field) {
				t.Errorf("row %d: expected %q in %q", i+1, field, lines[i+1])
			}
		}
	}
}

func TestPrintPipelines_SkipsPATsWithoutPipelines(t *testing.T) {
	factory := factoryOf(map[string]domain.Provider{
		"gh":  listingProvider{},
		"ado": pipelineListingProvider{pipelines: []domain.Pipeline{{Name: "ci", Project: "a"}}},
	})
	var out bytes.Buffer

	if err := PrintPipelines(context.Background(), []domain.PAT{selectedPAT("gh"), selectedPAT("ado")}, factory, &out); err != nil {
		t.Errorf("expected the GitHub PAT skipped, got %v", err)
	}
	if !strings.Contains(out.String(), "ci") {
		t.Errorf("expected the pipeline printed, got:\n%s", out.String())
	}
}

func TestPrintPipelines_NeedsAPATWithPipelines(t *testing.T) {
	factory := factoryOf(map[string]domain.Provider{"gh": listingProvider{}})

	err := PrintPipelines(context.Background(), []domain.PAT{selectedPAT("gh")}, factory, &bytes.Buffer{})

	if !errors.Is(err, errNoPipelinePATs) {
		t.Errorf("expected no pipeline PATs refused, got %v", err)
	}
}

func TestPrintPipelines_ReportsAFailingPAT(t *testing.T) {
	factory := factoryOf(map[string]domain.Provider{"ado": pipelineListingProvider{listingProvider: listingProvider{err: errListFailed}}})
	var out bytes.Buffer

	err := PrintPipelines(context.Background(), []domain.PAT{selectedPAT("ado")}, factory, &out)

	if !errors.Is(err, errListFailed) || !strings.Contains(out.String(), noPipelinesText) {
		t.Errorf("expected the failure reported after an empty listing, got %v and:\n%s", err, out.String())
	}
}
