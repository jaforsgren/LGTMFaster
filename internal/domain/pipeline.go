package domain

import (
	"fmt"
	"strings"
	"time"
)

// PipelineFolderSeparator separates the folders of a pipeline's path, as Azure DevOps writes them.
const PipelineFolderSeparator = `\`

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

var pipelineStatusLabels = map[PipelineStatus]string{
	PipelineStatusQueued:             "queued",
	PipelineStatusRunning:            "running",
	PipelineStatusWaitingForApproval: "waiting for approval",
	PipelineStatusSucceeded:          "succeeded",
	PipelineStatusPartiallySucceeded: "partially succeeded",
	PipelineStatusFailed:             "failed",
	PipelineStatusCanceled:           "canceled",
	PipelineStatusSkipped:            "skipped",
}

// Label is the status as people read it; anything unrecognised reads as unknown.
func (s PipelineStatus) Label() string {
	if label, ok := pipelineStatusLabels[s]; ok {
		return label
	}
	return string(PipelineStatusUnknown)
}

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

type PipelineSortMode string

const (
	PipelineSortFavorites  PipelineSortMode = "favorites"
	PipelineSortProject    PipelineSortMode = "project"
	PipelineSortRepository PipelineSortMode = "repository"
	PipelineSortPipeline   PipelineSortMode = "pipeline"
	PipelineSortLastRun    PipelineSortMode = "last_run"
	PipelineSortStatus     PipelineSortMode = "status"
)

// PipelineSortModes is ordered by the number key that selects each mode.
var PipelineSortModes = []PipelineSortMode{
	PipelineSortFavorites,
	PipelineSortProject,
	PipelineSortRepository,
	PipelineSortPipeline,
	PipelineSortLastRun,
	PipelineSortStatus,
}

// OrDefault maps unset or unknown modes (e.g. hand-edited settings) to the favorites mode.
func (m PipelineSortMode) OrDefault() PipelineSortMode {
	for _, mode := range PipelineSortModes {
		if m == mode {
			return m
		}
	}
	return PipelineSortFavorites
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

// DisplayName prefixes the pipeline name with its folder; the root folder is omitted.
func (p Pipeline) DisplayName() string {
	folder := strings.Trim(p.Folder, PipelineFolderSeparator)
	if folder == "" {
		return p.Name
	}
	return folder + PipelineFolderSeparator + p.Name
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
	// CommitMessage is the first line of the message of the commit the run builds.
	CommitMessage string
	// CurrentStage is the stage an unfinished run is at; empty when it is finished or unknown.
	CurrentStage string
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

func (r PipelineRun) PipelineIdentifier() PipelineIdentifier {
	return PipelineIdentifier{Project: r.Project, PipelineID: r.PipelineID}
}

func (r PipelineRun) FavoriteKey() string {
	return PipelineRunFavoriteKey(r.PATID, r.Project, r.ID)
}

type PipelineStage struct {
	ID   string
	Name string
	// RefName is the stage's identifier in the pipeline definition, which retrying it needs.
	RefName           string
	Order             int
	Status            PipelineStatus
	PendingApprovalID string
	StartedAt         time.Time
	FinishedAt        time.Time
}

func (s PipelineStage) AwaitsApproval() bool {
	return s.PendingApprovalID != ""
}

// ActiveStage is the stage a run is at: the first one waiting for an approval, else the first one
// running. Stages are in run order.
func ActiveStage(stages []PipelineStage) (PipelineStage, bool) {
	for _, stage := range stages {
		if stage.AwaitsApproval() {
			return stage, true
		}
	}
	for _, stage := range stages {
		if stage.Status == PipelineStatusRunning {
			return stage, true
		}
	}
	return PipelineStage{}, false
}

type StageApproval struct {
	Project    string
	ApprovalID string
	Decision   ApprovalDecision
	Comment    string
}
