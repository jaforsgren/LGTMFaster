package domain

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
}
