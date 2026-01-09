package common

import (
	"testing"

	"github.com/google/uuid"
)

func TestGetString(t *testing.T) {
	str := "hello"
	tests := []struct {
		name string
		ptr  *string
		want string
	}{
		{
			name: "non-nil pointer",
			ptr:  &str,
			want: "hello",
		},
		{
			name: "nil pointer",
			ptr:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetString(tt.ptr)
			if got != tt.want {
				t.Errorf("GetString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	trueVal := true
	falseVal := false
	tests := []struct {
		name string
		ptr  *bool
		want bool
	}{
		{
			name: "true pointer",
			ptr:  &trueVal,
			want: true,
		},
		{
			name: "false pointer",
			ptr:  &falseVal,
			want: false,
		},
		{
			name: "nil pointer",
			ptr:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetBool(tt.ptr)
			if got != tt.want {
				t.Errorf("GetBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	val := 42
	zero := 0
	negative := -10
	tests := []struct {
		name string
		ptr  *int
		want int
	}{
		{
			name: "positive value",
			ptr:  &val,
			want: 42,
		},
		{
			name: "zero value",
			ptr:  &zero,
			want: 0,
		},
		{
			name: "negative value",
			ptr:  &negative,
			want: -10,
		},
		{
			name: "nil pointer",
			ptr:  nil,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetInt(tt.ptr)
			if got != tt.want {
				t.Errorf("GetInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGetUUIDString(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name string
		ptr  *uuid.UUID
		want string
	}{
		{
			name: "non-nil UUID",
			ptr:  &id,
			want: id.String(),
		},
		{
			name: "nil UUID",
			ptr:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetUUIDString(tt.ptr)
			if got != tt.want {
				t.Errorf("GetUUIDString() = %q, want %q", got, tt.want)
			}
		})
	}
}
