package domain

import "testing"

func TestPRSortMode_OrDefault(t *testing.T) {
	for _, mode := range PRSortModes {
		if mode.OrDefault() != mode {
			t.Errorf("expected %s to be kept", mode)
		}
	}
	if PRSortMode("").OrDefault() != PRSortDefault || PRSortMode("bogus").OrDefault() != PRSortDefault {
		t.Error("expected unknown modes to fall back to the default")
	}
}
