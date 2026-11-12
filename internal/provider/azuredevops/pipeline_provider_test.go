package azuredevops

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/pipelinesapproval"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
)

const (
	testOrganization = "myorg"
	testProject      = "My Project"
)

type mockBuildClient struct {
	definitionPages   []build.GetDefinitionsResponseValue
	definitionCalls   []build.GetDefinitionsArgs
	builds            []build.Build
	lastGetBuildsArgs build.GetBuildsArgs
	build             *build.Build
	timeline          *build.Timeline
	logLines          []string
	lastLogLinesArgs  build.GetBuildLogLinesArgs
	definition        *build.BuildDefinition
	queued            *build.Build
	lastQueueArgs     *build.QueueBuildArgs
	lastUpdateArgs    *build.UpdateBuildArgs
	lastStageArgs     *build.UpdateStageArgs
	err               error
}

func (m *mockBuildClient) GetBuildLogLines(ctx context.Context, args build.GetBuildLogLinesArgs) (*[]string, error) {
	m.lastLogLinesArgs = args
	if m.err != nil {
		return nil, m.err
	}
	lines := append([]string(nil), m.logLines...)
	return &lines, nil
}

func (m *mockBuildClient) GetDefinition(ctx context.Context, args build.GetDefinitionArgs) (*build.BuildDefinition, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.definition, nil
}

func (m *mockBuildClient) QueueBuild(ctx context.Context, args build.QueueBuildArgs) (*build.Build, error) {
	m.lastQueueArgs = &args
	if m.err != nil {
		return nil, m.err
	}
	return m.queued, nil
}

func (m *mockBuildClient) UpdateBuild(ctx context.Context, args build.UpdateBuildArgs) (*build.Build, error) {
	m.lastUpdateArgs = &args
	if m.err != nil {
		return nil, m.err
	}
	return m.build, nil
}

func (m *mockBuildClient) UpdateStage(ctx context.Context, args build.UpdateStageArgs) error {
	m.lastStageArgs = &args
	return m.err
}

func (m *mockBuildClient) GetDefinitions(ctx context.Context, args build.GetDefinitionsArgs) (*build.GetDefinitionsResponseValue, error) {
	m.definitionCalls = append(m.definitionCalls, args)
	if m.err != nil {
		return nil, m.err
	}
	page := m.definitionPages[len(m.definitionCalls)-1]
	return &page, nil
}

func (m *mockBuildClient) GetBuilds(ctx context.Context, args build.GetBuildsArgs) (*build.GetBuildsResponseValue, error) {
	m.lastGetBuildsArgs = args
	if m.err != nil {
		return nil, m.err
	}
	return &build.GetBuildsResponseValue{Value: m.builds}, nil
}

func (m *mockBuildClient) GetBuild(ctx context.Context, args build.GetBuildArgs) (*build.Build, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.build, nil
}

func (m *mockBuildClient) GetBuildTimeline(ctx context.Context, args build.GetBuildTimelineArgs) (*build.Timeline, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.timeline, nil
}

type mockApprovalClient struct {
	lastArgs *pipelinesapproval.UpdateApprovalsArgs
	err      error
}

func (m *mockApprovalClient) UpdateApprovals(ctx context.Context, args pipelinesapproval.UpdateApprovalsArgs) (*[]pipelinesapproval.Approval, error) {
	m.lastArgs = &args
	if m.err != nil {
		return nil, m.err
	}
	return &[]pipelinesapproval.Approval{}, nil
}

func newPipelineTestProvider(buildClient *mockBuildClient, approvalClient *mockApprovalClient) *Provider {
	return &Provider{
		client: &Client{
			buildClient:    buildClient,
			approvalClient: approvalClient,
			organization:   testOrganization,
		},
	}
}

func newTestBuild(id int, number string, status build.BuildStatus, result *build.BuildResult) build.Build {
	definitionID := 7
	return build.Build{
		Id:           &id,
		BuildNumber:  &number,
		Status:       &status,
		Result:       result,
		SourceBranch: strPtr("refs/heads/main"),
		QueueTime:    &azuredevops.Time{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)},
		Definition:   &build.DefinitionReference{Id: &definitionID, Name: strPtr("CI")},
	}
}

func newTimelineRecord(id uuid.UUID, parentID *uuid.UUID, recordType, name string, order int, state build.TimelineRecordState, result *build.TaskResult) build.TimelineRecord {
	return build.TimelineRecord{
		Id:       &id,
		ParentId: parentID,
		Type:     &recordType,
		Name:     &name,
		Order:    &order,
		State:    &state,
		Result:   result,
	}
}

func TestListPipelineDefinitions_FollowsContinuationToken(t *testing.T) {
	first, second := 1, 2
	mock := &mockBuildClient{
		definitionPages: []build.GetDefinitionsResponseValue{
			{Value: []build.BuildDefinitionReference{{Id: &first}}, ContinuationToken: "next"},
			{Value: []build.BuildDefinitionReference{{Id: &second}}},
		},
	}
	client := &Client{buildClient: mock}

	definitions, err := client.ListPipelineDefinitions(context.Background(), "proj", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(definitions) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(definitions))
	}
	if len(mock.definitionCalls) != 2 {
		t.Fatalf("expected 2 API calls, got %d", len(mock.definitionCalls))
	}
	if mock.definitionCalls[1].ContinuationToken == nil || *mock.definitionCalls[1].ContinuationToken != "next" {
		t.Error("expected second call to pass continuation token")
	}
	firstCall := mock.definitionCalls[0]
	if *firstCall.RepositoryId != "repo" || *firstCall.RepositoryType != azureReposGitRepositoryType {
		t.Error("expected definitions to be filtered by Azure Repos repository")
	}
	if firstCall.IncludeLatestBuilds == nil || !*firstCall.IncludeLatestBuilds {
		t.Error("expected latest builds to be included")
	}
}

func TestListPipelineDefinitions_Error(t *testing.T) {
	client := &Client{buildClient: &mockBuildClient{err: errors.New("boom")}}

	if _, err := client.ListPipelineDefinitions(context.Background(), "proj", "repo"); err == nil {
		t.Fatal("expected error")
	}
}

func TestConvertPipeline_MapsFieldsAndLatestRun(t *testing.T) {
	provider := newPipelineTestProvider(&mockBuildClient{}, nil)
	id := 42
	result := build.BuildResultValues.Failed
	latest := newTestBuild(100, "20260102.1", build.BuildStatusValues.Completed, &result)

	pipeline := provider.convertPipeline(build.BuildDefinitionReference{
		Id:          &id,
		Name:        strPtr("deploy"),
		Path:        strPtr(`\infra`),
		LatestBuild: &latest,
	}, testProject, "api")

	if pipeline.ID != 42 || pipeline.Name != "deploy" || pipeline.Folder != `\infra` {
		t.Errorf("unexpected pipeline fields: %+v", pipeline)
	}
	if pipeline.Project != testProject || pipeline.Repository != "api" {
		t.Errorf("unexpected project/repo: %s/%s", pipeline.Project, pipeline.Repository)
	}
	if pipeline.URL != "https://dev.azure.com/myorg/My%20Project/_build?definitionId=42" {
		t.Errorf("unexpected URL: %s", pipeline.URL)
	}
	if pipeline.LatestRun == nil || pipeline.LatestRun.Status != domain.PipelineStatusFailed {
		t.Errorf("expected failed latest run, got %+v", pipeline.LatestRun)
	}
}

func TestConvertPipeline_NoLatestRun(t *testing.T) {
	provider := newPipelineTestProvider(&mockBuildClient{}, nil)
	id := 1

	pipeline := provider.convertPipeline(build.BuildDefinitionReference{Id: &id}, testProject, "api")

	if pipeline.LatestRun != nil {
		t.Error("expected no latest run")
	}
}

func TestListPipelineRuns_ConvertsBuilds(t *testing.T) {
	succeeded := build.BuildResultValues.Succeeded
	mock := &mockBuildClient{
		builds: []build.Build{
			newTestBuild(2, "run-2", build.BuildStatusValues.InProgress, nil),
			newTestBuild(1, "run-1", build.BuildStatusValues.Completed, &succeeded),
		},
	}
	provider := newPipelineTestProvider(mock, nil)

	runs, err := provider.ListPipelineRuns(context.Background(), domain.PipelineIdentifier{Project: testProject, PipelineID: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
	if runs[0].Status != domain.PipelineStatusRunning || runs[1].Status != domain.PipelineStatusSucceeded {
		t.Errorf("unexpected statuses: %s, %s", runs[0].Status, runs[1].Status)
	}
	if runs[0].SourceBranch != "main" {
		t.Errorf("expected branch prefix stripped, got %s", runs[0].SourceBranch)
	}
	if runs[0].PipelineID != 7 || runs[0].PipelineName != "CI" || runs[0].Project != testProject {
		t.Errorf("unexpected pipeline reference: %+v", runs[0])
	}
	if runs[0].URL != "https://dev.azure.com/myorg/My%20Project/_build/results?buildId=2" {
		t.Errorf("unexpected run URL: %s", runs[0].URL)
	}
	if (*mock.lastGetBuildsArgs.Definitions)[0] != 7 {
		t.Error("expected builds to be filtered by pipeline ID")
	}
	if *mock.lastGetBuildsArgs.Top != pipelineRunsPageSize {
		t.Error("expected run list to be limited to page size")
	}
}

func TestListPipelineRuns_Error(t *testing.T) {
	provider := newPipelineTestProvider(&mockBuildClient{err: errors.New("boom")}, nil)

	if _, err := provider.ListPipelineRuns(context.Background(), domain.PipelineIdentifier{Project: testProject, PipelineID: 7}); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetPipelineRun_PrefersRequestedFor(t *testing.T) {
	b := newTestBuild(5, "run-5", build.BuildStatusValues.NotStarted, nil)
	b.RequestedBy = identityRef("Service Account")
	b.RequestedFor = identityRef("Jane Doe")
	provider := newPipelineTestProvider(&mockBuildClient{build: &b}, nil)

	run, err := provider.GetPipelineRun(context.Background(), domain.PipelineRunIdentifier{Project: testProject, RunID: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if run.Status != domain.PipelineStatusQueued {
		t.Errorf("expected queued, got %s", run.Status)
	}
	if run.RequestedBy.Username != "Jane Doe" {
		t.Errorf("expected requested-for identity, got %s", run.RequestedBy.Username)
	}
}

func TestGetPipelineRun_NotFound(t *testing.T) {
	provider := newPipelineTestProvider(&mockBuildClient{}, nil)

	if _, err := provider.GetPipelineRun(context.Background(), domain.PipelineRunIdentifier{Project: testProject, RunID: 5}); err == nil {
		t.Fatal("expected not found error")
	}
}

func TestGetPipelineRunStages_AttachesPendingApprovalToStage(t *testing.T) {
	buildStage := uuid.New()
	deployStage := uuid.New()
	checkpoint := uuid.New()
	approval := uuid.New()
	job := uuid.New()
	succeeded := build.TaskResultValues.Succeeded

	records := []build.TimelineRecord{
		newTimelineRecord(deployStage, nil, timelineRecordTypeStage, "Deploy", 2, build.TimelineRecordStateValues.Pending, nil),
		newTimelineRecord(buildStage, nil, timelineRecordTypeStage, "Build", 1, build.TimelineRecordStateValues.Completed, &succeeded),
		newTimelineRecord(job, &buildStage, "Job", "Compile", 1, build.TimelineRecordStateValues.Completed, &succeeded),
		newTimelineRecord(checkpoint, &deployStage, "Checkpoint", "Checkpoint", 1, build.TimelineRecordStateValues.InProgress, nil),
		newTimelineRecord(approval, &checkpoint, timelineRecordTypeApproval, "Approval", 1, build.TimelineRecordStateValues.InProgress, nil),
	}
	provider := newPipelineTestProvider(&mockBuildClient{timeline: &build.Timeline{Records: &records}}, nil)

	stages, err := provider.GetPipelineRunStages(context.Background(), domain.PipelineRunIdentifier{Project: testProject, RunID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stages) != 2 {
		t.Fatalf("expected 2 stages (jobs excluded), got %d", len(stages))
	}
	if stages[0].Name != "Build" || stages[1].Name != "Deploy" {
		t.Errorf("expected stages sorted by order, got %s, %s", stages[0].Name, stages[1].Name)
	}
	if stages[0].Status != domain.PipelineStatusSucceeded || stages[0].AwaitsApproval() {
		t.Errorf("unexpected build stage: %+v", stages[0])
	}
	if stages[1].Status != domain.PipelineStatusWaitingForApproval {
		t.Errorf("expected deploy stage waiting for approval, got %s", stages[1].Status)
	}
	if stages[1].PendingApprovalID != approval.String() {
		t.Errorf("expected approval ID %s, got %s", approval, stages[1].PendingApprovalID)
	}
}

func TestGetPipelineRunStages_CarriesStartAndFinishTimes(t *testing.T) {
	stage := uuid.New()
	succeeded := build.TaskResultValues.Succeeded
	started := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	finished := started.Add(90 * time.Second)

	record := newTimelineRecord(stage, nil, timelineRecordTypeStage, "Build", 1, build.TimelineRecordStateValues.Completed, &succeeded)
	record.StartTime = &azuredevops.Time{Time: started}
	record.FinishTime = &azuredevops.Time{Time: finished}
	record.Identifier = strPtr("build_stage")
	records := []build.TimelineRecord{record}
	provider := newPipelineTestProvider(&mockBuildClient{timeline: &build.Timeline{Records: &records}}, nil)

	stages, err := provider.GetPipelineRunStages(context.Background(), domain.PipelineRunIdentifier{Project: testProject, RunID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stages[0].RefName != "build_stage" {
		t.Errorf("expected the stage identifier as ref name, got %q", stages[0].RefName)
	}
	if !stages[0].StartedAt.Equal(started) || !stages[0].FinishedAt.Equal(finished) {
		t.Errorf("expected stage times %s - %s, got %s - %s", started, finished, stages[0].StartedAt, stages[0].FinishedAt)
	}
}

func TestGetPipelineRunStages_IgnoresCompletedApprovals(t *testing.T) {
	stage := uuid.New()
	checkpoint := uuid.New()
	approval := uuid.New()
	succeeded := build.TaskResultValues.Succeeded

	records := []build.TimelineRecord{
		newTimelineRecord(stage, nil, timelineRecordTypeStage, "Deploy", 1, build.TimelineRecordStateValues.InProgress, nil),
		newTimelineRecord(checkpoint, &stage, "Checkpoint", "Checkpoint", 1, build.TimelineRecordStateValues.Completed, &succeeded),
		newTimelineRecord(approval, &checkpoint, timelineRecordTypeApproval, "Approval", 1, build.TimelineRecordStateValues.Completed, &succeeded),
	}
	provider := newPipelineTestProvider(&mockBuildClient{timeline: &build.Timeline{Records: &records}}, nil)

	stages, err := provider.GetPipelineRunStages(context.Background(), domain.PipelineRunIdentifier{Project: testProject, RunID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stages[0].AwaitsApproval() || stages[0].Status != domain.PipelineStatusRunning {
		t.Errorf("expected running stage without pending approval, got %+v", stages[0])
	}
}

func TestGetPipelineRunStages_EmptyTimeline(t *testing.T) {
	provider := newPipelineTestProvider(&mockBuildClient{}, nil)

	stages, err := provider.GetPipelineRunStages(context.Background(), domain.PipelineRunIdentifier{Project: testProject, RunID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stages) != 0 {
		t.Errorf("expected no stages, got %d", len(stages))
	}
}

func TestFindAncestor_StopsOnCycle(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	recordA := newTimelineRecord(a, &b, "Checkpoint", "A", 1, build.TimelineRecordStateValues.InProgress, nil)
	recordB := newTimelineRecord(b, &a, "Checkpoint", "B", 1, build.TimelineRecordStateValues.InProgress, nil)

	_, found := findAncestor(recordA, map[uuid.UUID]build.TimelineRecord{a: recordA, b: recordB}, timelineRecordTypeStage)

	if found {
		t.Error("expected no stage ancestor for cyclic records")
	}
}

func TestSubmitStageApproval_SendsDecision(t *testing.T) {
	tests := []struct {
		decision domain.ApprovalDecision
		expected pipelinesapproval.ApprovalStatus
	}{
		{domain.ApprovalDecisionApprove, pipelinesapproval.ApprovalStatusValues.Approved},
		{domain.ApprovalDecisionReject, pipelinesapproval.ApprovalStatusValues.Rejected},
	}

	for _, tt := range tests {
		t.Run(string(tt.decision), func(t *testing.T) {
			approvalClient := &mockApprovalClient{}
			provider := newPipelineTestProvider(nil, approvalClient)
			approvalID := uuid.New()

			err := provider.SubmitStageApproval(context.Background(), domain.StageApproval{
				Project:    testProject,
				ApprovalID: approvalID.String(),
				Decision:   tt.decision,
				Comment:    "looks good",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			args := approvalClient.lastArgs
			if args == nil || *args.Project != testProject {
				t.Fatal("expected approval update for project")
			}
			update := (*args.UpdateParameters)[0]
			if *update.ApprovalId != approvalID || *update.Status != tt.expected || *update.Comment != "looks good" {
				t.Errorf("unexpected update parameters: %+v", update)
			}
		})
	}
}

func TestSubmitStageApproval_InvalidApprovalID(t *testing.T) {
	approvalClient := &mockApprovalClient{}
	provider := newPipelineTestProvider(nil, approvalClient)

	err := provider.SubmitStageApproval(context.Background(), domain.StageApproval{
		Project:    testProject,
		ApprovalID: "not-a-uuid",
		Decision:   domain.ApprovalDecisionApprove,
	})

	if err == nil {
		t.Fatal("expected error for invalid approval ID")
	}
	if approvalClient.lastArgs != nil {
		t.Error("expected no API call for invalid approval ID")
	}
}

func TestSubmitStageApproval_UnsupportedDecision(t *testing.T) {
	provider := newPipelineTestProvider(nil, &mockApprovalClient{})

	err := provider.SubmitStageApproval(context.Background(), domain.StageApproval{
		Project:    testProject,
		ApprovalID: uuid.New().String(),
		Decision:   domain.ApprovalDecision("maybe"),
	})

	if err == nil {
		t.Fatal("expected error for unsupported decision")
	}
}

func TestSubmitStageApproval_APIError(t *testing.T) {
	provider := newPipelineTestProvider(nil, &mockApprovalClient{err: errors.New("forbidden")})

	err := provider.SubmitStageApproval(context.Background(), domain.StageApproval{
		Project:    testProject,
		ApprovalID: uuid.New().String(),
		Decision:   domain.ApprovalDecisionApprove,
	})

	if err == nil {
		t.Fatal("expected API error to propagate")
	}
}

func TestMapBuildStatus(t *testing.T) {
	succeeded := build.BuildResultValues.Succeeded
	partial := build.BuildResultValues.PartiallySucceeded
	failed := build.BuildResultValues.Failed
	canceled := build.BuildResultValues.Canceled

	tests := []struct {
		name     string
		status   *build.BuildStatus
		result   *build.BuildResult
		expected domain.PipelineStatus
	}{
		{"nil status", nil, nil, domain.PipelineStatusUnknown},
		{"not started", statusPtr(build.BuildStatusValues.NotStarted), nil, domain.PipelineStatusQueued},
		{"postponed", statusPtr(build.BuildStatusValues.Postponed), nil, domain.PipelineStatusQueued},
		{"in progress", statusPtr(build.BuildStatusValues.InProgress), nil, domain.PipelineStatusRunning},
		{"cancelling", statusPtr(build.BuildStatusValues.Cancelling), nil, domain.PipelineStatusRunning},
		{"succeeded", statusPtr(build.BuildStatusValues.Completed), &succeeded, domain.PipelineStatusSucceeded},
		{"partially succeeded", statusPtr(build.BuildStatusValues.Completed), &partial, domain.PipelineStatusPartiallySucceeded},
		{"failed", statusPtr(build.BuildStatusValues.Completed), &failed, domain.PipelineStatusFailed},
		{"canceled", statusPtr(build.BuildStatusValues.Completed), &canceled, domain.PipelineStatusCanceled},
		{"completed without result", statusPtr(build.BuildStatusValues.Completed), nil, domain.PipelineStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapBuildStatus(tt.status, tt.result); got != tt.expected {
				t.Errorf("mapBuildStatus() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestMapTimelineRecordStatus(t *testing.T) {
	tests := []struct {
		name     string
		state    *build.TimelineRecordState
		result   build.TaskResult
		expected domain.PipelineStatus
	}{
		{"nil state", nil, "", domain.PipelineStatusUnknown},
		{"pending", recordStatePtr(build.TimelineRecordStateValues.Pending), "", domain.PipelineStatusQueued},
		{"in progress", recordStatePtr(build.TimelineRecordStateValues.InProgress), "", domain.PipelineStatusRunning},
		{"succeeded", recordStatePtr(build.TimelineRecordStateValues.Completed), build.TaskResultValues.Succeeded, domain.PipelineStatusSucceeded},
		{"with issues", recordStatePtr(build.TimelineRecordStateValues.Completed), build.TaskResultValues.SucceededWithIssues, domain.PipelineStatusPartiallySucceeded},
		{"failed", recordStatePtr(build.TimelineRecordStateValues.Completed), build.TaskResultValues.Failed, domain.PipelineStatusFailed},
		{"canceled", recordStatePtr(build.TimelineRecordStateValues.Completed), build.TaskResultValues.Canceled, domain.PipelineStatusCanceled},
		{"abandoned", recordStatePtr(build.TimelineRecordStateValues.Completed), build.TaskResultValues.Abandoned, domain.PipelineStatusCanceled},
		{"skipped", recordStatePtr(build.TimelineRecordStateValues.Completed), build.TaskResultValues.Skipped, domain.PipelineStatusSkipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result
			if got := mapTimelineRecordStatus(tt.state, &result); got != tt.expected {
				t.Errorf("mapTimelineRecordStatus() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func identityRef(displayName string) *webapi.IdentityRef {
	return &webapi.IdentityRef{DisplayName: &displayName}
}

func statusPtr(s build.BuildStatus) *build.BuildStatus { return &s }

func recordStatePtr(s build.TimelineRecordState) *build.TimelineRecordState { return &s }

// mockCoreClient embeds the interface so only GetProjects needs implementing; any other call panics.
type mockCoreClient struct {
	core.Client
	projects []core.TeamProjectReference
	err      error
}

func (m *mockCoreClient) GetProjects(ctx context.Context, args core.GetProjectsArgs) (*core.GetProjectsResponseValue, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &core.GetProjectsResponseValue{Value: m.projects}, nil
}

func newActiveRunsTestProvider(buildClient *mockBuildClient, coreClient *mockCoreClient) *Provider {
	provider := newPipelineTestProvider(buildClient, nil)
	provider.client.coreClient = coreClient
	provider.client.userID = "user-guid"
	return provider
}

func TestListActivePipelineRuns_QueriesUnfinishedRunsRequestedForTheUser(t *testing.T) {
	succeeded := build.BuildResultValues.Succeeded
	mock := &mockBuildClient{builds: []build.Build{
		newTestBuild(1, "20260101.1", build.BuildStatusValues.InProgress, nil),
		newTestBuild(2, "20260101.2", build.BuildStatusValues.NotStarted, nil),
		newTestBuild(3, "20260101.3", build.BuildStatusValues.Completed, &succeeded),
	}}
	provider := newActiveRunsTestProvider(mock, &mockCoreClient{projects: []core.TeamProjectReference{{Name: strPtr(testProject)}}})

	runs, err := provider.ListActivePipelineRuns(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 2 || runs[0].Status != domain.PipelineStatusRunning || runs[1].Status != domain.PipelineStatusQueued {
		t.Fatalf("expected the running and the queued run, got %+v", runs)
	}
	if runs[0].Project != testProject || runs[0].PipelineName != "CI" {
		t.Errorf("expected the run to carry its project and pipeline, got %+v", runs[0])
	}
	args := mock.lastGetBuildsArgs
	if *args.RequestedFor != "user-guid" || *args.Project != testProject {
		t.Errorf("expected builds requested for the user in the project, got %+v", args)
	}
	if *args.StatusFilter != "notStarted,postponed,inProgress,cancelling" {
		t.Errorf("expected only unfinished statuses, got %s", *args.StatusFilter)
	}
}

func TestListActivePipelineRuns_SkipsProjectsThatFail(t *testing.T) {
	provider := newActiveRunsTestProvider(&mockBuildClient{err: errors.New("forbidden")},
		&mockCoreClient{projects: []core.TeamProjectReference{{Name: strPtr(testProject)}}})

	runs, err := provider.ListActivePipelineRuns(context.Background())

	if err != nil || len(runs) != 0 {
		t.Errorf("expected no runs and no error, got %v, %v", runs, err)
	}
}

func TestListActivePipelineRuns_ReturnsProjectListingErrors(t *testing.T) {
	provider := newActiveRunsTestProvider(&mockBuildClient{}, &mockCoreClient{err: errors.New("unauthorized")})

	if _, err := provider.ListActivePipelineRuns(context.Background()); err == nil {
		t.Error("expected the project listing error")
	}
}
