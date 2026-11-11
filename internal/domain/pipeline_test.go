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
