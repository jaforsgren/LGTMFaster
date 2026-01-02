package common

import (
	"errors"
	"testing"
)

func TestExtractErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "GitHub API error with Message field",
			err:      errors.New("POST https://api.github.com/repos/jaforsgren/LGTMFaster/pulls/6/reviews: 422 Unprocessable Entity [{Resource: Field: Code: Message:Review Can not request changes on your own pull request}]"),
			expected: "Review Can not request changes on your own pull request",
		},
		{
			name:     "GitHub API error with different message",
			err:      errors.New("POST https://api.github.com/repos/owner/repo/pulls/1/reviews: 422 Unprocessable Entity [{Resource:PullRequest Field:base Code:invalid Message:Base branch does not exist}]"),
			expected: "Base branch does not exist",
		},
		{
			name:     "Simple error message",
			err:      errors.New("connection timeout"),
			expected: "connection timeout",
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "Error with colon in URL",
			err:      errors.New("failed to create review: some error occurred"),
			expected: "failed to create review: some error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractErrorMessage(tt.err)
			if result != tt.expected {
				t.Errorf("ExtractErrorMessage() = %q, want %q", result, tt.expected)
			}
		})
	}
}
