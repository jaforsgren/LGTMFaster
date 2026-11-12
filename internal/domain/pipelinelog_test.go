package domain

import (
	"slices"
	"testing"
)

func TestPipelineLogPath_SkipsMissingLevels(t *testing.T) {
	tests := map[string]struct {
		log  PipelineLog
		want string
	}{
		"all levels": {PipelineLog{Stage: "Build", Job: "Linux", Name: "Test"}, "Build › Linux › Test"},
		"no stage":   {PipelineLog{Job: "Linux", Name: "Test"}, "Linux › Test"},
		"task only":  {PipelineLog{Name: "Test"}, "Test"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := tt.log.Path(); got != tt.want {
				t.Errorf("Path() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyLogLine(t *testing.T) {
	tests := map[string]LogLineLevel{
		"##[error]Build failed":          LogLineError,
		"::error file=a.go::bad":         LogLineError,
		"##[warning]Deprecated task":     LogLineWarning,
		"::warning::slow":                LogLineWarning,
		"error: plain text is info":      LogLineInfo,
		"Restored packages in 2 seconds": LogLineInfo,
	}
	for line, want := range tests {
		if got := ClassifyLogLine(line); got != want {
			t.Errorf("ClassifyLogLine(%q) = %v, want %v", line, got, want)
		}
	}
}

func TestDefaultLogIndex(t *testing.T) {
	logs := []PipelineLog{
		{Stage: "Build", Status: PipelineStatusSucceeded},
		{Stage: "Build", Status: PipelineStatusFailed},
		{Stage: "Deploy", Status: PipelineStatusSucceeded},
		{Stage: "Deploy", Status: PipelineStatusRunning},
	}
	tests := map[string]struct {
		logs  []PipelineLog
		stage string
		want  int
	}{
		"no logs":                       {nil, "", -1},
		"run: first failed":             {logs, "", 1},
		"stage: running":                {logs, "Deploy", 3},
		"stage without logs: whole run": {logs, "Test", 1},
		"stage all succeeded: first":    {logs[2:3], "Deploy", 0},
		"run all succeeded: last":       {[]PipelineLog{{}, {}}, "", 1},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := DefaultLogIndex(tt.logs, tt.stage); got != tt.want {
				t.Errorf("DefaultLogIndex() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFollowedLogIndex(t *testing.T) {
	running := []PipelineLog{{Status: PipelineStatusRunning}, {Status: PipelineStatusRunning}, {Status: PipelineStatusQueued}}
	if got := FollowedLogIndex(running); got != 1 {
		t.Errorf("expected the last running step, got %d", got)
	}
	if got := FollowedLogIndex([]PipelineLog{{}, {}}); got != 1 {
		t.Errorf("expected the last step when none runs, got %d", got)
	}
	if got := FollowedLogIndex(nil); got != -1 {
		t.Errorf("expected -1 without logs, got %d", got)
	}
}

func TestRetryScopes(t *testing.T) {
	failedStage := &PipelineStage{RefName: "build", Status: PipelineStatusFailed}
	tests := map[string]struct {
		run   PipelineRun
		stage *PipelineStage
		want  []PipelineRetryScope
	}{
		"running run: rerun only": {PipelineRun{Status: PipelineStatusRunning}, failedStage, []PipelineRetryScope{PipelineRetryRerun}},
		"succeeded run: rerun only": {PipelineRun{Status: PipelineStatusSucceeded}, nil,
			[]PipelineRetryScope{PipelineRetryRerun}},
		"failed run with failed stage": {PipelineRun{Status: PipelineStatusFailed}, failedStage,
			[]PipelineRetryScope{PipelineRetryFailedJobs, PipelineRetryStage, PipelineRetryRerun}},
		"failed run, stage succeeded": {PipelineRun{Status: PipelineStatusFailed}, &PipelineStage{RefName: "a", Status: PipelineStatusSucceeded},
			[]PipelineRetryScope{PipelineRetryFailedJobs, PipelineRetryRerun}},
		"canceled run, stage without ref name": {PipelineRun{Status: PipelineStatusCanceled}, &PipelineStage{Status: PipelineStatusCanceled},
			[]PipelineRetryScope{PipelineRetryFailedJobs, PipelineRetryRerun}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := RetryScopes(tt.run, tt.stage); !slices.Equal(got, tt.want) {
				t.Errorf("RetryScopes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipelineParameterLabel_PrefersDisplayName(t *testing.T) {
	if got := (PipelineParameter{Name: "env", DisplayName: "Environment"}).Label(); got != "Environment" {
		t.Errorf("expected display name, got %q", got)
	}
	if got := (PipelineParameter{Name: "env"}).Label(); got != "env" {
		t.Errorf("expected name, got %q", got)
	}
}
