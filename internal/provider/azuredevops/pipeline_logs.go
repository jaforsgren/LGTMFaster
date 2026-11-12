package azuredevops

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

const (
	timelineRecordTypeTask = "Task"
	timelineRecordTypeJob  = "Job"
	byteOrderMark          = "\uFEFF"
)

// logLineTimestamp is the UTC timestamp Azure DevOps prefixes every log line with.
var logLineTimestamp = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z ?`)

func (p *Provider) ListPipelineRunLogs(ctx context.Context, identifier domain.PipelineRunIdentifier) ([]domain.PipelineLog, error) {
	timeline, err := p.client.GetBuildTimeline(ctx, identifier.Project, identifier.RunID)
	if err != nil {
		logger.LogError("AZDO_LIST_PIPELINE_LOGS", fmt.Sprintf("%s/%d", identifier.Project, identifier.RunID), err)
		return nil, err
	}
	if timeline == nil || timeline.Records == nil {
		return []domain.PipelineLog{}, nil
	}
	return convertTimelineToLogs(*timeline.Records), nil
}

// GetPipelineLogLines only returns what the agent has uploaded so far, which for a running step
// can lag behind its live console output.
func (p *Provider) GetPipelineLogLines(ctx context.Context, identifier domain.PipelineRunIdentifier, logID int, offset int) ([]string, error) {
	lines, err := p.client.GetBuildLogLines(ctx, identifier.Project, identifier.RunID, logID, offset)
	if err != nil {
		logger.LogError("AZDO_GET_PIPELINE_LOG", fmt.Sprintf("%s/%d/%d", identifier.Project, identifier.RunID, logID), err)
		return nil, err
	}
	for i, line := range lines {
		lines[i] = logLineTimestamp.ReplaceAllString(strings.TrimPrefix(line, byteOrderMark), "")
	}
	return lines, nil
}

// convertTimelineToLogs lists the tasks that have a log. Records only carry their order among
// their siblings, so tasks are sorted by the orders of their ancestors first.
func convertTimelineToLogs(records []build.TimelineRecord) []domain.PipelineLog {
	recordsByID := indexTimelineRecords(records)

	type orderedLog struct {
		log   domain.PipelineLog
		order []int
	}
	var logs []orderedLog
	for _, record := range records {
		if common.GetString(record.Type) != timelineRecordTypeTask || record.Log == nil || record.Log.Id == nil {
			continue
		}
		stage, _ := findAncestor(record, recordsByID, timelineRecordTypeStage)
		job, _ := findAncestor(record, recordsByID, timelineRecordTypeJob)
		logs = append(logs, orderedLog{
			log: domain.PipelineLog{
				ID:         *record.Log.Id,
				Name:       common.GetString(record.Name),
				Stage:      common.GetString(stage.Name),
				Job:        common.GetString(job.Name),
				Status:     mapTimelineRecordStatus(record.State, record.Result),
				StartedAt:  timeOrZero(record.StartTime),
				FinishedAt: timeOrZero(record.FinishTime),
			},
			order: recordOrderPath(record, recordsByID),
		})
	}

	slices.SortStableFunc(logs, func(a, b orderedLog) int {
		return slices.Compare(a.order, b.order)
	})
	converted := make([]domain.PipelineLog, len(logs))
	for i, log := range logs {
		converted[i] = log.log
	}
	return converted
}

// recordOrderPath is the order of the record and each of its ancestors, outermost first.
func recordOrderPath(record build.TimelineRecord, recordsByID map[uuid.UUID]build.TimelineRecord) []int {
	path := []int{common.GetInt(record.Order)}
	for _, ancestor := range timelineAncestors(record, recordsByID) {
		path = append(path, common.GetInt(ancestor.Order))
	}
	slices.Reverse(path)
	return path
}
