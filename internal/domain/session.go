package domain

import "fmt"

// SessionView names a view in the stored session, independent of the UI's view enumeration.
type SessionView string

const (
	SessionViewPATs         SessionView = "pats"
	SessionViewPRList       SessionView = "pr_list"
	SessionViewPRInspect    SessionView = "pr_inspect"
	SessionViewPipelines    SessionView = "pipelines"
	SessionViewPipelineRuns SessionView = "pipeline_runs"
	SessionViewPipelineRun  SessionView = "pipeline_run"
)

// Session is where the UI was when it last quit, so the next start can resume there.
type Session struct {
	View SessionView
	// PRKey is the selected or inspected pull request, as PullRequest.Key returns it.
	PRKey string
	// PipelineKey is the selected or opened pipeline, as Pipeline.FavoriteKey returns it.
	PipelineKey string
	// RunKey is the selected or opened run, as PipelineRun.FavoriteKey returns it.
	RunKey               string
	PRListPaneOpen       bool
	PipelineListPaneOpen bool
	PipelineRunsPaneOpen bool
}

// IsPipelineView reports whether the session resumes in pipeline mode.
func (s Session) IsPipelineView() bool {
	return s.View == SessionViewPipelines || s.View == SessionViewPipelineRuns || s.View == SessionViewPipelineRun
}

// Key identifies a pull request across PATs, which can see the same repository.
func (pr PullRequest) Key() string {
	return fmt.Sprintf("%s/%s#%d", pr.PATID, pr.Repository.FullName, pr.Number)
}
