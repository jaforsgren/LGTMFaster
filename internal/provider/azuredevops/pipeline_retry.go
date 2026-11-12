package azuredevops

import (
	"context"
	"fmt"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

func (p *Provider) RetryPipelineRun(ctx context.Context, retry domain.PipelineRetry) (*domain.PipelineRun, error) {
	identifier := retry.Run
	logger.Log("AzureDevOps: Retrying %s of run %d in %s", retry.Scope, identifier.RunID, identifier.Project)

	var err error
	switch retry.Scope {
	case domain.PipelineRetryFailedJobs:
		_, err = p.client.RetryBuild(ctx, identifier.Project, identifier.RunID)
	case domain.PipelineRetryStage:
		err = p.client.RetryStage(ctx, identifier.Project, identifier.RunID, retry.Stage.RefName)
	case domain.PipelineRetryRerun:
		return p.rerunPipelineRun(ctx, identifier)
	default:
		return nil, fmt.Errorf("unsupported retry scope: %s", retry.Scope)
	}
	if err != nil {
		logger.LogError("AZDO_RETRY_PIPELINE_RUN", fmt.Sprintf("%s/%d", identifier.Project, identifier.RunID), err)
		return nil, err
	}
	return p.GetPipelineRun(ctx, identifier)
}

// rerunPipelineRun queues a new run of the commit, branch and parameters the run was started with.
func (p *Provider) rerunPipelineRun(ctx context.Context, identifier domain.PipelineRunIdentifier) (*domain.PipelineRun, error) {
	original, err := p.client.GetBuild(ctx, identifier.Project, identifier.RunID)
	if err != nil {
		logger.LogError("AZDO_RERUN_PIPELINE_RUN", fmt.Sprintf("%s/%d", identifier.Project, identifier.RunID), err)
		return nil, err
	}
	if original == nil || original.Definition == nil {
		return nil, fmt.Errorf("run %d in project '%s' has no pipeline to rerun", identifier.RunID, identifier.Project)
	}

	queued, err := p.client.QueueBuild(ctx, identifier.Project, common.GetInt(original.Definition.Id), build.Build{
		SourceBranch:       original.SourceBranch,
		SourceVersion:      original.SourceVersion,
		Parameters:         original.Parameters,
		TemplateParameters: original.TemplateParameters,
	})
	if err != nil {
		logger.LogError("AZDO_RERUN_PIPELINE_RUN", fmt.Sprintf("%s/%d", identifier.Project, identifier.RunID), err)
		return nil, err
	}
	run := p.convertPipelineRuns(ctx, []build.Build{*queued}, identifier.Project)[0]
	return &run, nil
}
