package views

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	prListTitle         = "Pull Requests"
	prInfoPaneTitle     = "Info"
	prDescriptionTitle  = "Description"
	noDescriptionText   = "No description provided."
	noPRSelectedText    = "No pull request selected."
	loadingCommentsText = "loading..."
	prDetailLabelWidth  = 11
	hoursPerDay         = 24
)

type prCategoryAppearance struct {
	icon     string
	color    lipgloss.Color
	rowStyle lipgloss.Style
}

var prCategoryAppearances = map[domain.PRCategory]prCategoryAppearance{
	domain.PRCategoryAuthored: {"✎", lipgloss.Color("#86EFAC"), lipgloss.NewStyle().Foreground(lipgloss.Color("#86EFAC"))},
	domain.PRCategoryAssigned: {"→", lipgloss.Color("#F59E0B"), listRowStyle},
	domain.PRCategoryOther:    {"○", lipgloss.Color("#6B7280"), lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))},
}

var prCategoryOrder = map[domain.PRCategory]int{
	domain.PRCategoryAuthored: 0,
	domain.PRCategoryAssigned: 1,
	domain.PRCategoryOther:    2,
}

var prApprovalBadges = map[domain.ApprovalStatus]string{
	domain.ApprovalStatusApproved:         "✓",
	domain.ApprovalStatusChangesRequested: "✗",
	domain.ApprovalStatusPending:          "◯",
}

var prApprovalLabels = map[domain.ApprovalStatus]string{
	domain.ApprovalStatusApproved:         "approved",
	domain.ApprovalStatusChangesRequested: "changes requested",
	domain.ApprovalStatusPending:          "pending",
}

func prCategoryAppearanceFor(category domain.PRCategory) prCategoryAppearance {
	if appearance, ok := prCategoryAppearances[category]; ok {
		return appearance
	}
	return prCategoryAppearances[domain.PRCategoryOther]
}

type PRListViewModel struct {
	// Source data (never mutated by sorting/filtering)
	sourcePRs    []domain.PullRequest
	sourceGroups []domain.PRGroup

	// Derived view data (filtered + sorted)
	visiblePRs []domain.PullRequest

	// commentThreads caches each PR's thread summary for the Info pane, fetched while it is shown.
	commentThreads  map[string]domain.CommentThreads
	commentsPending map[string]bool
	cursor          listCursor
	filter          listFilter
	split           splitView
	settings        domain.Settings
	width           int
	height          int
}

func NewPRListView() *PRListViewModel {
	return &PRListViewModel{
		commentThreads:  make(map[string]domain.CommentThreads),
		commentsPending: make(map[string]bool),
		cursor:          newListCursor(),
		filter:          newListFilter("Filter by title, author, or PR number..."),
	}
}

func (m *PRListViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.cursor.SetVisibleRows(height - listChromeRows)
}

func (m *PRListViewModel) SetSettings(settings domain.Settings) {
	m.settings = settings
	m.rebuild()
}

func (m *PRListViewModel) SetPRs(prs []domain.PullRequest) {
	m.sourceGroups = nil
	m.sourcePRs = append([]domain.PullRequest(nil), prs...)
	m.rebuild()
}

func (m *PRListViewModel) SetPRGroups(groups []domain.PRGroup) {
	m.sourceGroups = groups
	m.sourcePRs = flattenGroups(groups)
	m.rebuild()
}

// rebuild runs source → filter → sort → visible, keeping the cursor on the same PR so streamed
// results arriving from other PATs do not move the selection.
func (m *PRListViewModel) rebuild() {
	selectedKey := ""
	if pr := m.GetSelectedPR(); pr != nil {
		selectedKey = pr.Key()
	}

	filtered := m.filterOldOtherPRs(m.sourcePRs)
	filtered = m.filterPRs(filtered)
	m.visiblePRs = sortPRs(filtered)

	for i, pr := range m.visiblePRs {
		if pr.Key() == selectedKey {
			m.cursor.SetIndex(i, len(m.visiblePRs))
			return
		}
	}
	m.cursor.Clamp(len(m.visiblePRs))
}

func sortPRs(prs []domain.PullRequest) []domain.PullRequest {
	out := append([]domain.PullRequest(nil), prs...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return prCategoryOrder[out[i].Category] < prCategoryOrder[out[j].Category]
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func (m *PRListViewModel) filterOldOtherPRs(prs []domain.PullRequest) []domain.PullRequest {
	if !m.settings.HideOldOtherPRs {
		return prs
	}

	maxAge := time.Duration(m.settings.HideOldOtherPRsDays) * hoursPerDay * time.Hour
	cutoff := time.Now().Add(-maxAge)

	var out []domain.PullRequest
	for _, pr := range prs {
		if pr.Category == domain.PRCategoryOther && pr.CreatedAt.Before(cutoff) {
			continue
		}
		out = append(out, pr)
	}
	return out
}

func (m *PRListViewModel) filterPRs(prs []domain.PullRequest) []domain.PullRequest {
	var out []domain.PullRequest
	for _, pr := range prs {
		if m.filter.Matches(pr.Title, pr.Author.Username, strconv.Itoa(pr.Number)) {
			out = append(out, pr)
		}
	}
	return out
}

func (m *PRListViewModel) GetSelectedPR() *domain.PullRequest {
	idx := m.cursor.Index()
	if idx < 0 || idx >= len(m.visiblePRs) {
		return nil
	}
	return &m.visiblePRs[idx]
}

// SelectPR moves the cursor to the PR with key and reports whether it is listed.
func (m *PRListViewModel) SelectPR(key string) bool {
	for i, pr := range m.visiblePRs {
		if pr.Key() == key {
			m.cursor.SetIndex(i, len(m.visiblePRs))
			return true
		}
	}
	return false
}

func (m *PRListViewModel) MoveUp() {
	m.cursor.MoveUp()
	m.split.ResetScroll()
}

func (m *PRListViewModel) MoveDown() {
	m.cursor.MoveDown(len(m.visiblePRs))
	m.split.ResetScroll()
}

func (m *PRListViewModel) ActivateFilter() {
	m.filter.Activate()
}

func (m *PRListViewModel) DeactivateFilter() {
	m.filter.Deactivate()
}

func (m *PRListViewModel) ClearFilter() {
	m.filter.Clear()
	m.rebuild()
}

func (m *PRListViewModel) IsFiltering() bool {
	return m.filter.IsActive()
}

func (m *PRListViewModel) UpdateFilter(msg tea.Msg) tea.Cmd {
	cmd := m.filter.Update(msg)
	m.rebuild()
	m.cursor.Reset()
	return cmd
}

func (m *PRListViewModel) GetFilterText() string {
	return m.filter.Text()
}

// PendingCommentRequest returns the selected PR when the open Info pane needs its comment threads
// counted, and marks the request in flight so cursor movement does not refetch it.
func (m *PRListViewModel) PendingCommentRequest() *domain.PullRequest {
	pr := m.GetSelectedPR()
	if !m.split.IsOpen() || pr == nil {
		return nil
	}
	key := pr.Key()
	if _, loaded := m.commentThreads[key]; loaded || m.commentsPending[key] {
		return nil
	}
	m.commentsPending[key] = true
	return pr
}

func (m *PRListViewModel) SetCommentThreads(prKey string, threads domain.CommentThreads) {
	delete(m.commentsPending, prKey)
	m.commentThreads[prKey] = threads
}

// CommentThreadsFailed forgets the request so the count is fetched again next time.
func (m *PRListViewModel) CommentThreadsFailed(prKey string) {
	delete(m.commentsPending, prKey)
}

// AllPRs is every loaded pull request, before filtering.
func (m *PRListViewModel) AllPRs() []domain.PullRequest {
	return m.sourcePRs
}

func (m *PRListViewModel) TogglePreview() {
	m.split.Toggle()
}

func (m *PRListViewModel) IsPreviewOpen() bool {
	return m.split.IsOpen()
}

func (m *PRListViewModel) FocusNextPane() {
	m.split.FocusNext(len(m.detailPanes()))
}

func (m *PRListViewModel) FocusPrevPane() {
	m.split.FocusPrev()
}

func (m *PRListViewModel) IsPaneFocused() bool {
	return m.split.IsDetailFocused()
}

func (m *PRListViewModel) ScrollPane(delta int) {
	m.split.ScrollFocused(m.detailPanes(), delta, m.width, m.height)
}

func (m *PRListViewModel) columns() []listColumn {
	return prColumns(m.split.ListWidth(m.width), m.split.IsOpen())
}

// prColumns lays out pull request rows: compact leaves out the repository and author.
func prColumns(listWidth int, compact bool) []listColumn {
	const (
		approvalWidth = 1
		repoWidth     = 40
		numberWidth   = 7
		authorWidth   = 15
		ageWidth      = 14
		minTitleWidth = 20
	)
	if compact {
		const columnSpacing = 3
		return []listColumn{
			{"", approvalWidth},
			{"Title", flexColumnWidth(listWidth, approvalWidth+numberWidth+ageWidth+columnSpacing, minTitleWidth)},
			{"#", numberWidth},
			{"Age", ageWidth},
		}
	}
	const columnSpacing = 5
	fixed := approvalWidth + repoWidth + numberWidth + authorWidth + ageWidth + columnSpacing
	return []listColumn{
		{"", approvalWidth},
		{"Title", flexColumnWidth(listWidth, fixed, minTitleWidth)},
		{"Repo", repoWidth},
		{"#", numberWidth},
		{"Author", authorWidth},
		{"Age", ageWidth},
	}
}

func (m *PRListViewModel) View() string {
	var b strings.Builder
	b.WriteString(renderListTitle(prListTitle))
	b.WriteString("\n\n")

	switch {
	case len(m.visiblePRs) == 0 && m.filter.Text() != "":
		b.WriteString(listMutedStyle.Render("No pull requests match the filter."))
		b.WriteString("\n")
	case len(m.visiblePRs) == 0:
		b.WriteString(listMutedStyle.Render("No pull requests loaded."))
		b.WriteString("\n")
	default:
		columns := m.columns()
		b.WriteString(renderListRows(columns, len(m.visiblePRs), m.cursor, func(i int) listRow {
			return prRow(m.visiblePRs[i], m.split.IsOpen())
		}))
	}

	if m.filter.IsActive() {
		b.WriteString("\n")
		b.WriteString(m.filter.View())
	}

	return m.split.Render(b.String(), m.detailPanes(), m.width, m.height)
}

func prRow(pr domain.PullRequest, compact bool) listRow {
	appearance := prCategoryAppearanceFor(pr.Category)
	number := fmt.Sprintf("#%d", pr.Number)
	age := FormatAge(pr.CreatedAt)
	badge := prApprovalBadges[pr.ApprovalStatus]

	values := []string{badge, pr.Title, pr.Repository.FullName, number, pr.Author.Username, age}
	if compact {
		values = []string{badge, pr.Title, number, age}
	}
	return listRow{
		icon:   renderListIcon(appearance.icon, appearance.color),
		values: values,
		style:  appearance.rowStyle,
	}
}

func (m *PRListViewModel) detailPanes() []detailPane {
	pr := m.GetSelectedPR()
	if pr == nil {
		empty := []string{listMutedStyle.Render(noPRSelectedText)}
		return []detailPane{{title: prInfoPaneTitle, lines: empty, fitContent: true}, {title: prDescriptionTitle, lines: empty}}
	}
	return []detailPane{
		{title: fmt.Sprintf("#%d %s", pr.Number, pr.Title), lines: m.prInfoLines(*pr), fitContent: true},
		{title: prDescriptionTitle, lines: m.descriptionLines()},
	}
}

func (m *PRListViewModel) prInfoLines(pr domain.PullRequest) []string {
	status := string(pr.Status)
	if pr.IsDraft {
		status += " (draft)"
	}
	details := []struct {
		label string
		value string
	}{
		{"Repo", pr.Repository.FullName},
		{"Author", pr.Author.Username},
		{"Branch", fmt.Sprintf("%s → %s", valueOrPlaceholder(pr.SourceBranch), valueOrPlaceholder(pr.TargetBranch))},
		{"Status", status},
		{"Review", valueOrPlaceholder(prApprovalLabels[pr.ApprovalStatus])},
		{"Category", string(pr.Category)},
		{"Created", formatOptionalAge(pr.CreatedAt)},
		{"Updated", formatOptionalAge(pr.UpdatedAt)},
		{"Comments", m.commentThreadsText(pr)},
	}
	lines := make([]string, 0, len(details))
	for _, detail := range details {
		lines = append(lines, renderDetailLabel(detail.label, prDetailLabelWidth)+detailValueStyle.Render(valueOrPlaceholder(detail.value)))
	}
	return lines
}

// descriptionLines wraps the selected PR's description to the pane so it scrolls by visual line.
func (m *PRListViewModel) descriptionLines() []string {
	description := strings.TrimSpace(m.GetSelectedPR().Description)
	if description == "" {
		return []string{listMutedStyle.Render(noDescriptionText)}
	}
	_, width := splitWidths(m.width)
	wrapped := lipgloss.NewStyle().Width(width).Render(description)
	return strings.Split(wrapped, "\n")
}

func flattenGroups(groups []domain.PRGroup) []domain.PullRequest {
	var out []domain.PullRequest
	for _, g := range groups {
		out = append(out, g.PRs...)
	}
	return out
}

func (m *PRListViewModel) commentThreadsText(pr domain.PullRequest) string {
	if threads, loaded := m.commentThreads[pr.Key()]; loaded {
		return "💬 " + threads.Describe()
	}
	return loadingCommentsText
}
