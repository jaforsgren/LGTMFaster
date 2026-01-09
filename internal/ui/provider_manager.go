package ui

import (
	"fmt"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/azuredevops"
	"github.com/johanforsgren/lgtmfaster/internal/provider/github"
)

// ProviderManager handles PAT-to-provider mapping and provider resolution
type ProviderManager struct {
	providers       map[string]domain.Provider // PAT ID -> Provider
	primaryProvider domain.Provider
	primaryPATID    string
	singleProvider  domain.Provider // For backwards compatibility with single PAT mode
}

// NewProviderManager creates a new provider manager
func NewProviderManager() *ProviderManager {
	return &ProviderManager{
		providers: make(map[string]domain.Provider),
	}
}

// InitializeProviders creates providers for all selected PATs and sets up the primary provider
func (pm *ProviderManager) InitializeProviders(pats []domain.PAT) error {
	// Reset state
	pm.providers = make(map[string]domain.Provider)
	pm.primaryProvider = nil
	pm.primaryPATID = ""
	pm.singleProvider = nil

	// Create provider for first active PAT (backwards compatibility)
	for _, pat := range pats {
		if pat.IsActive && pm.singleProvider == nil {
			provider, err := pm.createProvider(pat)
			if err != nil {
				return fmt.Errorf("failed to create provider for active PAT: %w", err)
			}
			pm.singleProvider = provider
			break
		}
	}

	// Create providers for all selected PATs
	for _, pat := range pats {
		if pat.IsSelected {
			provider, err := pm.createProvider(pat)
			if err != nil {
				logger.LogError("CREATE_PROVIDER", pat.Name, err)
				continue
			}
			pm.providers[pat.ID] = provider

			if pat.IsPrimary {
				pm.primaryProvider = provider
				pm.primaryPATID = pat.ID
			}
		}
	}

	return nil
}

// GetProviderForPR returns the appropriate provider for a given pull request
func (pm *ProviderManager) GetProviderForPR(pr domain.PullRequest) domain.Provider {
	// If we have multiple providers, use the one that matches the PR's PATID
	if len(pm.providers) > 0 && pr.PATID != "" {
		if provider, ok := pm.providers[pr.PATID]; ok {
			return provider
		}
	}

	// Fallback to primary provider if available
	if pm.primaryProvider != nil {
		return pm.primaryProvider
	}

	// Fallback to single provider
	return pm.singleProvider
}

// GetProviderByPATID returns the provider for a specific PAT ID
func (pm *ProviderManager) GetProviderByPATID(patID string) domain.Provider {
	return pm.providers[patID]
}

// GetPrimaryProvider returns the primary provider and its PAT ID
func (pm *ProviderManager) GetPrimaryProvider() (domain.Provider, string) {
	return pm.primaryProvider, pm.primaryPATID
}

// GetSingleProvider returns the single provider (for backwards compatibility)
func (pm *ProviderManager) GetSingleProvider() domain.Provider {
	return pm.singleProvider
}

// GetAllProviders returns the map of all providers
func (pm *ProviderManager) GetAllProviders() map[string]domain.Provider {
	return pm.providers
}

// ProviderCount returns the number of initialized providers
func (pm *ProviderManager) ProviderCount() int {
	return len(pm.providers)
}

// HasProviders returns true if any providers are initialized
func (pm *ProviderManager) HasProviders() bool {
	return len(pm.providers) > 0 || pm.singleProvider != nil
}

// createProvider creates a provider instance based on PAT type
func (pm *ProviderManager) createProvider(pat domain.PAT) (domain.Provider, error) {
	switch pat.Provider {
	case domain.ProviderGitHub:
		return github.NewProvider(pat.Token, pat.Username), nil
	case domain.ProviderAzureDevOps:
		provider, err := azuredevops.NewProvider(pat.Token, pat.Organization, pat.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure DevOps provider: %w", err)
		}
		return provider, nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", pat.Provider)
	}
}
