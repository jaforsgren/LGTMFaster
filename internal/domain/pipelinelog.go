package domain

import (
	"strings"
	"time"
)

const pipelineLogPathSeparator = " › "

// Log lines carry these markers when a task reports an issue (Azure DevOps "##[error]", GitHub
// Actions "::error").
var (
	errorLogMarkers   = []string{"##[error]", "::error"}
	warningLogMarkers = []string{"##[warning]", "::warning"}
)

// PipelineLog is the log of one step of a run.
type PipelineLog struct {
	ID         int
	Name       string
	Stage      string
	Job        string
	Status     PipelineStatus
	StartedAt  time.Time
	FinishedAt time.Time
}

// Path names the step by its stage, job and task, leaving out the levels it does not have.
func (l PipelineLog) Path() string {
	var segments []string
	for _, segment := range []string{l.Stage, l.Job, l.Name} {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	return strings.Join(segments, pipelineLogPathSeparator)
}

type LogLineLevel int

const (
	LogLineInfo LogLineLevel = iota
	LogLineWarning
	LogLineError
)

func ClassifyLogLine(line string) LogLineLevel {
	switch {
	case containsAny(line, errorLogMarkers):
		return LogLineError
	case containsAny(line, warningLogMarkers):
		return LogLineWarning
	default:
		return LogLineInfo
	}
}

func containsAny(s string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

// DefaultLogIndex is the step to open the logs on: within the given stage when it has logs, the
// first failed step, else the first running one, else the stage's first step. Without a stage
// (or without logs for it) it is the run's first failed or running step, else its last one.
// It is -1 when there are no logs.
func DefaultLogIndex(logs []PipelineLog, stage string) int {
	if len(logs) == 0 {
		return -1
	}
	candidates, fallback := stageLogIndices(logs, stage), 0
	if len(candidates) > 0 {
		fallback = candidates[0]
	} else {
		for i := range logs {
			candidates = append(candidates, i)
		}
		fallback = len(logs) - 1
	}

	for _, i := range candidates {
		if logs[i].Status == PipelineStatusFailed {
			return i
		}
	}
	for _, i := range candidates {
		if logs[i].Status == PipelineStatusRunning {
			return i
		}
	}
	return fallback
}

func stageLogIndices(logs []PipelineLog, stage string) []int {
	var indices []int
	for i, log := range logs {
		if stage != "" && log.Stage == stage {
			indices = append(indices, i)
		}
	}
	return indices
}

// FollowedLogIndex is the step a followed log moves to: the last running step, else the last one.
func FollowedLogIndex(logs []PipelineLog) int {
	for i := len(logs) - 1; i >= 0; i-- {
		if logs[i].Status == PipelineStatusRunning {
			return i
		}
	}
	return len(logs) - 1
}
