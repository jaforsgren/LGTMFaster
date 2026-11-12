package azuredevops

import (
	"context"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

// WorkItemClientInterface defines the subset of workitemtracking.Client methods we use
// This allows us to mock the Azure DevOps Work Item Tracking client for testing
type WorkItemClientInterface interface {
	QueryByWiql(ctx context.Context, args workitemtracking.QueryByWiqlArgs) (*workitemtracking.WorkItemQueryResult, error)
	GetWorkItems(ctx context.Context, args workitemtracking.GetWorkItemsArgs) (*[]workitemtracking.WorkItem, error)
	UpdateWorkItem(ctx context.Context, args workitemtracking.UpdateWorkItemArgs) (*workitemtracking.WorkItem, error)
}
