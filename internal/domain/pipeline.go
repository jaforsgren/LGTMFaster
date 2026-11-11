package domain

import (
	"fmt"
	"time"
)

type PipelineStatus string

const (
	PipelineStatusQueued             PipelineStatus = "queued"
	PipelineStatusRunning            PipelineStatus = "running"
	PipelineStatusWaitingForApproval PipelineStatus = "waiting_for_approval"
	PipelineStatusSucceeded          PipelineStatus = "succeeded"
	PipelineStatusPartiallySucceeded PipelineStatus = "partially_succeeded"
	PipelineStatusFailed             PipelineStatus = "failed"
	PipelineStatusCanceled           PipelineStatus = "canceled"
	PipelineStatusSkipped            PipelineStatus = "skipped"
	PipelineStatusUnknown            PipelineStatus = "unknown"
)

func (s PipelineStatus) IsFinished() bool {
	switch s {
	case PipelineStatusSucceeded, PipelineStatusPartiallySucceeded, PipelineStatusFailed,
		PipelineStatusCanceled, PipelineStatusSkipped:
		return true
	default:
		return false
	}
}

// Favorite keys are scoped by PAT so identically named projects in different organizations stay distinct.
const (
	favoriteKindProject    = "project"
	favoriteKindRepository = "repo"
	favoriteKindPipeline   = "pipeline"
	favoriteKindRun        = "run"
)

func ProjectFavoriteKey(patID, project string) string {
	return fmt.Sprintf("%s:%s:%s", favoriteKindProject, patID, project)
}

func RepositoryFavoriteKey(patID, project, repository string) string {
	return fmt.Sprintf("%s:%s:%s/%s", favoriteKindRepository, patID, project, repository)
}

func PipelineFavoriteKey(patID, project string, pipelineID int) string {
	return fmt.Sprintf("%s:%s:%s:%d", favoriteKindPipeline, patID, project, pipelineID)
}

func PipelineRunFavoriteKey(patID, project string, runID int) string {
	return fmt.Sprintf("%s:%s:%s:%d", favoriteKindRun, patID, project, runID)
}

type ApprovalDecision string

const (
	ApprovalDecisionApprove ApprovalDecision = "approve"
	ApprovalDecisionReject  ApprovalDecision = "reject"
)

type PipelineIdentifier struct {
	Project    string
	PipelineID int
}

type PipelineRunIdentifier struct {
	Project string
	RunID   int
}

type Pipeline struct {
	ID           int
	Name         string
	Folder       string
	Project      string
	Repository   string
	URL          string
	LatestRun    *PipelineRun
	ProviderType ProviderType
	PATID        string
}

func (p Pipeline) Identifier() PipelineIdentifier {
	return PipelineIdentifier{Project: p.Project, PipelineID: p.ID}
}

func (p Pipeline) FavoriteKey() string {
	return PipelineFavoriteKey(p.PATID, p.Project, p.ID)
}

type PipelineRun struct {
	ID           int
	Number       string
	PipelineID   int
	PipelineName string
	Project      string
	Status       PipelineStatus
	SourceBranch string
	RequestedBy  User
	QueuedAt     time.Time
	StartedAt    time.Time
	FinishedAt   time.Time
	URL          string
	ProviderType ProviderType
	PATID        string
}

func (r PipelineRun) Identifier() PipelineRunIdentifier {
	return PipelineRunIdentifier{Project: r.Project, RunID: r.ID}
}

func (r PipelineRun) FavoriteKey() string {
	return PipelineRunFavoriteKey(r.PATID, r.Project, r.ID)
}

type PipelineStage struct {
	ID                string
	Name              string
	Order             int
	Status            PipelineStatus
	PendingApprovalID string
}

func (s PipelineStage) AwaitsApproval() bool {
	return s.PendingApprovalID != ""
}

type StageApproval struct {
	Project    string
	ApprovalID string
	Decision   ApprovalDecision
	Comment    string
}
