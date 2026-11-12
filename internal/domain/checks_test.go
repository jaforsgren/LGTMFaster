package domain

import "testing"

func TestSummarizeChecks_FailingOutranksPendingOutranksPassing(t *testing.T) {
	tests := []struct {
		checks   []Check
		expected CheckState
	}{
		{nil, CheckStateNone},
		{[]Check{{State: CheckStatePassing}}, CheckStatePassing},
		{[]Check{{State: CheckStatePassing}, {State: CheckStatePending}}, CheckStatePending},
		{[]Check{{State: CheckStatePending}, {Name: "lint", State: CheckStateFailing}}, CheckStateFailing},
	}
	for _, tt := range tests {
		if got := SummarizeChecks(tt.checks).State; got != tt.expected {
			t.Errorf("SummarizeChecks(%+v) = %s, want %s", tt.checks, got, tt.expected)
		}
	}

	failing := SummarizeChecks([]Check{{Name: "build", State: CheckStatePassing}, {Name: "lint", State: CheckStateFailing}}).Failing()
	if len(failing) != 1 || failing[0] != "lint" {
		t.Errorf("expected lint failing, got %v", failing)
	}
}
