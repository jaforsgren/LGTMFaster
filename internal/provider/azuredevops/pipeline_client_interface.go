package azuredevops

import (
	"context"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/pipelinesapproval"
)

// BuildClientInterface defines the subset of build.Client methods we use
// This allows us to mock the Azure DevOps Build client for testing
type BuildClientInterface interface {
	GetDefinitions(ctx context.Context, args build.GetDefinitionsArgs) (*build.GetDefinitionsResponseValue, error)
	GetBuilds(ctx context.Context, args build.GetBuildsArgs) (*build.GetBuildsResponseValue, error)
	GetBuild(ctx context.Context, args build.GetBuildArgs) (*build.Build, error)
	GetBuildTimeline(ctx context.Context, args build.GetBuildTimelineArgs) (*build.Timeline, error)
}

// ApprovalClientInterface defines the subset of pipelinesapproval.Client methods we use
// This allows us to mock the Azure DevOps Pipelines Approval client for testing
type ApprovalClientInterface interface {
	UpdateApprovals(ctx context.Context, args pipelinesapproval.UpdateApprovalsArgs) (*[]pipelinesapproval.Approval, error)
}
