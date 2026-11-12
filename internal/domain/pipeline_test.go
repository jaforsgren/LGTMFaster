package domain

import "testing"

func TestPipelineStatus_IsFinished(t *testing.T) {
	tests := []struct {
		status   PipelineStatus
		finished bool
	}{
		{PipelineStatusQueued, false},
		{PipelineStatusRunning, false},
		{PipelineStatusWaitingForApproval, false},
		{PipelineStatusUnknown, false},
		{PipelineStatusSucceeded, true},
		{PipelineStatusPartiallySucceeded, true},
		{PipelineStatusFailed, true},
		{PipelineStatusCanceled, true},
		{PipelineStatusSkipped, true},
	}

	for _, tt := range tests {
		if got := tt.status.IsFinished(); got != tt.finished {
			t.Errorf("%s.IsFinished() = %v, want %v", tt.status, got, tt.finished)
		}
	}
}

func TestPipelineStage_AwaitsApproval(t *testing.T) {
	if (PipelineStage{}).AwaitsApproval() {
		t.Error("expected stage without approval ID not to await approval")
	}
	if !(PipelineStage{PendingApprovalID: "id"}).AwaitsApproval() {
		t.Error("expected stage with approval ID to await approval")
	}
}

func TestFavoriteKeys_AreDistinctPerKindAndPAT(t *testing.T) {
	keys := []string{
		ProjectFavoriteKey("pat-a", "Proj"),
		ProjectFavoriteKey("pat-b", "Proj"),
		RepositoryFavoriteKey("pat-a", "Proj", "api"),
		PipelineFavoriteKey("pat-a", "Proj", 1),
		PipelineRunFavoriteKey("pat-a", "Proj", 1),
	}

	seen := make(map[string]bool)
	for _, key := range keys {
		if seen[key] {
			t.Errorf("duplicate favorite key %s", key)
		}
		seen[key] = true
	}
}

func TestFavoriteKey_MethodsMatchConstructors(t *testing.T) {
	pipeline := Pipeline{ID: 3, Project: "Proj", PATID: "pat"}
	run := PipelineRun{ID: 9, Project: "Proj", PATID: "pat"}

	if pipeline.FavoriteKey() != PipelineFavoriteKey("pat", "Proj", 3) {
		t.Error("pipeline favorite key mismatch")
	}
	if run.FavoriteKey() != PipelineRunFavoriteKey("pat", "Proj", 9) {
		t.Error("run favorite key mismatch")
	}
}

func TestPipelineSortMode_OrDefault(t *testing.T) {
	for _, mode := range PipelineSortModes {
		if mode.OrDefault() != mode {
			t.Errorf("expected %s to be kept", mode)
		}
	}
	if PipelineSortMode("").OrDefault() != PipelineSortFavorites || PipelineSortMode("bogus").OrDefault() != PipelineSortFavorites {
		t.Error("expected unknown modes to default to favorites")
	}
}

func TestActiveStage_PrefersAStageWaitingForApproval(t *testing.T) {
	done := PipelineStage{Name: "Build", Status: PipelineStatusSucceeded}
	running := PipelineStage{Name: "Test", Status: PipelineStatusRunning}
	waiting := PipelineStage{Name: "Prod", Status: PipelineStatusWaitingForApproval, PendingApprovalID: "id"}

	if stage, found := ActiveStage([]PipelineStage{done, running, waiting}); !found || stage.Name != "Prod" {
		t.Errorf("expected the stage waiting for approval, got %+v", stage)
	}
	if stage, found := ActiveStage([]PipelineStage{done, running}); !found || stage.Name != "Test" {
		t.Errorf("expected the running stage, got %+v", stage)
	}
	if _, found := ActiveStage([]PipelineStage{done}); found {
		t.Error("expected no active stage when every stage is done")
	}
}
