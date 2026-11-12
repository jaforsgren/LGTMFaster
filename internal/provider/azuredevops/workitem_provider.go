package azuredevops

import (
	"context"
	"fmt"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

const (
	workItemFieldType        = "System.WorkItemType"
	workItemFieldTitle       = "System.Title"
	workItemFieldState       = "System.State"
	workItemFieldDescription = "System.Description"
	// workItemFieldReproSteps is where bugs keep their description.
	workItemFieldReproSteps = "Microsoft.VSTS.TCM.ReproSteps"
	workItemURLFormat       = "https://dev.azure.com/%s/_workitems/edit/%d"
)

func (p *Provider) ListLinkedWorkItems(ctx context.Context, identifier domain.PRIdentifier) ([]domain.WorkItem, error) {
	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, identifier.Repository)
	if err != nil {
		return nil, err
	}
	ids, err := p.client.ListPullRequestWorkItemIDs(ctx, projectID, repoID, identifier.Number)
	if err != nil {
		logger.LogError("AZDO_LIST_PR_WORK_ITEMS", fmt.Sprintf("%s#%d", identifier.Repository, identifier.Number), err)
		return nil, err
	}
	return p.getWorkItems(ctx, ids)
}

// ListAssignedWorkItems searches the whole organization, so work items of other projects can be
// linked too.
func (p *Provider) ListAssignedWorkItems(ctx context.Context, _ domain.PRIdentifier) ([]domain.WorkItem, error) {
	ids, err := p.client.QueryAssignedWorkItemIDs(ctx)
	if err != nil {
		logger.LogError("AZDO_LIST_ASSIGNED_WORK_ITEMS", p.client.organization, err)
		return nil, err
	}
	return p.getWorkItems(ctx, ids)
}

func (p *Provider) LinkWorkItem(ctx context.Context, identifier domain.PRIdentifier, workItemID int) error {
	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, identifier.Repository)
	if err != nil {
		return err
	}
	if err := p.client.LinkPullRequest(ctx, workItemID, projectID, repoID, identifier.Number); err != nil {
		logger.LogError("AZDO_LINK_WORK_ITEM", fmt.Sprintf("%s#%d", identifier.Repository, identifier.Number), err)
		return err
	}
	logger.Log("AzureDevOps: Linked work item %d to PR #%d", workItemID, identifier.Number)
	return nil
}

func (p *Provider) getWorkItems(ctx context.Context, ids []int) ([]domain.WorkItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	workItems, err := p.client.GetWorkItems(ctx, ids)
	if err != nil {
		logger.LogError("AZDO_GET_WORK_ITEMS", p.client.organization, err)
		return nil, err
	}
	converted := make([]domain.WorkItem, 0, len(workItems))
	for _, workItem := range workItems {
		converted = append(converted, p.toDomainWorkItem(workItem))
	}
	return converted, nil
}

func (p *Provider) toDomainWorkItem(workItem workitemtracking.WorkItem) domain.WorkItem {
	var fields map[string]interface{}
	if workItem.Fields != nil {
		fields = *workItem.Fields
	}
	description := workItemField(fields, workItemFieldDescription)
	if description == "" {
		description = workItemField(fields, workItemFieldReproSteps)
	}
	id := common.GetInt(workItem.Id)
	return domain.WorkItem{
		ID:          id,
		Type:        workItemField(fields, workItemFieldType),
		Title:       workItemField(fields, workItemFieldTitle),
		State:       workItemField(fields, workItemFieldState),
		Description: common.HTMLToText(description),
		URL:         fmt.Sprintf(workItemURLFormat, p.client.organization, id),
	}
}

func workItemField(fields map[string]interface{}, name string) string {
	value, _ := fields[name].(string)
	return value
}
