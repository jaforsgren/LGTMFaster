package domain

import (
	"testing"
	"time"
)

func TestSettingsWithDefaults_FillsOnlyMissingValues(t *testing.T) {
	got := Settings{PipelineAutoRefreshSeconds: 30}.WithDefaults()

	if got.PipelineAutoRefreshSeconds != 30 {
		t.Errorf("expected the configured interval kept, got %d", got.PipelineAutoRefreshSeconds)
	}
	if got.HideOldOtherPRsDays != DefaultHideOldOtherPRsDays {
		t.Errorf("expected the missing days defaulted, got %d", got.HideOldOtherPRsDays)
	}
}

func TestRecentBranchWindow_DefaultsTo60Minutes(t *testing.T) {
	if got := (Settings{}).RecentBranchWindow(); got != 60*time.Minute {
		t.Errorf("expected 60m, got %s", got)
	}
	if got := (Settings{RecentBranchMinutes: 15}).RecentBranchWindow(); got != 15*time.Minute {
		t.Errorf("expected the configured 15m, got %s", got)
	}
}

func TestPipelineAutoRefreshInterval_DefaultsTo15Seconds(t *testing.T) {
	if got := (Settings{}).PipelineAutoRefreshInterval(); got != 15*time.Second {
		t.Errorf("expected 15s, got %s", got)
	}
	if got := (Settings{PipelineAutoRefreshSeconds: -3}).PipelineAutoRefreshInterval(); got != 15*time.Second {
		t.Errorf("expected an invalid interval to fall back to 15s, got %s", got)
	}
}
