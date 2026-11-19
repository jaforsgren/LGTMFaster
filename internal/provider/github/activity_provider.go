package github

import (
	"context"
	"strings"

	"github.com/google/go-github/v57/github"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/provider/common"
)

const (
	timelineEventCommented       = "commented"
	timelineEventReviewed        = "reviewed"
	timelineEventCommitted       = "committed"
	timelineEventHeadForcePushed = "head_ref_force_pushed"
	commentedSummaryPrefix       = "commented: "
	committedSummaryPrefix       = "pushed: "
	forcePushedSummary           = "force-pushed"
	reviewedSummary              = "reviewed"
)

// reviewSummaries are keyed by the lower-case review states the timeline reports.
var reviewSummaries = map[string]string{
	"approved":          "approved",
	"changes_requested": "requested changes",
	"commented":         reviewedSummary,
	"dismissed":         "had a review dismissed",
}

// ListActivity reads the pull request's timeline, oldest first; the other events (labels,
// assignments, references) are left out.
func (p *Provider) ListActivity(ctx context.Context, identifier domain.PRIdentifier) ([]domain.Activity, error) {
	owner, repo, err := common.ParseGitHubRepository(identifier.Repository)
	if err != nil {
		return nil, err
	}
	events, err := p.client.ListTimeline(ctx, owner, repo, identifier.Number)
	if err != nil {
		return nil, err
	}

	var activity []domain.Activity
	for _, event := range events {
		if item, ok := timelineActivity(event, p.username); ok {
			activity = append(activity, item)
		}
	}
	return activity, nil
}

func timelineActivity(event *github.Timeline, username string) (domain.Activity, bool) {
	switch event.GetEvent() {
	case timelineEventCommented:
		return userActivity(domain.ActivityKindComment, event.GetUser(), event.GetCreatedAt(),
			commentedSummaryPrefix+common.CommitSummary(event.GetBody()), username), true
	case timelineEventReviewed:
		summary, known := reviewSummaries[strings.ToLower(event.GetState())]
		if !known {
			summary = reviewedSummary
		}
		return userActivity(domain.ActivityKindReview, event.GetUser(), event.GetSubmittedAt(), summary, username), true
	case timelineEventHeadForcePushed:
		return userActivity(domain.ActivityKindPush, event.GetActor(), event.GetCreatedAt(), forcePushedSummary, username), true
	case timelineEventCommitted:
		// Commit events carry the git author, not a GitHub account, so ByMe can only compare names.
		author := event.GetAuthor()
		return domain.Activity{
			Kind:    domain.ActivityKindPush,
			Author:  domain.User{Username: author.GetName(), Email: author.GetEmail()},
			At:      author.GetDate().Time,
			Summary: committedSummaryPrefix + common.CommitSummary(event.GetMessage()),
			ByMe:    strings.EqualFold(author.GetName(), username) || strings.EqualFold(author.GetLogin(), username),
		}, true
	}
	return domain.Activity{}, false
}

func userActivity(kind domain.ActivityKind, user *github.User, at github.Timestamp, summary, username string) domain.Activity {
	return domain.Activity{
		Kind:    kind,
		Author:  domain.User{ID: user.GetNodeID(), Username: user.GetLogin(), Avatar: user.GetAvatarURL()},
		At:      at.Time,
		Summary: summary,
		ByMe:    strings.EqualFold(user.GetLogin(), username),
	}
}
