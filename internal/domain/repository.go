package domain

type PAT struct {
	ID       string
	Name     string
	Token    string
	Provider ProviderType
	Username string
	IsActive bool
}

type Repository interface {
	ListPATs() ([]PAT, error)

	GetPAT(id string) (*PAT, error)

	SavePAT(pat PAT) error

	DeletePAT(id string) error

	SetActivePAT(id string) error

	GetActivePAT() (*PAT, error)
}
