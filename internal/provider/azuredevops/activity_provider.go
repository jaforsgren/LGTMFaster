package azuredevops

import (
	"context"
	"sort"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
)

const (
	commentedSummaryPrefix   = "commented: "
	updatedSourceBranchLabel = "updated the source branch"
)

var iterationSummaries = map[git.IterationReason]string{
	git.IterationReasonValues.Push:      "pushed",
	git.IterationReasonValues.ForcePush: "force-pushed",
	git.IterationReasonValues.Rebase:    "rebased",
}

// ListActivity reads the comments and votes from the threads and the pushes from the iterations;
// the other system comments (reviewers, status, ref updates) and the iteration that opened the pull
// request (domain.PullRequest.OpenedActivity) are left out.
func (p *Provider) ListActivity(ctx context.Context, identifier domain.PRIdentifier) ([]domain.Activity, error) {
	projectID, repoID, err := p.resolveProjectAndRepoWithCache(ctx, identifier.Repository)
	if err != nil {
		return nil, err
	}
	threads, err := p.client.GetPullRequestThreads(ctx, projectID, repoID, identifier.Number)
	if err != nil {
		return nil, err
	}
	iterations, err := p.client.GetPullRequestIterations(ctx, projectID, repoID, identifier.Number)
	if err != nil {
		return nil, err
	}

	activity := append(p.threadActivity(threads), p.iterationActivity(iterations)...)
	sort.SliceStable(activity, func(i, j int) bool { return activity[i].At.Before(activity[j].At) })
	return activity, nil
}

func (p *Provider) threadActivity(threads *[]git.GitPullRequestCommentThread) []domain.Activity {
	var activity []domain.Activity
	if threads == nil {
		return activity
	}
	for _, thread := range *threads {
		if thread.Comments == nil {
			continue
		}
		for _, comment := range *thread.Comments {
			if common.GetBool(comment.IsDeleted) || comment.PublishedDate == nil {
				continue
			}
			item, ok := commentActivity(comment)
			if !ok {
				continue
			}
			item.At = comment.PublishedDate.Time
			if comment.Author != nil {
				item.Author = convertIdentity(comment.Author)
			}
			item.ByMe = p.isMe(comment.Author)
			activity = append(activity, item)
		}
	}
	return activity
}

func commentActivity(comment git.Comment) (domain.Activity, bool) {
	content := common.GetString(comment.Content)
	if comment.CommentType == nil || *comment.CommentType != git.CommentTypeValues.System {
		return domain.Activity{Kind: domain.ActivityKindComment, Summary: commentedSummaryPrefix + common.CommitSummary(content)}, true
	}
	_, description, isVote := parseVote(content)
	return domain.Activity{Kind: domain.ActivityKindReview, Summary: description}, isVote
}

func (p *Provider) iterationActivity(iterations *[]git.GitPullRequestIteration) []domain.Activity {
	var activity []domain.Activity
	if iterations == nil {
		return activity
	}
	for _, iteration := range *iterations {
		if iteration.CreatedDate == nil || isOpeningIteration(iteration) {
			continue
		}
		pusher := iteration.Author
		if iteration.Push != nil && iteration.Push.PushedBy != nil {
			pusher = iteration.Push.PushedBy
		}
		item := domain.Activity{
			Kind:    domain.ActivityKindPush,
			At:      iteration.CreatedDate.Time,
			Summary: iterationSummary(iteration.Reason),
			ByMe:    p.isMe(pusher),
		}
		if pusher != nil {
			item.Author = convertIdentity(pusher)
		}
		activity = append(activity, item)
	}
	return activity
}

func isOpeningIteration(iteration git.GitPullRequestIteration) bool {
	return iteration.Reason != nil && *iteration.Reason == git.IterationReasonValues.Create
}

func iterationSummary(reason *git.IterationReason) string {
	if reason != nil {
		if summary, known := iterationSummaries[*reason]; known {
			return summary
		}
	}
	return updatedSourceBranchLabel
}

func (p *Provider) isMe(identity *webapi.IdentityRef) bool {
	if identity == nil {
		return false
	}
	if p.client.userID != "" && common.GetString(identity.Id) == p.client.userID {
		return true
	}
	return p.client.matchesUsername(identity.DisplayName, identity.UniqueName)
}
