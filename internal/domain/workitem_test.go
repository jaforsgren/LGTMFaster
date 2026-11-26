package domain

import (
	"slices"
	"testing"
)

func TestParseWorkItemIDs(t *testing.T) {
	tests := []struct {
		text string
		want []int
	}{
		{"", []int{}},
		{"1234", []int{1234}},
		{"#1234", []int{1234}},
		{"1234, 5678", []int{1234, 5678}},
		{"#1234 #5678 1234", []int{1234, 5678}},
	}
	for _, tt := range tests {
		got, err := ParseWorkItemIDs(tt.text)
		if err != nil || !slices.Equal(got, tt.want) {
			t.Errorf("ParseWorkItemIDs(%q) = %v, %v; want %v", tt.text, got, err, tt.want)
		}
	}
}

func TestFormatWorkItemIDs_IsReadBackByParseWorkItemIDs(t *testing.T) {
	ids := []int{1234, 5678}

	formatted := FormatWorkItemIDs(ids)
	parsed, err := ParseWorkItemIDs(formatted)

	if formatted != "1234, 5678" || err != nil || !slices.Equal(parsed, ids) {
		t.Errorf("expected 1234, 5678 to round-trip, got %q -> %v, %v", formatted, parsed, err)
	}
}

func TestParseWorkItemIDs_RejectsWhatIsNotANumber(t *testing.T) {
	for _, text := range []string{"login", "12a", "#", "0", "-3"} {
		if _, err := ParseWorkItemIDs(text); err == nil {
			t.Errorf("expected %q rejected", text)
		}
	}
}
