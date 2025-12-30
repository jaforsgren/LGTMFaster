package common

import (
	"errors"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestParseGitHubIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		wantOwner  string
		wantRepo   string
		wantNumber int
		wantErr    bool
	}{
		{
			name:       "valid identifier",
			identifier: "jaforsgren/lgtmfaster/42",
			wantOwner:  "jaforsgren",
			wantRepo:   "lgtmfaster",
			wantNumber: 42,
			wantErr:    false,
		},
		{
			name:       "invalid format - too few parts",
			identifier: "jaforsgren/lgtmfaster",
			wantErr:    true,
		},
		{
			name:       "invalid format - too many parts",
			identifier: "jaforsgren/lgtmfaster/42/extra",
			wantErr:    true,
		},
		{
			name:       "invalid PR number - non-numeric",
			identifier: "jaforsgren/lgtmfaster/abc",
			wantErr:    true,
		},
		{
			name:       "invalid PR number - zero",
			identifier: "jaforsgren/lgtmfaster/0",
			wantErr:    true,
		},
		{
			name:       "invalid PR number - negative",
			identifier: "jaforsgren/lgtmfaster/-1",
			wantErr:    true,
		},
		{
			name:       "empty owner",
			identifier: "/lgtmfaster/42",
			wantErr:    true,
		},
		{
			name:       "empty repo",
			identifier: "jaforsgren//42",
			wantErr:    true,
		},
		{
			name:       "empty string",
			identifier: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRepo, gotNumber, err := ParseGitHubIdentifier(tt.identifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGitHubIdentifier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && !errors.Is(err, ErrInvalidIdentifierFormat) {
				t.Errorf("ParseGitHubIdentifier() error should wrap ErrInvalidIdentifierFormat, got %v", err)
			}
			if !tt.wantErr {
				if gotOwner != tt.wantOwner {
					t.Errorf("ParseGitHubIdentifier() owner = %v, want %v", gotOwner, tt.wantOwner)
				}
				if gotRepo != tt.wantRepo {
					t.Errorf("ParseGitHubIdentifier() repo = %v, want %v", gotRepo, tt.wantRepo)
				}
				if gotNumber != tt.wantNumber {
					t.Errorf("ParseGitHubIdentifier() number = %v, want %v", gotNumber, tt.wantNumber)
				}
			}
		})
	}
}

func TestParseAzureDevOpsIdentifier(t *testing.T) {
	tests := []struct {
		name        string
		identifier  string
		wantProject string
		wantRepo    string
		wantNumber  int
		wantErr     bool
	}{
		{
			name:        "valid identifier",
			identifier:  "MyProject/MyRepo/123",
			wantProject: "MyProject",
			wantRepo:    "MyRepo",
			wantNumber:  123,
			wantErr:     false,
		},
		{
			name:       "invalid format - too few parts",
			identifier: "MyProject/MyRepo",
			wantErr:    true,
		},
		{
			name:       "invalid format - too many parts",
			identifier: "MyProject/MyRepo/123/extra",
			wantErr:    true,
		},
		{
			name:       "invalid PR number - non-numeric",
			identifier: "MyProject/MyRepo/abc",
			wantErr:    true,
		},
		{
			name:       "invalid PR number - zero",
			identifier: "MyProject/MyRepo/0",
			wantErr:    true,
		},
		{
			name:       "invalid PR number - negative",
			identifier: "MyProject/MyRepo/-1",
			wantErr:    true,
		},
		{
			name:       "empty project",
			identifier: "/MyRepo/123",
			wantErr:    true,
		},
		{
			name:       "empty repo",
			identifier: "MyProject//123",
			wantErr:    true,
		},
		{
			name:       "empty string",
			identifier: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProject, gotRepo, gotNumber, err := ParseAzureDevOpsIdentifier(tt.identifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAzureDevOpsIdentifier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && !errors.Is(err, ErrInvalidIdentifierFormat) {
				t.Errorf("ParseAzureDevOpsIdentifier() error should wrap ErrInvalidIdentifierFormat, got %v", err)
			}
			if !tt.wantErr {
				if gotProject != tt.wantProject {
					t.Errorf("ParseAzureDevOpsIdentifier() project = %v, want %v", gotProject, tt.wantProject)
				}
				if gotRepo != tt.wantRepo {
					t.Errorf("ParseAzureDevOpsIdentifier() repo = %v, want %v", gotRepo, tt.wantRepo)
				}
				if gotNumber != tt.wantNumber {
					t.Errorf("ParseAzureDevOpsIdentifier() number = %v, want %v", gotNumber, tt.wantNumber)
				}
			}
		})
	}
}

func TestFormatPRIdentifier(t *testing.T) {
	tests := []struct {
		name string
		id   domain.PRIdentifier
		want string
	}{
		{
			name: "GitHub format",
			id: domain.PRIdentifier{
				Provider:   domain.ProviderGitHub,
				Repository: "jaforsgren/lgtmfaster",
				Number:     42,
			},
			want: "jaforsgren/lgtmfaster/42",
		},
		{
			name: "Azure DevOps format",
			id: domain.PRIdentifier{
				Provider:   domain.ProviderAzureDevOps,
				Repository: "MyProject/MyRepo",
				Number:     123,
			},
			want: "MyProject/MyRepo/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatPRIdentifier(tt.id); got != tt.want {
				t.Errorf("FormatPRIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}
