package azuredevops

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/pipelinesapproval"
)

const (
	timelineRecordTypeStage    = "Stage"
	timelineRecordTypeApproval = "Checkpoint.Approval"
)

var _ domain.PipelineProvider = (*Provider)(nil)

func (p *Provider) ListPipelines(ctx context.Context) ([]domain.Pipeline, error) {
	pipelines := []domain.Pipeline{}
	var mu sync.Mutex

	err := p.visitRepositories(ctx, func(projectID, projectName, repoID, repoName string) {
		definitions, err := p.client.ListPipelineDefinitions(ctx, projectID, repoID)
		if err != nil {
			logger.Log("AzureDevOps: Skipping pipelines for inaccessible repo %s/%s: %v", projectName, repoName, err)
			return
		}

		for _, definition := range definitions {
			pipeline := p.convertPipeline(definition, projectName, repoName)
			mu.Lock()
			pipelines = append(pipelines, pipeline)
			mu.Unlock()
		}
	})

	logger.Log("AzureDevOps: Listed %d pipelines", len(pipelines))
	return pipelines, err
}

func (p *Provider) ListPipelineRuns(ctx context.Context, identifier domain.PipelineIdentifier) ([]domain.PipelineRun, error) {
	builds, err := p.client.ListBuilds(ctx, identifier.Project, identifier.PipelineID)
	if err != nil {
		logger.LogError("AZDO_LIST_PIPELINE_RUNS", fmt.Sprintf("%s/%d", identifier.Project, identifier.PipelineID), err)
		return nil, err
	}

	runs := make([]domain.PipelineRun, 0, len(builds))
	for i := range builds {
		runs = append(runs, p.convertPipelineRun(&builds[i], identifier.Project))
	}
	return runs, nil
}

func (p *Provider) GetPipelineRun(ctx context.Context, identifier domain.PipelineRunIdentifier) (*domain.PipelineRun, error) {
	b, err := p.client.GetBuild(ctx, identifier.Project, identifier.RunID)
	if err != nil {
		logger.LogError("AZDO_GET_PIPELINE_RUN", fmt.Sprintf("%s/%d", identifier.Project, identifier.RunID), err)
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("run %d not found in project '%s'", identifier.RunID, identifier.Project)
	}

	run := p.convertPipelineRun(b, identifier.Project)
	return &run, nil
}

func (p *Provider) GetPipelineRunStages(ctx context.Context, identifier domain.PipelineRunIdentifier) ([]domain.PipelineStage, error) {
	timeline, err := p.client.GetBuildTimeline(ctx, identifier.Project, identifier.RunID)
	if err != nil {
		logger.LogError("AZDO_GET_PIPELINE_STAGES", fmt.Sprintf("%s/%d", identifier.Project, identifier.RunID), err)
		return nil, err
	}
	if timeline == nil || timeline.Records == nil {
		return []domain.PipelineStage{}, nil
	}
	return convertTimelineToStages(*timeline.Records), nil
}

func (p *Provider) SubmitStageApproval(ctx context.Context, approval domain.StageApproval) error {
	status, err := convertApprovalDecision(approval.Decision)
	if err != nil {
		return err
	}

	logger.Log("AzureDevOps: Submitting %s for approval %s in %s", approval.Decision, approval.ApprovalID, approval.Project)
	if err := p.client.UpdateApproval(ctx, approval.Project, approval.ApprovalID, status, approval.Comment); err != nil {
		logger.LogError("AZDO_SUBMIT_STAGE_APPROVAL", approval.ApprovalID, err)
		return err
	}
	return nil
}

func (p *Provider) convertPipeline(definition build.BuildDefinitionReference, projectName, repoName string) domain.Pipeline {
	pipeline := domain.Pipeline{
		ID:         common.GetInt(definition.Id),
		Name:       common.GetString(definition.Name),
		Folder:     common.GetString(definition.Path),
		Project:    projectName,
		Repository: repoName,
	}
	pipeline.URL = p.buildPipelineURL(projectName, pipeline.ID)

	if definition.LatestBuild != nil {
		latestRun := p.convertPipelineRun(definition.LatestBuild, projectName)
		pipeline.LatestRun = &latestRun
	}
	return pipeline
}

func (p *Provider) convertPipelineRun(b *build.Build, projectName string) domain.PipelineRun {
	run := domain.PipelineRun{
		ID:           common.GetInt(b.Id),
		Number:       common.GetString(b.BuildNumber),
		Project:      projectName,
		Status:       mapBuildStatus(b.Status, b.Result),
		SourceBranch: extractBranchName(b.SourceBranch),
		QueuedAt:     timeOrZero(b.QueueTime),
		StartedAt:    timeOrZero(b.StartTime),
		FinishedAt:   timeOrZero(b.FinishTime),
	}

	if b.Definition != nil {
		run.PipelineID = common.GetInt(b.Definition.Id)
		run.PipelineName = common.GetString(b.Definition.Name)
	}

	if b.RequestedFor != nil {
		run.RequestedBy = convertIdentity(b.RequestedFor)
	} else if b.RequestedBy != nil {
		run.RequestedBy = convertIdentity(b.RequestedBy)
	}

	run.URL = p.buildPipelineRunURL(projectName, run.ID)
	return run
}

func (p *Provider) buildPipelineURL(projectName string, pipelineID int) string {
	return fmt.Sprintf("https://dev.azure.com/%s/%s/_build?definitionId=%d",
		p.client.organization, url.PathEscape(projectName), pipelineID)
}

func (p *Provider) buildPipelineRunURL(projectName string, runID int) string {
	return fmt.Sprintf("https://dev.azure.com/%s/%s/_build/results?buildId=%d",
		p.client.organization, url.PathEscape(projectName), runID)
}

// convertTimelineToStages walks the flat timeline record list. Approvals are nested as
// Stage -> Checkpoint -> Checkpoint.Approval, so each pending approval is attributed to
// its nearest Stage ancestor.
func convertTimelineToStages(records []build.TimelineRecord) []domain.PipelineStage {
	recordsByID := make(map[uuid.UUID]build.TimelineRecord, len(records))
	for _, record := range records {
		if record.Id != nil {
			recordsByID[*record.Id] = record
		}
	}

	stagesByID := make(map[uuid.UUID]*domain.PipelineStage)
	for _, record := range records {
		if record.Id == nil || common.GetString(record.Type) != timelineRecordTypeStage {
			continue
		}
		stagesByID[*record.Id] = &domain.PipelineStage{
			ID:     record.Id.String(),
			Name:   common.GetString(record.Name),
			Order:  common.GetInt(record.Order),
			Status: mapTimelineRecordStatus(record.State, record.Result),
		}
	}

	for _, record := range records {
		if record.Id == nil || common.GetString(record.Type) != timelineRecordTypeApproval || isRecordCompleted(record.State) {
			continue
		}
		stageID, found := findStageAncestor(record, recordsByID)
		if !found {
			continue
		}
		if stage, ok := stagesByID[stageID]; ok {
			stage.PendingApprovalID = record.Id.String()
			stage.Status = domain.PipelineStatusWaitingForApproval
		}
	}

	stages := make([]domain.PipelineStage, 0, len(stagesByID))
	for _, stage := range stagesByID {
		stages = append(stages, *stage)
	}
	sort.SliceStable(stages, func(i, j int) bool {
		return stages[i].Order < stages[j].Order
	})
	return stages
}

func findStageAncestor(record build.TimelineRecord, recordsByID map[uuid.UUID]build.TimelineRecord) (uuid.UUID, bool) {
	visited := make(map[uuid.UUID]bool)
	current := record
	for current.ParentId != nil && !visited[*current.ParentId] {
		visited[*current.ParentId] = true
		parent, ok := recordsByID[*current.ParentId]
		if !ok {
			return uuid.UUID{}, false
		}
		if common.GetString(parent.Type) == timelineRecordTypeStage {
			return *parent.Id, true
		}
		current = parent
	}
	return uuid.UUID{}, false
}

func isRecordCompleted(state *build.TimelineRecordState) bool {
	return state != nil && *state == build.TimelineRecordStateValues.Completed
}

func mapBuildStatus(status *build.BuildStatus, result *build.BuildResult) domain.PipelineStatus {
	if status == nil {
		return domain.PipelineStatusUnknown
	}

	switch *status {
	case build.BuildStatusValues.NotStarted, build.BuildStatusValues.Postponed:
		return domain.PipelineStatusQueued
	case build.BuildStatusValues.InProgress, build.BuildStatusValues.Cancelling:
		return domain.PipelineStatusRunning
	case build.BuildStatusValues.Completed:
		return mapBuildResult(result)
	default:
		return domain.PipelineStatusUnknown
	}
}

func mapBuildResult(result *build.BuildResult) domain.PipelineStatus {
	if result == nil {
		return domain.PipelineStatusUnknown
	}

	switch *result {
	case build.BuildResultValues.Succeeded:
		return domain.PipelineStatusSucceeded
	case build.BuildResultValues.PartiallySucceeded:
		return domain.PipelineStatusPartiallySucceeded
	case build.BuildResultValues.Failed:
		return domain.PipelineStatusFailed
	case build.BuildResultValues.Canceled:
		return domain.PipelineStatusCanceled
	default:
		return domain.PipelineStatusUnknown
	}
}

func mapTimelineRecordStatus(state *build.TimelineRecordState, result *build.TaskResult) domain.PipelineStatus {
	if state == nil {
		return domain.PipelineStatusUnknown
	}

	switch *state {
	case build.TimelineRecordStateValues.Pending:
		return domain.PipelineStatusQueued
	case build.TimelineRecordStateValues.InProgress:
		return domain.PipelineStatusRunning
	case build.TimelineRecordStateValues.Completed:
		return mapTaskResult(result)
	default:
		return domain.PipelineStatusUnknown
	}
}

func mapTaskResult(result *build.TaskResult) domain.PipelineStatus {
	if result == nil {
		return domain.PipelineStatusUnknown
	}

	switch *result {
	case build.TaskResultValues.Succeeded:
		return domain.PipelineStatusSucceeded
	case build.TaskResultValues.SucceededWithIssues:
		return domain.PipelineStatusPartiallySucceeded
	case build.TaskResultValues.Failed:
		return domain.PipelineStatusFailed
	case build.TaskResultValues.Canceled, build.TaskResultValues.Abandoned:
		return domain.PipelineStatusCanceled
	case build.TaskResultValues.Skipped:
		return domain.PipelineStatusSkipped
	default:
		return domain.PipelineStatusUnknown
	}
}

func convertApprovalDecision(decision domain.ApprovalDecision) (pipelinesapproval.ApprovalStatus, error) {
	switch decision {
	case domain.ApprovalDecisionApprove:
		return pipelinesapproval.ApprovalStatusValues.Approved, nil
	case domain.ApprovalDecisionReject:
		return pipelinesapproval.ApprovalStatusValues.Rejected, nil
	default:
		return "", fmt.Errorf("unsupported approval decision: %s", decision)
	}
}

func timeOrZero(t *azuredevops.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.Time
}
