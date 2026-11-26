package listing

import (
	"context"
	"fmt"
	"io"
	"slices"
	"text/tabwriter"
	"time"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	noPRsText          = "No open pull requests."
	sectionTitleFormat = "%s (%d)\n"
	prRowFormat        = "  #%d\t%s\t%s\t%s\t%s\t%s\n"
	draftPrefix        = "[draft] "
	tabMinWidth        = 0
	tabWidth           = 0
	tabPadding         = 2
	tabPadChar         = ' '
)

// prSections are the categories in the order they are printed.
var prSections = []struct {
	category domain.PRCategory
	title    string
}{
	{domain.PRCategoryAuthored, "Created by you"},
	{domain.PRCategoryAssigned, "Assigned to you"},
	{domain.PRCategoryOther, "Other"},
}

var approvalLabels = map[domain.ApprovalStatus]string{
	domain.ApprovalStatusNone:             "",
	domain.ApprovalStatusPending:          "pending",
	domain.ApprovalStatusApproved:         "approved",
	domain.ApprovalStatusChangesRequested: "changes requested",
}

// PrintPRs prints the open pull requests of the selected PATs to out, grouped like the PR list
// and most recently updated first. Old pull requests of others are left out as the settings say.
// What the other PATs listed is printed even when one fails.
func PrintPRs(ctx context.Context, pats []domain.PAT, newProvider domain.ProviderFactory, settings domain.Settings, out io.Writer) error {
	listings, err := listSelectedAccounts(ctx, pats, newProvider, func(ctx context.Context, pat domain.PAT, provider domain.Provider) ([]domain.PullRequest, error) {
		return provider.ListPullRequests(ctx, pat.Username)
	})
	if err != nil {
		return err
	}

	var prs []domain.PullRequest
	for _, listing := range listings {
		prs = append(prs, listing.items...)
	}
	if settings.HideOldOtherPRs {
		prs = domain.WithoutOldOtherPRs(prs, settings.HideOldOtherPRsDays, time.Now())
	}
	writePRs(out, prs)
	return accountErrors(listings, nil)
}

func writePRs(out io.Writer, prs []domain.PullRequest) {
	if len(prs) == 0 {
		fmt.Fprintln(out, noPRsText)
		return
	}
	slices.SortStableFunc(prs, func(a, b domain.PullRequest) int { return b.UpdatedAt.Compare(a.UpdatedAt) })

	printed := 0
	for _, section := range prSections {
		inSection := slices.DeleteFunc(slices.Clone(prs), func(pr domain.PullRequest) bool { return pr.Category != section.category })
		if len(inSection) == 0 {
			continue
		}
		if printed > 0 {
			fmt.Fprintln(out)
		}
		printed++
		fmt.Fprintf(out, sectionTitleFormat, section.title, len(inSection))
		table := tabwriter.NewWriter(out, tabMinWidth, tabWidth, tabPadding, tabPadChar, 0)
		for _, pr := range inSection {
			fmt.Fprintf(table, prRowFormat, pr.Number, pr.Repository.FullName, prTitle(pr), pr.Author.Username, approvalLabels[pr.ApprovalStatus], pr.URL)
		}
		table.Flush()
	}
}

func prTitle(pr domain.PullRequest) string {
	if pr.IsDraft {
		return draftPrefix + pr.Title
	}
	return pr.Title
}
