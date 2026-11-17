package github

import (
	"testing"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestConvertWorkflowRun_CompletedRun(t *testing.T) {
	updatedAt := time.Date(2026, 11, 10, 9, 5, 0, 0, time.UTC)
	run := &github.WorkflowRun{
		ID: github.Int64(42), RunNumber: github.Int(7), WorkflowID: github.Int64(3), Name: github.String("CI"),
		Status: github.String(workflowRunCompleted), Conclusion: github.String("failure"),
		HeadBranch: github.String("main"), HeadCommit: &github.HeadCommit{Message: github.String("feature: login\n\nbody")},
		UpdatedAt: &github.Timestamp{Time: updatedAt}, HTMLURL: github.String("https://github.com/o/r/actions/runs/42"),
	}

	converted := convertWorkflowRun(run, "o/r")

	if converted.ID != 42 || converted.Number != "#7" || converted.PipelineName != "CI" || converted.Project != "o/r" {
		t.Errorf("expected the run's identity, got %+v", converted)
	}
	if converted.Status != domain.PipelineStatusFailed || !converted.FinishedAt.Equal(updatedAt) {
		t.Errorf("expected a failed, finished run, got %+v", converted)
	}
	if converted.CommitMessage != "feature: login" || converted.ProviderType != domain.ProviderGitHub {
		t.Errorf("expected the commit summary and provider, got %+v", converted)
	}
}

func TestWorkflowRunStatus_UnfinishedRuns(t *testing.T) {
	for status, want := range map[string]domain.PipelineStatus{
		"queued":      domain.PipelineStatusQueued,
		"in_progress": domain.PipelineStatusRunning,
		"waiting":     domain.PipelineStatusWaitingForApproval,
		"surprising":  domain.PipelineStatusUnknown,
	} {
		run := &github.WorkflowRun{Status: github.String(status)}
		if got := workflowRunStatus(run); got != want {
			t.Errorf("status %q: expected %s, got %s", status, want, got)
		}
		if convertWorkflowRun(run, "o/r").FinishedAt != (time.Time{}) {
			t.Errorf("status %q: expected an unfinished run", status)
		}
	}
}
