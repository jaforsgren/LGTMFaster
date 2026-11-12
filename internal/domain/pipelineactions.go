package domain

// PipelineRetryScope is what a retry runs again.
type PipelineRetryScope string

const (
	// PipelineRetryFailedJobs reruns the failed jobs of a finished run, in the same run.
	PipelineRetryFailedJobs PipelineRetryScope = "failed_jobs"
	// PipelineRetryStage reruns the failed jobs of one stage, in the same run.
	PipelineRetryStage PipelineRetryScope = "stage"
	// PipelineRetryRerun queues a new run of the same commit, branch and parameters.
	PipelineRetryRerun PipelineRetryScope = "rerun"
)

type PipelineRetry struct {
	Run   PipelineRunIdentifier
	Scope PipelineRetryScope
	// Stage is set for PipelineRetryStage.
	Stage PipelineStage
}

// RetryScopes are the retries a run allows, given the stage selected in it (nil when none is):
// failed jobs once a run finished unsuccessfully, a stage that did not succeed once the run
// finished, and a rerun at any time.
func RetryScopes(run PipelineRun, selectedStage *PipelineStage) []PipelineRetryScope {
	var scopes []PipelineRetryScope
	if run.Status.IsFinished() && isRetryable(run.Status) {
		scopes = append(scopes, PipelineRetryFailedJobs)
	}
	if run.Status.IsFinished() && selectedStage != nil && selectedStage.RefName != "" && isRetryable(selectedStage.Status) {
		scopes = append(scopes, PipelineRetryStage)
	}
	return append(scopes, PipelineRetryRerun)
}

func isRetryable(status PipelineStatus) bool {
	switch status {
	case PipelineStatusFailed, PipelineStatusPartiallySucceeded, PipelineStatusCanceled:
		return true
	default:
		return false
	}
}

type PipelineParameterType string

const (
	PipelineParameterString  PipelineParameterType = "string"
	PipelineParameterNumber  PipelineParameterType = "number"
	PipelineParameterBoolean PipelineParameterType = "boolean"
)

// PipelineParameter is a run-time parameter a pipeline can be started with. Values, when set, are
// the only ones allowed.
type PipelineParameter struct {
	Name        string
	DisplayName string
	Type        PipelineParameterType
	Default     string
	Values      []string
}

func (p PipelineParameter) Label() string {
	if p.DisplayName != "" {
		return p.DisplayName
	}
	return p.Name
}

// PipelineTriggerOptions are the branches and parameters a pipeline can be started with.
type PipelineTriggerOptions struct {
	Branches   RepositoryBranches
	Parameters []PipelineParameter
}

type PipelineTrigger struct {
	Pipeline   PipelineIdentifier
	Branch     string
	Parameters map[string]string
}
