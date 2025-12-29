package storage

import "github.com/johanforsgren/lgtmfaster/internal/domain"

type Config struct {
	PATs      []domain.PAT `json:"pats"`
	ActivePAT string       `json:"active_pat"`
}
