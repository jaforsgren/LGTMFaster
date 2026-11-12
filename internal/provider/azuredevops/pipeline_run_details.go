package azuredevops

import (
	"context"
	"strings"
	"sync"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

const (
	triggerInfoCIMessage   = "ci.message"
	triggerInfoPRSourceSHA = "pr.sourceSha"
	runDetailConcurrency   = 8
)

// convertPipelineRuns converts builds and fills in what the build itself lacks: the commit message
// when the trigger info has none (manual and pull request runs), and for unfinished runs the stage
// they are at. Commits never change, so messages are cached for good.
func (p *Provider) convertPipelineRuns(ctx context.Context, builds []build.Build, projectName string) []domain.PipelineRun {
	runs := make([]domain.PipelineRun, len(builds))
	slots := make(chan struct{}, runDetailConcurrency)
	var wg sync.WaitGroup
	for i := range builds {
		runs[i] = p.convertPipelineRun(&builds[i], projectName)
		needsCommit, needsStage := runs[i].CommitMessage == "", !runs[i].Status.IsFinished()
		if !needsCommit && !needsStage {
			continue
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			if needsCommit {
				runs[i].CommitMessage = p.lookupCommitMessage(ctx, &builds[i], projectName)
			}
			if needsStage {
				p.applyActiveStage(ctx, &runs[i])
			}
		}(i)
	}
	wg.Wait()
	return runs
}

// applyActiveStage names the stage the run is at and marks the run as waiting when that stage
// waits for an approval.
func (p *Provider) applyActiveStage(ctx context.Context, run *domain.PipelineRun) {
	stages, err := p.GetPipelineRunStages(ctx, run.Identifier())
	if err != nil {
		return
	}
	stage, found := domain.ActiveStage(stages)
	if !found {
		return
	}
	run.CurrentStage = stage.Name
	if stage.AwaitsApproval() {
		run.Status = domain.PipelineStatusWaitingForApproval
	}
}

func triggerCommitMessage(b *build.Build) string {
	return commitSummary(triggerInfo(b, triggerInfoCIMessage))
}

func triggerInfo(b *build.Build, key string) string {
	if b.TriggerInfo == nil {
		return ""
	}
	return (*b.TriggerInfo)[key]
}

// lookupCommitMessage reads the commit the run builds. A pull request run builds a merge commit, so
// the pull request's source commit is preferred. Only Azure Repos commits can be read.
func (p *Provider) lookupCommitMessage(ctx context.Context, b *build.Build, projectName string) string {
	if b.Repository == nil || common.GetString(b.Repository.Type) != azureReposGitRepositoryType || b.Repository.Id == nil {
		return ""
	}
	commitID := triggerInfo(b, triggerInfoPRSourceSHA)
	if commitID == "" {
		commitID = common.GetString(b.SourceVersion)
	}
	if commitID == "" {
		return ""
	}

	cacheKey := *b.Repository.Id + "@" + commitID
	if message, cached := p.commitMessages.Load(cacheKey); cached {
		return message.(string)
	}
	comment, err := p.client.GetCommitComment(ctx, projectName, *b.Repository.Id, commitID)
	if err != nil {
		logger.LogError("AZDO_GET_RUN_COMMIT", commitID, err)
		return ""
	}
	message := commitSummary(comment)
	p.commitMessages.Store(cacheKey, message)
	return message
}

// commitSummary is the first line of a commit message.
func commitSummary(message string) string {
	summary, _, _ := strings.Cut(strings.TrimSpace(message), "\n")
	return strings.TrimSpace(summary)
}
