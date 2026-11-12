package azuredevops

import (
	"context"
	"fmt"
	"strconv"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

const (
	// maxWorkItemsPerRequest is the most IDs the work items API accepts in one call.
	maxWorkItemsPerRequest = 200
	// assignedWorkItemsQuery is org-wide: without a project, WIQL searches every project.
	assignedWorkItemsQuery = "SELECT [System.Id] FROM WorkItems" +
		" WHERE [System.AssignedTo] = @Me" +
		" AND [System.WorkItemType] IN ('Bug', 'Task', 'User Story')" +
		" AND [System.State] NOT IN ('Closed', 'Done', 'Removed', 'Completed')" +
		" ORDER BY [System.ChangedDate] DESC"
	// pullRequestArtifactURLFormat is how a work item refers to a pull request: project and repository
	// IDs and the pull request number, separated by escaped slashes.
	pullRequestArtifactURLFormat = "vstfs:///Git/PullRequestId/%s%%2F%s%%2F%d"
	artifactLinkRelation         = "ArtifactLink"
	pullRequestLinkName          = "Pull Request"
	appendRelationPath           = "/relations/-"
)

// ListPullRequestWorkItemIDs returns the IDs of the work items linked to a pull request.
func (c *Client) ListPullRequestWorkItemIDs(ctx context.Context, projectID, repoID string, pullRequestID int) ([]int, error) {
	refs, err := c.gitClient.GetPullRequestWorkItemRefs(ctx, git.GetPullRequestWorkItemRefsArgs{
		Project:       &projectID,
		RepositoryId:  &repoID,
		PullRequestId: &pullRequestID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get work items of PR %d: %w", pullRequestID, err)
	}
	if refs == nil {
		return nil, nil
	}
	ids := make([]int, 0, len(*refs))
	for _, ref := range *refs {
		if ref.Id == nil {
			continue
		}
		id, err := strconv.Atoi(*ref.Id)
		if err != nil {
			return nil, fmt.Errorf("unexpected work item ID %q on PR %d: %w", *ref.Id, pullRequestID, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// QueryAssignedWorkItemIDs returns the IDs of the user's open bugs, tasks and user stories, most
// recently changed first.
func (c *Client) QueryAssignedWorkItemIDs(ctx context.Context) ([]int, error) {
	query := assignedWorkItemsQuery
	result, err := c.workItemClient.QueryByWiql(ctx, workitemtracking.QueryByWiqlArgs{
		Wiql: &workitemtracking.Wiql{Query: &query},
		Top:  intPtr(maxWorkItemsPerRequest),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query assigned work items: %w", err)
	}
	if result == nil || result.WorkItems == nil {
		return nil, nil
	}
	ids := make([]int, 0, len(*result.WorkItems))
	for _, ref := range *result.WorkItems {
		if ref.Id != nil {
			ids = append(ids, *ref.Id)
		}
	}
	return ids, nil
}

// GetWorkItems returns the work items with the given IDs in that order, leaving out deleted ones.
func (c *Client) GetWorkItems(ctx context.Context, ids []int) ([]workitemtracking.WorkItem, error) {
	var workItems []workitemtracking.WorkItem
	errorPolicy := workitemtracking.WorkItemErrorPolicyValues.Omit
	for start := 0; start < len(ids); start += maxWorkItemsPerRequest {
		batch := ids[start:min(start+maxWorkItemsPerRequest, len(ids))]
		response, err := c.workItemClient.GetWorkItems(ctx, workitemtracking.GetWorkItemsArgs{
			Ids:         &batch,
			ErrorPolicy: &errorPolicy,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get work items: %w", err)
		}
		if response == nil {
			continue
		}
		for _, workItem := range *response {
			// The Omit error policy returns an empty entry for a deleted or inaccessible work item.
			if workItem.Id != nil {
				workItems = append(workItems, workItem)
			}
		}
	}
	return workItems, nil
}

// LinkPullRequest adds the pull request to the work item's links.
func (c *Client) LinkPullRequest(ctx context.Context, workItemID int, projectID, repoID string, pullRequestID int) error {
	operation := webapi.OperationValues.Add
	path := appendRelationPath
	document := []webapi.JsonPatchOperation{{
		Op:   &operation,
		Path: &path,
		Value: map[string]interface{}{
			"rel":        artifactLinkRelation,
			"url":        fmt.Sprintf(pullRequestArtifactURLFormat, projectID, repoID, pullRequestID),
			"attributes": map[string]interface{}{"name": pullRequestLinkName},
		},
	}}
	if _, err := c.workItemClient.UpdateWorkItem(ctx, workitemtracking.UpdateWorkItemArgs{Id: &workItemID, Document: &document}); err != nil {
		return fmt.Errorf("failed to link work item %d to PR %d: %w", workItemID, pullRequestID, err)
	}
	return nil
}
