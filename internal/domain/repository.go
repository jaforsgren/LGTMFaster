package domain

import "time"

type PAT struct {
	ID           string
	Name         string
	Token        string
	Provider     ProviderType
	Username     string
	Organization string
	IsActive     bool
	IsSelected   bool
	IsPrimary    bool
}

type Repository interface {
	ListPATs() ([]PAT, error)

	GetPAT(id string) (*PAT, error)

	SavePAT(pat PAT) error

	DeletePAT(id string) error

	SetActivePAT(id string) error

	GetActivePAT() (*PAT, error)

	GetSelectedPATs() ([]PAT, error)

	SetSelectedPATs(ids []string, primaryID string) error

	GetPrimaryPAT() (*PAT, error)

	TogglePATSelection(id string) error

	SetPrimaryPAT(id string) error

	ListRepositoryLinks() ([]RepositoryLink, error)

	GetRepositoryLink(id string) (*RepositoryLink, error)

	GetRepositoryLinkByRepoFullName(repoFullName string) (*RepositoryLink, error)

	SaveRepositoryLink(link RepositoryLink) error

	DeleteRepositoryLink(id string) error

	ListPipelineFavorites() ([]string, error)

	// TogglePipelineFavorite returns whether the key is a favorite after the toggle.
	TogglePipelineFavorite(key string) (bool, error)

	ListPipelineLastViewed() (map[string]time.Time, error)

	MarkPipelineViewed(key string, viewedAt time.Time) error

	GetSettings() (Settings, error)

	SaveSettings(settings Settings) error

	GetSession() (Session, error)

	SaveSession(session Session) error

	ConfigPath() string

	Reload() error
}
