package common

import "testing"

func TestCommitSummary_IsTheFirstLineTrimmed(t *testing.T) {
	if got := CommitSummary("  feature: add login \n\nlonger body"); got != "feature: add login" {
		t.Errorf("expected the trimmed first line, got %q", got)
	}
}
