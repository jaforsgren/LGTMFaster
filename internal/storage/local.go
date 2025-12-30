package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
)

const (
	configDir  = ".lgtmfaster"
	configFile = "config.json"
)

type LocalRepository struct {
	configPath string
	config     *Config
	mu         sync.RWMutex
}

func NewLocalRepository() (*LocalRepository, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, configDir, configFile)

	repo := &LocalRepository{
		configPath: configPath,
		config:     &Config{PATs: []domain.PAT{}},
	}

	if err := repo.ensureConfigDir(); err != nil {
		return nil, err
	}

	if err := repo.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return repo, nil
}

func (r *LocalRepository) ensureConfigDir() error {
	dir := filepath.Dir(r.configPath)
	return os.MkdirAll(dir, 0700)
}

func (r *LocalRepository) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	logger.LogFileOpen(r.configPath)
	data, err := os.ReadFile(r.configPath)
	if err != nil {
		logger.LogError("LOAD", r.configPath, err)
		return err
	}

	if err := json.Unmarshal(data, r.config); err != nil {
		logger.LogError("UNMARSHAL", r.configPath, err)
		return err
	}

	if len(r.config.SelectedPATs) == 0 && r.config.ActivePAT != "" {
		logger.Log("Migrating old config format: ActivePAT=%s -> SelectedPATs", r.config.ActivePAT)
		r.config.SelectedPATs = []string{r.config.ActivePAT}
		r.config.PrimaryPAT = r.config.ActivePAT
		r.mu.Unlock()
		if err := r.save(); err != nil {
			r.mu.Lock()
			logger.LogError("MIGRATION_SAVE", r.configPath, err)
			return err
		}
		r.mu.Lock()
		logger.Log("Config migration completed successfully")
	}

	logger.Log("Config loaded successfully from %s", r.configPath)
	return nil
}

func (r *LocalRepository) save() error {
	data, err := json.MarshalIndent(r.config, "", "  ")
	if err != nil {
		logger.LogError("MARSHAL", r.configPath, err)
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	logger.LogFileWrite(r.configPath)
	if err := os.WriteFile(r.configPath, data, 0600); err != nil {
		logger.LogError("SAVE", r.configPath, err)
		return err
	}

	logger.Log("Config saved successfully to %s", r.configPath)
	return nil
}

func (r *LocalRepository) ListPATs() ([]domain.PAT, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	logger.Log("Listing PATs: found %d", len(r.config.PATs))
	pats := make([]domain.PAT, len(r.config.PATs))
	copy(pats, r.config.PATs)
	return pats, nil
}

func (r *LocalRepository) GetPAT(id string) (*domain.PAT, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, pat := range r.config.PATs {
		if pat.ID == id {
			return &pat, nil
		}
	}

	return nil, fmt.Errorf("PAT not found: %s", id)
}

func (r *LocalRepository) SavePAT(pat domain.PAT) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for i, p := range r.config.PATs {
		if p.ID == pat.ID {
			r.config.PATs[i] = pat
			found = true
			logger.Log("Updating existing PAT: %s (Provider: %s)", pat.Name, pat.Provider)
			break
		}
	}

	if !found {
		r.config.PATs = append(r.config.PATs, pat)
		logger.Log("Adding new PAT: %s (Provider: %s)", pat.Name, pat.Provider)
	}

	return r.save()
}

func (r *LocalRepository) DeletePAT(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, pat := range r.config.PATs {
		if pat.ID == id {
			logger.Log("Deleting PAT: %s (Provider: %s)", pat.Name, pat.Provider)
			r.config.PATs = append(r.config.PATs[:i], r.config.PATs[i+1:]...)

			if r.config.ActivePAT == id {
				r.config.ActivePAT = ""
				logger.Log("Cleared active PAT")
			}

			for i, selectedID := range r.config.SelectedPATs {
				if selectedID == id {
					r.config.SelectedPATs = append(r.config.SelectedPATs[:i], r.config.SelectedPATs[i+1:]...)
					logger.Log("Removed PAT from selected list")

					if r.config.PrimaryPAT == id {
						if len(r.config.SelectedPATs) > 0 {
							r.config.PrimaryPAT = r.config.SelectedPATs[0]
							logger.Log("Changed primary PAT to: %s", r.config.PrimaryPAT)
						} else {
							r.config.PrimaryPAT = ""
							logger.Log("Cleared primary PAT")
						}
					}
					break
				}
			}

			return r.save()
		}
	}

	logger.LogError("DELETE_PAT", id, fmt.Errorf("PAT not found"))
	return fmt.Errorf("PAT not found: %s", id)
}

func (r *LocalRepository) SetActivePAT(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, pat := range r.config.PATs {
		if pat.ID == id {
			logger.Log("Setting active PAT: %s (Provider: %s)", pat.Name, pat.Provider)
			for i := range r.config.PATs {
				r.config.PATs[i].IsActive = r.config.PATs[i].ID == id
			}
			r.config.ActivePAT = id
			return r.save()
		}
	}

	logger.LogError("SET_ACTIVE_PAT", id, fmt.Errorf("PAT not found"))
	return fmt.Errorf("PAT not found: %s", id)
}

func (r *LocalRepository) GetActivePAT() (*domain.PAT, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.config.ActivePAT == "" {
		logger.LogError("GET_ACTIVE_PAT", "", fmt.Errorf("no active PAT set"))
		return nil, fmt.Errorf("no active PAT set")
	}

	for _, pat := range r.config.PATs {
		if pat.ID == r.config.ActivePAT {
			logger.Log("Retrieved active PAT: %s (Provider: %s)", pat.Name, pat.Provider)
			return &pat, nil
		}
	}

	logger.LogError("GET_ACTIVE_PAT", r.config.ActivePAT, fmt.Errorf("active PAT not found"))
	return nil, fmt.Errorf("active PAT not found: %s", r.config.ActivePAT)
}

func (r *LocalRepository) GetSelectedPATs() ([]domain.PAT, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.config.SelectedPATs) == 0 {
		logger.Log("No PATs selected")
		return []domain.PAT{}, nil
	}

	selected := make([]domain.PAT, 0, len(r.config.SelectedPATs))
	for _, id := range r.config.SelectedPATs {
		for _, pat := range r.config.PATs {
			if pat.ID == id {
				patCopy := pat
				patCopy.IsSelected = true
				patCopy.IsPrimary = (id == r.config.PrimaryPAT)
				selected = append(selected, patCopy)
				break
			}
		}
	}

	logger.Log("Retrieved %d selected PATs (primary: %s)", len(selected), r.config.PrimaryPAT)
	return selected, nil
}

func (r *LocalRepository) SetSelectedPATs(ids []string, primaryID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(ids) == 0 {
		return fmt.Errorf("must select at least one PAT")
	}

	for _, id := range ids {
		found := false
		for _, pat := range r.config.PATs {
			if pat.ID == id {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("PAT not found: %s", id)
		}
	}

	primaryFound := false
	for _, id := range ids {
		if id == primaryID {
			primaryFound = true
			break
		}
	}
	if !primaryFound {
		return fmt.Errorf("primary PAT must be in selected set")
	}

	r.config.SelectedPATs = ids
	r.config.PrimaryPAT = primaryID

	for i := range r.config.PATs {
		r.config.PATs[i].IsSelected = false
		r.config.PATs[i].IsPrimary = false
		for _, id := range ids {
			if r.config.PATs[i].ID == id {
				r.config.PATs[i].IsSelected = true
				if id == primaryID {
					r.config.PATs[i].IsPrimary = true
				}
				break
			}
		}
	}

	logger.Log("Set %d selected PATs (primary: %s)", len(ids), primaryID)
	return r.save()
}

func (r *LocalRepository) GetPrimaryPAT() (*domain.PAT, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.config.PrimaryPAT == "" {
		logger.LogError("GET_PRIMARY_PAT", "", fmt.Errorf("no primary PAT set"))
		return nil, fmt.Errorf("no primary PAT set")
	}

	for _, pat := range r.config.PATs {
		if pat.ID == r.config.PrimaryPAT {
			patCopy := pat
			patCopy.IsPrimary = true
			logger.Log("Retrieved primary PAT: %s (Provider: %s)", pat.Name, pat.Provider)
			return &patCopy, nil
		}
	}

	logger.LogError("GET_PRIMARY_PAT", r.config.PrimaryPAT, fmt.Errorf("primary PAT not found"))
	return nil, fmt.Errorf("primary PAT not found: %s", r.config.PrimaryPAT)
}

func (r *LocalRepository) TogglePATSelection(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for _, pat := range r.config.PATs {
		if pat.ID == id {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("PAT not found: %s", id)
	}

	isSelected := false
	selectedIndex := -1
	for i, selectedID := range r.config.SelectedPATs {
		if selectedID == id {
			isSelected = true
			selectedIndex = i
			break
		}
	}

	if isSelected {
		if len(r.config.SelectedPATs) == 1 {
			return fmt.Errorf("cannot deselect the last PAT")
		}

		r.config.SelectedPATs = append(r.config.SelectedPATs[:selectedIndex], r.config.SelectedPATs[selectedIndex+1:]...)

		if r.config.PrimaryPAT == id {
			r.config.PrimaryPAT = r.config.SelectedPATs[0]
			logger.Log("Deselected PAT and changed primary to: %s", r.config.PrimaryPAT)
		} else {
			logger.Log("Deselected PAT: %s", id)
		}
	} else {
		r.config.SelectedPATs = append(r.config.SelectedPATs, id)
		if len(r.config.SelectedPATs) == 1 {
			r.config.PrimaryPAT = id
			logger.Log("Selected first PAT and set as primary: %s", id)
		} else {
			logger.Log("Selected PAT: %s", id)
		}
	}

	for i := range r.config.PATs {
		r.config.PATs[i].IsSelected = false
		r.config.PATs[i].IsPrimary = false
		for _, selectedID := range r.config.SelectedPATs {
			if r.config.PATs[i].ID == selectedID {
				r.config.PATs[i].IsSelected = true
				if selectedID == r.config.PrimaryPAT {
					r.config.PATs[i].IsPrimary = true
				}
				break
			}
		}
	}

	return r.save()
}

func (r *LocalRepository) SetPrimaryPAT(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	isSelected := false
	for _, selectedID := range r.config.SelectedPATs {
		if selectedID == id {
			isSelected = true
			break
		}
	}
	if !isSelected {
		return fmt.Errorf("cannot set non-selected PAT as primary: %s", id)
	}

	r.config.PrimaryPAT = id

	for i := range r.config.PATs {
		r.config.PATs[i].IsPrimary = (r.config.PATs[i].ID == id)
	}

	logger.Log("Set primary PAT: %s", id)
	return r.save()
}
