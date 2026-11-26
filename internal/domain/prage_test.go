package domain

import (
	"testing"
	"time"
)

func TestWithoutOldOtherPRs_KeepsYourOwnAndRecentOnes(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -31)
	prs := []PullRequest{
		{Number: 1, Category: PRCategoryOther, CreatedAt: old},
		{Number: 2, Category: PRCategoryOther, CreatedAt: now.AddDate(0, 0, -2)},
		{Number: 3, Category: PRCategoryAuthored, CreatedAt: old},
		{Number: 4, Category: PRCategoryAssigned, CreatedAt: old},
	}

	got := WithoutOldOtherPRs(prs, 30, now)

	if len(got) != 3 || got[0].Number != 2 || got[1].Number != 3 || got[2].Number != 4 {
		t.Errorf("expected PRs 2, 3 and 4, got %+v", got)
	}
}
