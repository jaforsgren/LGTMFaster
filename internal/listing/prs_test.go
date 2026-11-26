package listing

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

var errListFailed = errors.New("list failed")

func TestPrintPRs_GroupsByCategoryNewestFirst(t *testing.T) {
	now := time.Now()
	prs := []domain.PullRequest{
		{Number: 1, Title: "other", Category: domain.PRCategoryOther, UpdatedAt: now, CreatedAt: now},
		{Number: 2, Title: "older mine", Category: domain.PRCategoryAuthored, UpdatedAt: now.Add(-time.Hour), CreatedAt: now},
		{Number: 3, Title: "review me", Category: domain.PRCategoryAssigned, UpdatedAt: now, CreatedAt: now, ApprovalStatus: domain.ApprovalStatusChangesRequested},
		{Number: 4, Title: "newer mine", Category: domain.PRCategoryAuthored, UpdatedAt: now, CreatedAt: now, IsDraft: true},
	}
	var out bytes.Buffer

	err := PrintPRs(context.Background(), []domain.PAT{selectedPAT("ado")}, factoryOf(map[string]domain.Provider{"ado": listingProvider{prs: prs}}), domain.Settings{}, &out)

	if err != nil {
		t.Fatal(err)
	}
	text := out.String()
	order := []string{"Created by you (2)", "#4", "[draft] newer mine", "#2", "Assigned to you (1)", "#3", "changes requested", "Other (1)", "#1"}
	last := -1
	for _, want := range order {
		at := strings.Index(text, want)
		if at <= last {
			t.Fatalf("expected %q after the previous entries in:\n%s", want, text)
		}
		last = at
	}
}

func TestPrintPRs_HidesOldPRsOfOthersWhenTheSettingsSaySo(t *testing.T) {
	old := time.Now().AddDate(0, 0, -60)
	prs := []domain.PullRequest{{Number: 9, Title: "stale", Category: domain.PRCategoryOther, CreatedAt: old, UpdatedAt: old}}
	factory := factoryOf(map[string]domain.Provider{"ado": listingProvider{prs: prs}})
	var out bytes.Buffer

	if err := PrintPRs(context.Background(), []domain.PAT{selectedPAT("ado")}, factory, domain.Settings{HideOldOtherPRs: true, HideOldOtherPRsDays: 30}, &out); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), noPRsText) {
		t.Errorf("expected the old PR hidden, got:\n%s", out.String())
	}
}

func TestPrintPRs_OnlyAsksTheSelectedPATs(t *testing.T) {
	pats := []domain.PAT{selectedPAT("ado"), {ID: "gh", Name: "gh"}}
	factory := factoryOf(map[string]domain.Provider{"ado": listingProvider{}})
	var out bytes.Buffer

	if err := PrintPRs(context.Background(), pats, factory, domain.Settings{}, &out); err != nil {
		t.Errorf("expected only the selected PAT asked, got %v", err)
	}
}

func TestPrintPRs_PrintsTheOthersWhenOnePATFails(t *testing.T) {
	now := time.Now()
	factory := factoryOf(map[string]domain.Provider{
		"ado":    listingProvider{prs: []domain.PullRequest{{Number: 5, Title: "kept", Category: domain.PRCategoryAuthored, CreatedAt: now, UpdatedAt: now}}},
		"broken": listingProvider{err: errListFailed},
	})
	var out bytes.Buffer

	err := PrintPRs(context.Background(), []domain.PAT{selectedPAT("ado"), selectedPAT("broken")}, factory, domain.Settings{}, &out)

	if !errors.Is(err, errListFailed) || !strings.Contains(err.Error(), "broken:") {
		t.Errorf("expected the failing PAT named, got %v", err)
	}
	if !strings.Contains(out.String(), "kept") {
		t.Errorf("expected the other PAT's PRs printed, got:\n%s", out.String())
	}
}

func TestPrintPRs_NeedsASelectedPAT(t *testing.T) {
	err := PrintPRs(context.Background(), []domain.PAT{{ID: "gh"}}, factoryOf(nil), domain.Settings{}, &bytes.Buffer{})

	if !errors.Is(err, errNoSelectedPATs) {
		t.Errorf("expected no selected PATs refused, got %v", err)
	}
}
