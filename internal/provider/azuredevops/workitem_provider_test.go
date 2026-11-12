package azuredevops

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

type mockWorkItemClient struct {
	queryResult    *workitemtracking.WorkItemQueryResult
	lastQuery      string
	workItems      map[int]workitemtracking.WorkItem
	requestedIDs   [][]int
	lastUpdateArgs *workitemtracking.UpdateWorkItemArgs
	err            error
}

func (m *mockWorkItemClient) QueryByWiql(ctx context.Context, args workitemtracking.QueryByWiqlArgs) (*workitemtracking.WorkItemQueryResult, error) {
	m.lastQuery = *args.Wiql.Query
	return m.queryResult, m.err
}

func (m *mockWorkItemClient) GetWorkItems(ctx context.Context, args workitemtracking.GetWorkItemsArgs) (*[]workitemtracking.WorkItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.requestedIDs = append(m.requestedIDs, *args.Ids)
	var found []workitemtracking.WorkItem
	for _, id := range *args.Ids {
		found = append(found, m.workItems[id])
	}
	return &found, nil
}

func (m *mockWorkItemClient) UpdateWorkItem(ctx context.Context, args workitemtracking.UpdateWorkItemArgs) (*workitemtracking.WorkItem, error) {
	m.lastUpdateArgs = &args
	return nil, m.err
}

func testWorkItem(id int, fields map[string]interface{}) workitemtracking.WorkItem {
	return workitemtracking.WorkItem{Id: &id, Fields: &fields}
}

func workItemTestProvider(gitClient *mockGitClient, workItemClient *mockWorkItemClient) *Provider {
	return &Provider{
		client:    &Client{gitClient: gitClient, workItemClient: workItemClient, organization: testOrganization},
		repoCache: map[string]*ResolvedRepository{testRepository: {ProjectID: "proj-id", RepoID: "repo-id", CachedAt: time.Now()}},
		cacheTTL:  time.Hour,
	}
}

func TestListLinkedWorkItems_ConvertsTheLinkedWorkItems(t *testing.T) {
	gitClient := &mockGitClient{workItemRefs: &[]webapi.ResourceRef{{Id: strPtr("12")}, {Id: strPtr("7")}}}
	workItemClient := &mockWorkItemClient{workItems: map[int]workitemtracking.WorkItem{
		12: testWorkItem(12, map[string]interface{}{
			workItemFieldType: "Task", workItemFieldTitle: "Add login", workItemFieldState: "Active",
			workItemFieldDescription: "<div>Use <b>OAuth</b></div>",
		}),
		7: testWorkItem(7, map[string]interface{}{
			workItemFieldType: "Bug", workItemFieldTitle: "Crash", workItemFieldReproSteps: "<p>Click save</p>",
		}),
	}}
	provider := workItemTestProvider(gitClient, workItemClient)

	workItems, err := provider.ListLinkedWorkItems(context.Background(), domain.PRIdentifier{Repository: testRepository, Number: 42})
	if err != nil {
		t.Fatal(err)
	}

	if len(workItems) != 2 {
		t.Fatalf("expected both linked work items, got %+v", workItems)
	}
	task, bug := workItems[0], workItems[1]
	if task.ID != 12 || task.Type != "Task" || task.Title != "Add login" || task.State != "Active" || task.Description != "Use OAuth" {
		t.Errorf("expected the task converted with a plain text description, got %+v", task)
	}
	if task.URL != "https://dev.azure.com/myorg/_workitems/edit/12" {
		t.Errorf("expected the work item's web URL, got %q", task.URL)
	}
	if bug.Description != "Click save" {
		t.Errorf("expected a bug's repro steps as its description, got %q", bug.Description)
	}
}

func TestListLinkedWorkItems_NoLinksMakesNoWorkItemRequest(t *testing.T) {
	workItemClient := &mockWorkItemClient{}
	provider := workItemTestProvider(&mockGitClient{workItemRefs: &[]webapi.ResourceRef{}}, workItemClient)

	workItems, err := provider.ListLinkedWorkItems(context.Background(), domain.PRIdentifier{Repository: testRepository, Number: 42})

	if err != nil || len(workItems) != 0 || len(workItemClient.requestedIDs) != 0 {
		t.Errorf("expected no work items and no request, got %+v, %v, %v", workItems, err, workItemClient.requestedIDs)
	}
}

func TestListAssignedWorkItems_QueriesOpenBugsTasksAndStoriesAssignedToTheUser(t *testing.T) {
	first, second := 3, 5
	workItemClient := &mockWorkItemClient{
		queryResult: &workitemtracking.WorkItemQueryResult{WorkItems: &[]workitemtracking.WorkItemReference{{Id: &first}, {Id: &second}}},
		workItems: map[int]workitemtracking.WorkItem{
			3: testWorkItem(3, map[string]interface{}{workItemFieldTitle: "Story"}),
		},
	}
	provider := workItemTestProvider(&mockGitClient{}, workItemClient)

	workItems, err := provider.ListAssignedWorkItems(context.Background(), domain.PRIdentifier{Repository: testRepository, Number: 42})
	if err != nil {
		t.Fatal(err)
	}

	for _, clause := range []string{"@Me", "'Bug', 'Task', 'User Story'", "NOT IN ('Closed', 'Done', 'Removed', 'Completed')"} {
		if !strings.Contains(workItemClient.lastQuery, clause) {
			t.Errorf("expected the query to contain %q, got %q", clause, workItemClient.lastQuery)
		}
	}
	if len(workItems) != 1 || workItems[0].ID != 3 {
		t.Errorf("expected the deleted work item left out, got %+v", workItems)
	}
}

func TestGetWorkItems_SplitsRequestsAtTheAPILimit(t *testing.T) {
	workItemClient := &mockWorkItemClient{workItems: map[int]workitemtracking.WorkItem{}}
	ids := make([]int, maxWorkItemsPerRequest+1)
	for i := range ids {
		ids[i] = i + 1
		workItemClient.workItems[i+1] = testWorkItem(i+1, nil)
	}
	client := &Client{workItemClient: workItemClient}

	workItems, err := client.GetWorkItems(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}

	if len(workItemClient.requestedIDs) != 2 || len(workItemClient.requestedIDs[1]) != 1 || len(workItems) != len(ids) {
		t.Errorf("expected two requests returning every work item, got %d requests and %d work items", len(workItemClient.requestedIDs), len(workItems))
	}
}

func TestLinkWorkItem_AddsAnArtifactLinkToThePullRequest(t *testing.T) {
	workItemClient := &mockWorkItemClient{}
	provider := workItemTestProvider(&mockGitClient{}, workItemClient)

	if err := provider.LinkWorkItem(context.Background(), domain.PRIdentifier{Repository: testRepository, Number: 42}, 12); err != nil {
		t.Fatal(err)
	}

	args := workItemClient.lastUpdateArgs
	if args == nil || *args.Id != 12 || len(*args.Document) != 1 {
		t.Fatalf("expected one patch on work item 12, got %+v", args)
	}
	operation := (*args.Document)[0]
	relation := operation.Value.(map[string]interface{})
	if *operation.Path != appendRelationPath || relation["rel"] != artifactLinkRelation {
		t.Errorf("expected an appended artifact link, got %+v", operation)
	}
	if relation["url"] != "vstfs:///Git/PullRequestId/proj-id%2Frepo-id%2F42" {
		t.Errorf("expected the pull request's artifact URL, got %v", relation["url"])
	}
}

func TestLinkWorkItem_ReportsAPIErrors(t *testing.T) {
	provider := workItemTestProvider(&mockGitClient{}, &mockWorkItemClient{err: errors.New("relation already exists")})

	err := provider.LinkWorkItem(context.Background(), domain.PRIdentifier{Repository: testRepository, Number: 42}, 12)

	if err == nil || !strings.Contains(err.Error(), "relation already exists") {
		t.Errorf("expected the API error, got %v", err)
	}
}
