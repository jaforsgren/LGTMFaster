package domain

import "time"

type ProviderType string

const (
	ProviderGitHub      ProviderType = "github"
	ProviderAzureDevOps ProviderType = "azuredevops"
)

type ReviewAction string

const (
	ReviewActionApprove        ReviewAction = "approve"
	ReviewActionRequestChanges ReviewAction = "request_changes"
	ReviewActionComment        ReviewAction = "comment"
)

type PRStatus string

const (
	PRStatusOpen   PRStatus = "open"
	PRStatusClosed PRStatus = "closed"
	PRStatusMerged PRStatus = "merged"
)

type PRCategory string

const (
	PRCategoryAuthored PRCategory = "authored"
	PRCategoryAssigned PRCategory = "assigned"
	PRCategoryOther    PRCategory = "other"
)

type User struct {
	ID       string
	Username string
	Email    string
	Avatar   string
}

type Repo struct {
	ID       string
	Name     string
	FullName string
	Owner    string
	URL      string
}

type PullRequest struct {
	ID           string
	Number       int
	Title        string
	Description  string
	Author       User
	Repository   Repo
	SourceBranch string
	TargetBranch string
	Status       PRStatus
	Category     PRCategory
	CreatedAt    time.Time
	UpdatedAt    time.Time
	URL          string
	IsDraft      bool
	Mergeable    bool
}

type Comment struct {
	ID        string
	Author    User
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
	FilePath  string
	Line      int
	Side      string
}

type DiffLine struct {
	Type    string
	Content string
	OldLine int
	NewLine int
}

type DiffHunk struct {
	Header string
	Lines  []DiffLine
}

type FileDiff struct {
	OldPath   string
	NewPath   string
	OldMode   string
	NewMode   string
	IsNew     bool
	IsDeleted bool
	IsRenamed bool
	Hunks     []DiffHunk
	Comments  []Comment
}

type Diff struct {
	Files []FileDiff
}

type Review struct {
	PRIdentifier string
	Action       ReviewAction
	Body         string
	Comments     []Comment
}

type PRIdentifier struct {
	Provider   ProviderType
	Repository string
	Number     int
}

type PRGroup struct {
	PATName   string
	PATID     string
	Provider  ProviderType
	Username  string
	IsPrimary bool
	PRs       []PullRequest
}
