package azuredevops

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

const testPipelineYAML = `
trigger: none
parameters:
  - name: environment
    displayName: Environment
    type: string
    default: dev
    values: [dev, prod]
  - name: dryRun
    type: boolean
    default: true
  - name: retries
    type: number
    default: 3
  - name: note
  - name: matrix
    type: object
    default:
      linux: ubuntu
  - name: steps
    type: stepList
    default: []
steps:
  - script: echo hi
`

func taskRecord(id, parentID uuid.UUID, name string, order, logID int, state build.TimelineRecordState, result *build.TaskResult) build.TimelineRecord {
	record := newTimelineRecord(id, &parentID, timelineRecordTypeTask, name, order, state, result)
	record.Log = &build.BuildLogReference{Id: &logID}
	return record
}

func TestListPipelineRunLogs_OrdersTasksByTheirAncestors(t *testing.T) {
	completed := build.TimelineRecordStateValues.Completed
	succeeded, failed := build.TaskResultValues.Succeeded, build.TaskResultValues.Failed
	deploy, buildStage := uuid.New(), uuid.New()
	deployJob, buildPhase, buildJob := uuid.New(), uuid.New(), uuid.New()

	records := []build.TimelineRecord{
		newTimelineRecord(deploy, nil, timelineRecordTypeStage, "Deploy", 2, completed, &succeeded),
		newTimelineRecord(buildStage, nil, timelineRecordTypeStage, "Build", 1, completed, &failed),
		newTimelineRecord(deployJob, &deploy, timelineRecordTypeJob, "Release", 1, completed, &succeeded),
		newTimelineRecord(buildPhase, &buildStage, "Phase", "Build", 1, completed, &failed),
		newTimelineRecord(buildJob, &buildPhase, timelineRecordTypeJob, "Linux", 1, completed, &failed),
		taskRecord(uuid.New(), deployJob, "Upload", 1, 30, completed, &succeeded),
		taskRecord(uuid.New(), buildJob, "Test", 2, 12, completed, &failed),
		taskRecord(uuid.New(), buildJob, "Restore", 1, 11, completed, &succeeded),
		newTimelineRecord(uuid.New(), &buildJob, timelineRecordTypeTask, "Skipped without log", 3, completed, nil),
	}
	provider := newPipelineTestProvider(&mockBuildClient{timeline: &build.Timeline{Records: &records}}, nil)

	logs, err := provider.ListPipelineRunLogs(context.Background(), domain.PipelineRunIdentifier{Project: testProject, RunID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var paths []string
	for _, log := range logs {
		paths = append(paths, log.Path())
	}
	want := []string{"Build › Linux › Restore", "Build › Linux › Test", "Deploy › Release › Upload"}
	if !slices.Equal(paths, want) {
		t.Fatalf("expected %v, got %v", want, paths)
	}
	if logs[1].ID != 12 || logs[1].Status != domain.PipelineStatusFailed {
		t.Errorf("unexpected failed log: %+v", logs[1])
	}
}

func TestListPipelineRunLogs_APIError(t *testing.T) {
	provider := newPipelineTestProvider(&mockBuildClient{err: errors.New("boom")}, nil)

	if _, err := provider.ListPipelineRunLogs(context.Background(), domain.PipelineRunIdentifier{Project: testProject, RunID: 1}); err == nil {
		t.Error("expected error")
	}
}

func TestGetPipelineLogLines_StripsTimestampsAndReadsAfterOffset(t *testing.T) {
	mock := &mockBuildClient{logLines: []string{
		"\uFEFF2026-01-02T03:04:05.1234567Z ##[section]Starting: Test",
		"2026-01-02T03:04:06.0000000Z ##[error]boom",
		"no timestamp",
	}}
	provider := newPipelineTestProvider(mock, nil)
	run := domain.PipelineRunIdentifier{Project: testProject, RunID: 1}

	lines, err := provider.GetPipelineLogLines(context.Background(), run, 12, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"##[section]Starting: Test", "##[error]boom", "no timestamp"}
	if !slices.Equal(lines, want) {
		t.Errorf("expected %q, got %q", want, lines)
	}
	if mock.lastLogLinesArgs.StartLine != nil {
		t.Error("expected the whole log without an offset")
	}

	if _, err := provider.GetPipelineLogLines(context.Background(), run, 12, 40); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastLogLinesArgs.StartLine == nil || *mock.lastLogLinesArgs.StartLine != 41 || *mock.lastLogLinesArgs.LogId != 12 {
		t.Errorf("expected to read log 12 from line 41, got %+v", mock.lastLogLinesArgs)
	}
}

func TestRetryPipelineRun_FailedJobsRetriesTheBuild(t *testing.T) {
	retried := newTestBuild(5, "5", build.BuildStatusValues.InProgress, nil)
	mock := &mockBuildClient{build: &retried, timeline: &build.Timeline{}}
	provider := newPipelineTestProvider(mock, nil)

	run, err := provider.RetryPipelineRun(context.Background(), domain.PipelineRetry{
		Run:   domain.PipelineRunIdentifier{Project: testProject, RunID: 5},
		Scope: domain.PipelineRetryFailedJobs,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastUpdateArgs == nil || mock.lastUpdateArgs.Retry == nil || !*mock.lastUpdateArgs.Retry || *mock.lastUpdateArgs.BuildId != 5 {
		t.Errorf("expected a retry update of build 5, got %+v", mock.lastUpdateArgs)
	}
	if run.ID != 5 || run.Status != domain.PipelineStatusRunning {
		t.Errorf("expected the refreshed run, got %+v", run)
	}
}

func TestRetryPipelineRun_StageRetriesByRefName(t *testing.T) {
	refreshed := newTestBuild(5, "5", build.BuildStatusValues.InProgress, nil)
	mock := &mockBuildClient{build: &refreshed, timeline: &build.Timeline{}}
	provider := newPipelineTestProvider(mock, nil)

	_, err := provider.RetryPipelineRun(context.Background(), domain.PipelineRetry{
		Run:   domain.PipelineRunIdentifier{Project: testProject, RunID: 5},
		Scope: domain.PipelineRetryStage,
		Stage: domain.PipelineStage{Name: "Build", RefName: "build_stage"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	args := mock.lastStageArgs
	if args == nil || *args.StageRefName != "build_stage" || *args.UpdateParameters.State != build.StageUpdateTypeValues.Retry {
		t.Errorf("expected a retry of stage build_stage, got %+v", args)
	}
	if *args.UpdateParameters.ForceRetryAllJobs {
		t.Error("expected only the failed jobs to be retried")
	}
}

func TestRetryPipelineRun_RerunQueuesTheSameCommitAndParameters(t *testing.T) {
	failed := build.BuildResultValues.Failed
	original := newTestBuild(5, "5", build.BuildStatusValues.Completed, &failed)
	original.SourceVersion = strPtr("abc123")
	original.TemplateParameters = &map[string]string{"environment": "prod"}
	queued := newTestBuild(6, "6", build.BuildStatusValues.NotStarted, nil)
	queued.TriggerInfo = &map[string]string{triggerInfoCIMessage: "Fix it"}
	mock := &mockBuildClient{build: &original, queued: &queued, timeline: &build.Timeline{}}
	provider := newPipelineTestProvider(mock, nil)

	run, err := provider.RetryPipelineRun(context.Background(), domain.PipelineRetry{
		Run:   domain.PipelineRunIdentifier{Project: testProject, RunID: 5},
		Scope: domain.PipelineRetryRerun,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	queuedBuild := mock.lastQueueArgs.Build
	if *queuedBuild.Definition.Id != 7 || *queuedBuild.SourceBranch != "refs/heads/main" || *queuedBuild.SourceVersion != "abc123" {
		t.Errorf("expected the original pipeline, branch and commit, got %+v", queuedBuild)
	}
	if (*queuedBuild.TemplateParameters)["environment"] != "prod" {
		t.Errorf("expected the original parameters, got %v", queuedBuild.TemplateParameters)
	}
	if run.ID != 6 {
		t.Errorf("expected the new run, got %d", run.ID)
	}
}

func TestRetryPipelineRun_Errors(t *testing.T) {
	identifier := domain.PipelineRunIdentifier{Project: testProject, RunID: 5}
	provider := newPipelineTestProvider(&mockBuildClient{err: errors.New("boom")}, nil)
	for _, scope := range []domain.PipelineRetryScope{domain.PipelineRetryFailedJobs, domain.PipelineRetryStage, domain.PipelineRetryRerun, "unknown"} {
		if _, err := provider.RetryPipelineRun(context.Background(), domain.PipelineRetry{Run: identifier, Scope: scope}); err == nil {
			t.Errorf("expected an error for %s", scope)
		}
	}
}

func newTriggerTestProvider(buildClient *mockBuildClient, gitClient *mockGitClient) *Provider {
	provider := newPipelineTestProvider(buildClient, nil)
	provider.client.gitClient = gitClient
	return provider
}

func TestGetPipelineTriggerOptions_ReadsBranchesAndYamlParameters(t *testing.T) {
	repoID := "repo-id"
	definition := &build.BuildDefinition{
		Repository: &build.BuildRepository{Id: &repoID, DefaultBranch: strPtr("refs/heads/main")},
		Process:    map[string]interface{}{"type": 2, definitionYamlFilenameField: "pipelines/deploy.yml"},
	}
	gitClient := &mockGitClient{
		refPages:     []*git.GetRefsResponseValue{{Value: []git.GitRef{{Name: strPtr("refs/heads/main")}, {Name: strPtr("refs/heads/feature/a")}}}},
		itemContents: map[string]string{"/pipelines/deploy.yml@main": testPipelineYAML},
	}
	provider := newTriggerTestProvider(&mockBuildClient{definition: definition}, gitClient)

	options, err := provider.GetPipelineTriggerOptions(context.Background(), domain.PipelineIdentifier{Project: testProject, PipelineID: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.Branches.Default != "main" || !slices.Equal(options.Branches.Names, []string{"feature/a", "main"}) {
		t.Errorf("unexpected branches: %+v", options.Branches)
	}
	want := []domain.PipelineParameter{
		{Name: "environment", DisplayName: "Environment", Type: domain.PipelineParameterString, Default: "dev", Values: []string{"dev", "prod"}},
		{Name: "dryRun", Type: domain.PipelineParameterBoolean, Default: "true"},
		{Name: "retries", Type: domain.PipelineParameterNumber, Default: "3"},
		{Name: "note", Type: domain.PipelineParameterString},
	}
	if len(options.Parameters) != len(want) {
		t.Fatalf("expected %d parameters, got %+v", len(want), options.Parameters)
	}
	for i := range want {
		got := options.Parameters[i]
		if got.Name != want[i].Name || got.DisplayName != want[i].DisplayName || got.Type != want[i].Type ||
			got.Default != want[i].Default || !slices.Equal(got.Values, want[i].Values) {
			t.Errorf("parameter %d: expected %+v, got %+v", i, want[i], got)
		}
	}
}

func TestGetPipelineTriggerOptions_UnreadableYamlLeavesNoParameters(t *testing.T) {
	repoID := "repo-id"
	definition := &build.BuildDefinition{
		Repository: &build.BuildRepository{Id: &repoID, DefaultBranch: strPtr("refs/heads/main")},
		Process:    map[string]interface{}{definitionYamlFilenameField: "missing.yml"},
	}
	provider := newTriggerTestProvider(&mockBuildClient{definition: definition}, &mockGitClient{})

	options, err := provider.GetPipelineTriggerOptions(context.Background(), domain.PipelineIdentifier{Project: testProject, PipelineID: 7})
	if err != nil {
		t.Fatalf("expected the pipeline to stay triggerable, got %v", err)
	}
	if len(options.Parameters) != 0 || options.Branches.Default != "main" {
		t.Errorf("unexpected options: %+v", options)
	}
}

func TestGetPipelineTriggerOptions_Errors(t *testing.T) {
	identifier := domain.PipelineIdentifier{Project: testProject, PipelineID: 7}
	if _, err := newTriggerTestProvider(&mockBuildClient{err: errors.New("boom")}, &mockGitClient{}).GetPipelineTriggerOptions(context.Background(), identifier); err == nil {
		t.Error("expected the definition error")
	}
	noRepo := &mockBuildClient{definition: &build.BuildDefinition{}}
	if _, err := newTriggerTestProvider(noRepo, &mockGitClient{}).GetPipelineTriggerOptions(context.Background(), identifier); err == nil {
		t.Error("expected an error for a pipeline without repository")
	}
}

func TestTriggerPipeline_QueuesBranchWithParameters(t *testing.T) {
	queued := newTestBuild(9, "9", build.BuildStatusValues.NotStarted, nil)
	queued.TriggerInfo = &map[string]string{triggerInfoCIMessage: "Add login"}
	mock := &mockBuildClient{queued: &queued, timeline: &build.Timeline{}}
	provider := newPipelineTestProvider(mock, nil)

	run, err := provider.TriggerPipeline(context.Background(), domain.PipelineTrigger{
		Pipeline:   domain.PipelineIdentifier{Project: testProject, PipelineID: 7},
		Branch:     "feature/a",
		Parameters: map[string]string{"environment": "prod"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	queuedBuild := mock.lastQueueArgs.Build
	if *queuedBuild.Definition.Id != 7 || *queuedBuild.SourceBranch != "refs/heads/feature/a" {
		t.Errorf("expected pipeline 7 on refs/heads/feature/a, got %+v", queuedBuild)
	}
	if (*queuedBuild.TemplateParameters)["environment"] != "prod" {
		t.Errorf("expected the parameters, got %v", queuedBuild.TemplateParameters)
	}
	if run.ID != 9 || run.Status != domain.PipelineStatusQueued {
		t.Errorf("expected the queued run, got %+v", run)
	}
}

func TestTriggerPipeline_KeepsFullRefsAndOmitsEmptyParameters(t *testing.T) {
	queued := newTestBuild(9, "9", build.BuildStatusValues.NotStarted, nil)
	queued.TriggerInfo = &map[string]string{triggerInfoCIMessage: "x"}
	mock := &mockBuildClient{queued: &queued, timeline: &build.Timeline{}}
	provider := newPipelineTestProvider(mock, nil)

	if _, err := provider.TriggerPipeline(context.Background(), domain.PipelineTrigger{
		Pipeline: domain.PipelineIdentifier{Project: testProject, PipelineID: 7},
		Branch:   "refs/tags/v1",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *mock.lastQueueArgs.Build.SourceBranch != "refs/tags/v1" || mock.lastQueueArgs.Build.TemplateParameters != nil {
		t.Errorf("unexpected queued build: %+v", mock.lastQueueArgs.Build)
	}
}

func TestTriggerPipeline_APIError(t *testing.T) {
	provider := newPipelineTestProvider(&mockBuildClient{err: errors.New("boom")}, nil)
	if _, err := provider.TriggerPipeline(context.Background(), domain.PipelineTrigger{Pipeline: domain.PipelineIdentifier{Project: testProject, PipelineID: 7}, Branch: "main"}); err == nil {
		t.Error("expected error")
	}
}

func TestParsePipelineParameters_WithoutParameters(t *testing.T) {
	parameters, err := parsePipelineParameters("steps:\n  - script: echo hi\n")
	if err != nil || len(parameters) != 0 {
		t.Errorf("expected no parameters, got %v (%v)", parameters, err)
	}
	if _, err := parsePipelineParameters("parameters: [unclosed"); err == nil {
		t.Error("expected an error for invalid YAML")
	}
}
