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

type ApprovalStatus string

const (
	ApprovalStatusNone             ApprovalStatus = "none"
	ApprovalStatusApproved         ApprovalStatus = "approved"
	ApprovalStatusChangesRequested ApprovalStatus = "changes_requested"
	ApprovalStatusPending          ApprovalStatus = "pending"
)

type MergeMethod string

const (
	MergeMethodMerge         MergeMethod = "merge"
	MergeMethodSquash        MergeMethod = "squash"
	MergeMethodRebase        MergeMethod = "rebase"
	MergeMethodNoFastForward MergeMethod = "noFastForward"
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
	ID             string
	Number         int
	Title          string
	Description    string
	Author         User
	Repository     Repo
	SourceBranch   string
	TargetBranch   string
	Status         PRStatus
	Category       PRCategory
	ApprovalStatus ApprovalStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	URL            string
	IsDraft        bool
	Mergeable      bool
	// ReviewedByMe is set once the current user voted on or reviewed the pull request.
	ReviewedByMe bool
	ProviderType ProviderType
	PATID        string
}

const (
	DefaultHideOldOtherPRsDays        = 30
	DefaultPipelineAutoRefreshSeconds = 15
	DefaultRecentBranchMinutes        = 60
)

type Settings struct {
	HideOldOtherPRs     bool
	HideOldOtherPRsDays int
	PipelineSortMode    PipelineSortMode
	// PipelineAutoRefreshSeconds is how often the pipeline views refresh while auto-refresh is on.
	PipelineAutoRefreshSeconds int
	// PipelineAutoRefreshEnabled turns the pipeline views' auto-refresh on; it is off by default.
	PipelineAutoRefreshEnabled bool
	// RecentBranchMinutes is how far back :pr looks for pushed branches to open a pull request from.
	RecentBranchMinutes int
}

// WithDefaults fills in the settings that are missing or out of range.
func (s Settings) WithDefaults() Settings {
	if s.HideOldOtherPRsDays <= 0 {
		s.HideOldOtherPRsDays = DefaultHideOldOtherPRsDays
	}
	if s.PipelineAutoRefreshSeconds <= 0 {
		s.PipelineAutoRefreshSeconds = DefaultPipelineAutoRefreshSeconds
	}
	if s.RecentBranchMinutes <= 0 {
		s.RecentBranchMinutes = DefaultRecentBranchMinutes
	}
	return s
}

func (s Settings) RecentBranchWindow() time.Duration {
	return time.Duration(s.WithDefaults().RecentBranchMinutes) * time.Minute
}

func (s Settings) PipelineAutoRefreshInterval() time.Duration {
	return time.Duration(s.WithDefaults().PipelineAutoRefreshSeconds) * time.Second
}

// CommentSide says which version of a file an inline comment's line number refers to.
const (
	CommentSideLeft  = "LEFT"
	CommentSideRight = "RIGHT"
)

// Comment is a pull request comment. FilePath is relative to the repository root without a leading
// slash, like the paths in a Diff, and Line counts in the file version Side names.
type Comment struct {
	ID        string
	Author    User
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
	FilePath  string
	Line      int
	Side      string
	// IsSystem marks comments the provider wrote itself, such as vote changes.
	IsSystem bool
	// ThreadID groups a comment with its replies; ThreadState is that thread's resolution, empty
	// when the provider does not report one.
	ThreadID    string
	ThreadState ThreadState
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

type RepositoryLink struct {
	ID           string
	RemoteURL    string
	RepoFullName string
	LocalPath    string
	LinkedAt     time.Time
}
