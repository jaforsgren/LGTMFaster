package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
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

	data, err := os.ReadFile(r.configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, r.config)
}

func (r *LocalRepository) save() error {
	data, err := json.MarshalIndent(r.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(r.configPath, data, 0600)
}

func (r *LocalRepository) ListPATs() ([]domain.PAT, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

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
			break
		}
	}

	if !found {
		r.config.PATs = append(r.config.PATs, pat)
	}

	return r.save()
}

func (r *LocalRepository) DeletePAT(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, pat := range r.config.PATs {
		if pat.ID == id {
			r.config.PATs = append(r.config.PATs[:i], r.config.PATs[i+1:]...)
			if r.config.ActivePAT == id {
				r.config.ActivePAT = ""
			}
			return r.save()
		}
	}

	return fmt.Errorf("PAT not found: %s", id)
}

func (r *LocalRepository) SetActivePAT(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, pat := range r.config.PATs {
		if pat.ID == id {
			for i := range r.config.PATs {
				r.config.PATs[i].IsActive = r.config.PATs[i].ID == id
			}
			r.config.ActivePAT = id
			return r.save()
		}
	}

	return fmt.Errorf("PAT not found: %s", id)
}

func (r *LocalRepository) GetActivePAT() (*domain.PAT, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.config.ActivePAT == "" {
		return nil, fmt.Errorf("no active PAT set")
	}

	for _, pat := range r.config.PATs {
		if pat.ID == r.config.ActivePAT {
			return &pat, nil
		}
	}

	return nil, fmt.Errorf("active PAT not found: %s", r.config.ActivePAT)
}
