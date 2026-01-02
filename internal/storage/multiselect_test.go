package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestConfigMigration(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	configPath := filepath.Join(tmpDir, configDir, configFile)
	os.MkdirAll(filepath.Dir(configPath), 0700)

	oldConfig := Config{
		PATs: []domain.PAT{
			{
				ID:       "pat-1",
				Name:     "GitHub PAT",
				Token:    "ghp_test",
				Provider: domain.ProviderGitHub,
				Username: "user1",
				IsActive: true,
			},
			{
				ID:       "pat-2",
				Name:     "Azure PAT",
				Token:    "azure_test",
				Provider: domain.ProviderAzureDevOps,
				Username: "user2",
				IsActive: false,
			},
		},
		ActivePAT: "pat-1",
	}

	data, _ := json.MarshalIndent(oldConfig, "", "  ")
	os.WriteFile(configPath, data, 0600)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	if len(repo.config.SelectedPATs) != 1 {
		t.Errorf("Expected 1 selected PAT after migration, got %d", len(repo.config.SelectedPATs))
	}

	if repo.config.SelectedPATs[0] != "pat-1" {
		t.Errorf("Expected selected PAT to be pat-1, got %s", repo.config.SelectedPATs[0])
	}

	if repo.config.PrimaryPAT != "pat-1" {
		t.Errorf("Expected primary PAT to be pat-1, got %s", repo.config.PrimaryPAT)
	}

	data, _ = os.ReadFile(configPath)
	var savedConfig Config
	json.Unmarshal(data, &savedConfig)

	if len(savedConfig.SelectedPATs) != 1 || savedConfig.SelectedPATs[0] != "pat-1" {
		t.Error("Migration was not saved to disk")
	}
}

func TestTogglePATSelection(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	pat1 := domain.PAT{ID: "pat-1", Name: "PAT 1", Provider: domain.ProviderGitHub, Username: "user1"}
	pat2 := domain.PAT{ID: "pat-2", Name: "PAT 2", Provider: domain.ProviderGitHub, Username: "user2"}
	pat3 := domain.PAT{ID: "pat-3", Name: "PAT 3", Provider: domain.ProviderAzureDevOps, Username: "user3"}

	repo.SavePAT(pat1)
	repo.SavePAT(pat2)
	repo.SavePAT(pat3)

	err = repo.TogglePATSelection("pat-1")
	if err != nil {
		t.Fatalf("Failed to toggle first PAT: %v", err)
	}

	if len(repo.config.SelectedPATs) != 1 || repo.config.SelectedPATs[0] != "pat-1" {
		t.Error("First toggle should select PAT and set as primary")
	}

	if repo.config.PrimaryPAT != "pat-1" {
		t.Errorf("Primary PAT should be pat-1, got %s", repo.config.PrimaryPAT)
	}

	err = repo.TogglePATSelection("pat-2")
	if err != nil {
		t.Fatalf("Failed to toggle second PAT: %v", err)
	}

	if len(repo.config.SelectedPATs) != 2 {
		t.Errorf("Expected 2 selected PATs, got %d", len(repo.config.SelectedPATs))
	}

	if repo.config.PrimaryPAT != "pat-1" {
		t.Error("Primary PAT should remain pat-1")
	}

	err = repo.TogglePATSelection("pat-1")
	if err != nil {
		t.Fatalf("Failed to deselect PAT: %v", err)
	}

	if len(repo.config.SelectedPATs) != 1 || repo.config.SelectedPATs[0] != "pat-2" {
		t.Error("Deselection failed")
	}

	if repo.config.PrimaryPAT != "pat-2" {
		t.Error("Primary PAT should change to pat-2 when pat-1 (primary) is deselected")
	}
}

func TestCannotDeselectLastPAT(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	pat := domain.PAT{ID: "pat-1", Name: "PAT 1", Provider: domain.ProviderGitHub, Username: "user1"}
	repo.SavePAT(pat)

	err = repo.TogglePATSelection("pat-1")
	if err != nil {
		t.Fatalf("Failed to select PAT: %v", err)
	}

	err = repo.TogglePATSelection("pat-1")
	if err == nil {
		t.Error("Should not allow deselecting the last PAT")
	}

	if len(repo.config.SelectedPATs) != 1 {
		t.Error("Last PAT should remain selected")
	}
}

func TestGetSelectedPATs(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	pat1 := domain.PAT{ID: "pat-1", Name: "PAT 1", Provider: domain.ProviderGitHub, Username: "user1"}
	pat2 := domain.PAT{ID: "pat-2", Name: "PAT 2", Provider: domain.ProviderAzureDevOps, Username: "user2"}

	repo.SavePAT(pat1)
	repo.SavePAT(pat2)

	repo.TogglePATSelection("pat-1")
	repo.TogglePATSelection("pat-2")

	selected, err := repo.GetSelectedPATs()
	if err != nil {
		t.Fatalf("Failed to get selected PATs: %v", err)
	}

	if len(selected) != 2 {
		t.Errorf("Expected 2 selected PATs, got %d", len(selected))
	}

	primaryCount := 0
	selectedCount := 0
	for _, pat := range selected {
		if pat.IsPrimary {
			primaryCount++
		}
		if pat.IsSelected {
			selectedCount++
		}
	}

	if primaryCount != 1 {
		t.Errorf("Expected exactly 1 primary PAT, got %d", primaryCount)
	}

	if selectedCount != 2 {
		t.Errorf("Expected 2 selected PATs, got %d", selectedCount)
	}
}

func TestSetSelectedPATs(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	pat1 := domain.PAT{ID: "pat-1", Name: "PAT 1", Provider: domain.ProviderGitHub, Username: "user1"}
	pat2 := domain.PAT{ID: "pat-2", Name: "PAT 2", Provider: domain.ProviderGitHub, Username: "user2"}
	pat3 := domain.PAT{ID: "pat-3", Name: "PAT 3", Provider: domain.ProviderAzureDevOps, Username: "user3"}

	repo.SavePAT(pat1)
	repo.SavePAT(pat2)
	repo.SavePAT(pat3)

	err = repo.SetSelectedPATs([]string{"pat-1", "pat-3"}, "pat-3")
	if err != nil {
		t.Fatalf("Failed to set selected PATs: %v", err)
	}

	if len(repo.config.SelectedPATs) != 2 {
		t.Errorf("Expected 2 selected PATs, got %d", len(repo.config.SelectedPATs))
	}

	if repo.config.PrimaryPAT != "pat-3" {
		t.Errorf("Expected primary PAT to be pat-3, got %s", repo.config.PrimaryPAT)
	}

	for _, pat := range repo.config.PATs {
		if pat.ID == "pat-2" {
			if pat.IsSelected {
				t.Error("PAT 2 should not be selected")
			}
		}
		if pat.ID == "pat-3" {
			if !pat.IsPrimary {
				t.Error("PAT 3 should be primary")
			}
		}
	}
}

func TestSetSelectedPATsValidation(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	pat1 := domain.PAT{ID: "pat-1", Name: "PAT 1", Provider: domain.ProviderGitHub, Username: "user1"}
	repo.SavePAT(pat1)

	err = repo.SetSelectedPATs([]string{}, "pat-1")
	if err == nil {
		t.Error("Should not allow empty selection")
	}

	err = repo.SetSelectedPATs([]string{"nonexistent"}, "nonexistent")
	if err == nil {
		t.Error("Should not allow selecting nonexistent PAT")
	}

	err = repo.SetSelectedPATs([]string{"pat-1"}, "nonexistent")
	if err == nil {
		t.Error("Should not allow primary PAT not in selected set")
	}
}

func TestGetPrimaryPAT(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	pat1 := domain.PAT{ID: "pat-1", Name: "PAT 1", Provider: domain.ProviderGitHub, Username: "user1"}
	pat2 := domain.PAT{ID: "pat-2", Name: "PAT 2", Provider: domain.ProviderAzureDevOps, Username: "user2"}

	repo.SavePAT(pat1)
	repo.SavePAT(pat2)
	repo.SetSelectedPATs([]string{"pat-1", "pat-2"}, "pat-2")

	primary, err := repo.GetPrimaryPAT()
	if err != nil {
		t.Fatalf("Failed to get primary PAT: %v", err)
	}

	if primary.ID != "pat-2" {
		t.Errorf("Expected primary PAT to be pat-2, got %s", primary.ID)
	}

	if !primary.IsPrimary {
		t.Error("Primary PAT should have IsPrimary=true")
	}
}

func TestSetPrimaryPAT(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	pat1 := domain.PAT{ID: "pat-1", Name: "PAT 1", Provider: domain.ProviderGitHub, Username: "user1"}
	pat2 := domain.PAT{ID: "pat-2", Name: "PAT 2", Provider: domain.ProviderGitHub, Username: "user2"}

	repo.SavePAT(pat1)
	repo.SavePAT(pat2)

	repo.TogglePATSelection("pat-1")
	repo.TogglePATSelection("pat-2")

	err = repo.SetPrimaryPAT("pat-2")
	if err != nil {
		t.Fatalf("Failed to set primary PAT: %v", err)
	}

	if repo.config.PrimaryPAT != "pat-2" {
		t.Errorf("Expected primary PAT to be pat-2, got %s", repo.config.PrimaryPAT)
	}

	pat3 := domain.PAT{ID: "pat-3", Name: "PAT 3", Provider: domain.ProviderAzureDevOps, Username: "user3"}
	repo.SavePAT(pat3)

	err = repo.SetPrimaryPAT("pat-3")
	if err == nil {
		t.Error("Should not allow setting non-selected PAT as primary")
	}
}

func TestDeletePATCleansUpSelection(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	repo, err := NewLocalRepository()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	pat1 := domain.PAT{ID: "pat-1", Name: "PAT 1", Provider: domain.ProviderGitHub, Username: "user1"}
	pat2 := domain.PAT{ID: "pat-2", Name: "PAT 2", Provider: domain.ProviderGitHub, Username: "user2"}
	pat3 := domain.PAT{ID: "pat-3", Name: "PAT 3", Provider: domain.ProviderAzureDevOps, Username: "user3"}

	repo.SavePAT(pat1)
	repo.SavePAT(pat2)
	repo.SavePAT(pat3)

	repo.SetSelectedPATs([]string{"pat-1", "pat-2", "pat-3"}, "pat-1")

	err = repo.DeletePAT("pat-1")
	if err != nil {
		t.Fatalf("Failed to delete PAT: %v", err)
	}

	if len(repo.config.SelectedPATs) != 2 {
		t.Errorf("Expected 2 selected PATs after deletion, got %d", len(repo.config.SelectedPATs))
	}

	if repo.config.PrimaryPAT == "pat-1" {
		t.Error("Primary PAT should have changed after deleting primary")
	}

	if repo.config.PrimaryPAT != "pat-2" && repo.config.PrimaryPAT != "pat-3" {
		t.Errorf("Expected new primary to be pat-2 or pat-3, got %s", repo.config.PrimaryPAT)
	}
}
