package headless

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

// creatingProvider records the pull request created and the work items linked.
type creatingProvider struct {
	domain.Provider
	created  *domain.NewPullRequest
	linked   []int
	linkErrs map[int]error
}

func (p *creatingProvider) CreatePullRequest(_ context.Context, pr domain.NewPullRequest) (*domain.PullRequest, error) {
	p.created = &pr
	return &domain.PullRequest{Number: 7, Title: pr.Title, URL: "https://example.test/pr/7", Repository: domain.Repo{FullName: pr.RepositoryFullName}}, nil
}

func (p *creatingProvider) LinkWorkItem(_ context.Context, _ domain.PRIdentifier, workItemID int) error {
	p.linked = append(p.linked, workItemID)
	return p.linkErrs[workItemID]
}

var testRemote = domain.GitRemote{Provider: domain.ProviderAzureDevOps, Organization: "contoso", RepositoryFullName: "Proj/api"}

func testPATs() []domain.PAT {
	return []domain.PAT{
		{ID: "gh", Provider: domain.ProviderGitHub, IsSelected: true},
		{ID: "contoso", Provider: domain.ProviderAzureDevOps, Organization: "contoso"},
	}
}

func factoryFor(t *testing.T, provider domain.Provider, wantPATID string) ProviderFactory {
	return func(pat domain.PAT) (domain.Provider, error) {
		if pat.ID != wantPATID {
			t.Errorf("expected the %s PAT, got %s", wantPATID, pat.ID)
		}
		return provider, nil
	}
}

func TestCreate_OpensThePullRequestIntoMainAndLinksTheWorkItems(t *testing.T) {
	provider := &creatingProvider{}
	draft := domain.NewPullRequestDraft{Remote: testRemote, SourceBranch: "feature/add-login", Description: "Adds it", WorkItemIDs: []int{1234, 5678}}
	var out bytes.Buffer

	err := Create(context.Background(), testPATs(), factoryFor(t, provider, "contoso"), draft, &out)

	if err != nil {
		t.Fatal(err)
	}
	want := domain.NewPullRequest{RepositoryFullName: "Proj/api", SourceBranch: "feature/add-login", TargetBranch: DefaultTargetBranch, Title: "Add login", Description: "Adds it"}
	if *provider.created != want {
		t.Errorf("expected %+v, got %+v", want, *provider.created)
	}
	if !slices.Equal(provider.linked, []int{1234, 5678}) {
		t.Errorf("expected both work items linked, got %v", provider.linked)
	}
	for _, line := range []string{"✔ Created pull request #7 in Proj/api", "Add login", "feature/add-login → main", "🔗 https://example.test/pr/7", "✔ Linked work item #1234", "✔ Linked work item #5678"} {
		if !strings.Contains(out.String(), line) {
			t.Errorf("expected %q in %q", line, out.String())
		}
	}
}

func TestCreate_MergesIntoTheGivenTarget(t *testing.T) {
	provider := &creatingProvider{}
	draft := domain.NewPullRequestDraft{Remote: testRemote, SourceBranch: "fix", TargetBranch: "develop"}

	if err := Create(context.Background(), testPATs(), factoryFor(t, provider, "contoso"), draft, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if provider.created.TargetBranch != "develop" {
		t.Errorf("expected develop, got %q", provider.created.TargetBranch)
	}
}

func TestCreate_ReportsWorkItemsThatCouldNotBeLinked(t *testing.T) {
	provider := &creatingProvider{linkErrs: map[int]error{1234: errors.New("does not exist")}}
	draft := domain.NewPullRequestDraft{Remote: testRemote, SourceBranch: "fix", WorkItemIDs: []int{1234, 5678}}
	var out bytes.Buffer

	err := Create(context.Background(), testPATs(), factoryFor(t, provider, "contoso"), draft, &out)

	if !errors.Is(err, ErrNotAllLinked) || !strings.Contains(err.Error(), "could not link #1234: does not exist") {
		t.Errorf("expected the failed link reported, got %v", err)
	}
	if !strings.Contains(out.String(), "✘ Could not link work item #1234: does not exist") || !strings.Contains(out.String(), "✔ Linked work item #5678") {
		t.Errorf("expected 1234 failed and 5678 linked, got %q", out.String())
	}
}

func TestCreate_NeedsAPATForTheRepository(t *testing.T) {
	draft := domain.NewPullRequestDraft{Remote: domain.GitRemote{Provider: domain.ProviderAzureDevOps, Organization: "fabrikam"}}

	err := Create(context.Background(), testPATs(), factoryFor(t, nil, ""), draft, &bytes.Buffer{})

	if err == nil || !strings.Contains(err.Error(), "fabrikam") {
		t.Errorf("expected the missing PAT reported, got %v", err)
	}
}

func TestSelectedPAT_OnlyUsesSelectedPATs(t *testing.T) {
	draft := domain.NewPullRequestDraft{Remote: testRemote}

	if _, err := SelectedPAT(testPATs(), draft); err == nil || !strings.Contains(err.Error(), ":pats") {
		t.Errorf("expected the unselected contoso PAT refused, got %v", err)
	}
}

func TestCreate_RefusesAPullRequestIntoItsOwnBranch(t *testing.T) {
	draft := domain.NewPullRequestDraft{Remote: testRemote, SourceBranch: DefaultTargetBranch}

	err := Create(context.Background(), testPATs(), factoryFor(t, nil, ""), draft, &bytes.Buffer{})

	if !errors.Is(err, errSameBranch) {
		t.Errorf("expected main into main refused, got %v", err)
	}
}

// urlLessProvider creates pull requests whose response has no web URL.
type urlLessProvider struct{ creatingProvider }

func (p *urlLessProvider) CreatePullRequest(ctx context.Context, pr domain.NewPullRequest) (*domain.PullRequest, error) {
	created, err := p.creatingProvider.CreatePullRequest(ctx, pr)
	created.URL = ""
	return created, err
}

func TestCreate_PrintsALinkBuiltFromTheRemoteWhenTheProviderGivesNone(t *testing.T) {
	var out bytes.Buffer
	draft := domain.NewPullRequestDraft{Remote: testRemote, SourceBranch: "fix"}

	if err := Create(context.Background(), testPATs(), factoryFor(t, &urlLessProvider{}, "contoso"), draft, &out); err != nil {
		t.Fatal(err)
	}
	if want := "🔗 https://dev.azure.com/contoso/Proj/_git/api/pullrequest/7"; !strings.Contains(out.String(), want) {
		t.Errorf("expected %q in %q", want, out.String())
	}
}
