package storage

import (
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type Config struct {
	PATs               []domain.PAT            `json:"pats"`
	ActivePAT          string                  `json:"active_pat"`
	SelectedPATs       []string                `json:"selected_pats"`
	PrimaryPAT         string                  `json:"primary_pat"`
	RepositoryLinks    []domain.RepositoryLink `json:"repository_links,omitempty"`
	PipelineFavorites  []string                `json:"pipeline_favorites,omitempty"`
	PipelineLastViewed map[string]time.Time    `json:"pipeline_last_viewed,omitempty"`
	Settings           domain.Settings         `json:"settings"`
	Session            domain.Session          `json:"session"`
}
