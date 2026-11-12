package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/johanforsgren/lgtmfaster/internal/ui/markdown"
)

type PRInspectMode int

const (
	PRInspectModeDescription PRInspectMode = iota
	PRInspectModeDiff
	// PRInspectModeCommits lists the pull request's commits; opening one shows its diff in
	// PRInspectModeDiff with viewingCommit set.
	PRInspectModeCommits
)

type DiffViewMode int

const (
	DiffViewModeFull    DiffViewMode = iota // Show all lines including context
	DiffViewModeCompact                     // Show only added/deleted lines
)

type PRInspectViewModel struct {
	pr *domain.PullRequest
	// sourceDiff is the pull request's diff; diff is what is shown of it after the filters.
	sourceDiff       *domain.Diff
	diff             *domain.Diff
	hiddenExtensions map[string]bool
	ignoreWhitespace bool
	commits          []domain.Commit
	commitsLoaded    bool
	commitCursor     listCursor
	viewingCommit    *domain.Commit
	// prDiff keeps the pull request's diff while a commit's diff is the source.
	prDiff         *domain.Diff
	comments       []domain.Comment
	viewport       viewport.Model
	currentFile    int
	currentLineIdx int
	// cursorRow is the row of the diff content the cursor's line was rendered on, noCursorRow when
	// it is not shown.
	cursorRow int
	motion    diffMotion
	search    searchField
	// diffHeader is the sticky file header above the diff, empty outside the diff.
	diffHeader      string
	width           int
	height          int
	showComments    bool
	mode            PRInspectMode
	diffViewMode    DiffViewMode
	pendingComments []domain.Comment
	contentLines    int
	mdRenderer      *markdown.Renderer
	split           splitView
	workItems       []domain.WorkItem
	workItemsState  workItemsState
}

func NewPRInspectView() *PRInspectViewModel {
	vp := viewport.New(0, 0)

	return &PRInspectViewModel{
		commitCursor: newListCursor(),
		cursorRow:    noCursorRow,
		search:       newSearchField("text in the diff"),
		viewport:     vp,
		currentFile:  0,
		showComments: false,
		mode:         PRInspectModeDescription,
		mdRenderer:   markdown.NewRenderer(markdown.DefaultStyles()),
	}
}

func (m *PRInspectViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Height = max(height-prInspectStatusRows, minPaneSize)
	m.commitCursor.SetVisibleRows(height - listChromeRows)
	m.updateViewport()
}

func (m *PRInspectViewModel) SetPR(pr *domain.PullRequest) {
	if m.pr == nil || pr == nil || m.pr.Key() != pr.Key() {
		m.commits = nil
		m.commitsLoaded = false
		m.viewingCommit = nil
		m.commitCursor.Reset()
		m.resetWorkItems()
	}
	m.pr = pr
	m.mode = PRInspectModeDescription
	m.updateViewport()
}

func (m *PRInspectViewModel) SetDiff(diff *domain.Diff) {
	if m.viewingCommit != nil {
		m.prDiff = diff
		return
	}
	m.sourceDiff = diff
	m.diff = filterDiff(diff, m.hiddenExtensions, m.ignoreWhitespace)
	m.currentFile = 0
	m.currentLineIdx = 0
	logger.Log("PRInspectView: SetDiff called with %d files", len(diff.Files))
	if len(diff.Files) > 0 {
		for i, file := range diff.Files {
			logger.Log("PRInspectView: File %d: %s -> %s (%d hunks)", i+1, file.OldPath, file.NewPath, len(file.Hunks))
			if len(file.Hunks) > 0 {
				logger.Log("PRInspectView: File %d has %d lines in first hunk", i+1, len(file.Hunks[0].Lines))
			}
		}
	}
	m.updateViewport()
}

func (m *PRInspectViewModel) SetComments(comments []domain.Comment) {
	m.comments = comments
	m.updateViewport()
}

func (m *PRInspectViewModel) GetPR() *domain.PullRequest {
	return m.pr
}

func (m *PRInspectViewModel) GetComments() []domain.Comment {
	return m.comments
}

// GetDiff is the pull request's whole diff, whatever the filters hide or which commit is open.
func (m *PRInspectViewModel) GetDiff() *domain.Diff {
	if m.viewingCommit != nil {
		return m.prDiff
	}
	return m.sourceDiff
}

func (m *PRInspectViewModel) ShowCommits() {
	m.mode = PRInspectModeCommits
	m.updateViewport()
}

func (m *PRInspectViewModel) HasCommits() bool {
	return m.commitsLoaded
}

func (m *PRInspectViewModel) SetCommits(commits []domain.Commit) {
	m.commits = commits
	m.commitsLoaded = true
	m.commitCursor.Clamp(len(commits))
	m.updateViewport()
}

func (m *PRInspectViewModel) SelectedCommit() *domain.Commit {
	if index := m.commitCursor.Index(); index < len(m.commits) {
		return &m.commits[index]
	}
	return nil
}

func (m *PRInspectViewModel) MoveCommitUp() {
	m.commitCursor.MoveUp()
	m.updateViewport()
}

func (m *PRInspectViewModel) MoveCommitDown() {
	m.commitCursor.MoveDown(len(m.commits))
	m.updateViewport()
}

// ShowCommitDiff makes the commit's diff the one shown and filtered, keeping the pull request's.
func (m *PRInspectViewModel) ShowCommitDiff(commit domain.Commit, diff *domain.Diff) {
	if m.viewingCommit == nil {
		m.prDiff = m.sourceDiff
	}
	m.viewingCommit = &commit
	m.sourceDiff = diff
	m.mode = PRInspectModeDiff
	m.applyFilters()
}

// LeaveCommit goes back from a commit's diff to the commit list, restoring the pull request's diff.
func (m *PRInspectViewModel) LeaveCommit() {
	m.sourceDiff = m.prDiff
	m.prDiff = nil
	m.viewingCommit = nil
	m.mode = PRInspectModeCommits
	m.applyFilters()
}

func (m *PRInspectViewModel) ViewingCommit() *domain.Commit {
	return m.viewingCommit
}

func (m *PRInspectViewModel) ToggleIgnoreWhitespace() {
	m.ignoreWhitespace = !m.ignoreWhitespace
	m.applyFilters()
}

func (m *PRInspectViewModel) IgnoresWhitespace() bool {
	return m.ignoreWhitespace
}

// Extensions lists the file extensions of the whole diff, for the extension filter.
func (m *PRInspectViewModel) Extensions() []ExtensionCount {
	return DiffExtensions(m.sourceDiff)
}

func (m *PRInspectViewModel) HiddenExtensions() map[string]bool {
	hidden := make(map[string]bool, len(m.hiddenExtensions))
	for extension, isHidden := range m.hiddenExtensions {
		hidden[extension] = isHidden
	}
	return hidden
}

func (m *PRInspectViewModel) SetHiddenExtensions(hidden map[string]bool) {
	m.hiddenExtensions = hidden
	m.applyFilters()
}

// applyFilters rebuilds the shown diff, staying on the same file when it is still shown.
func (m *PRInspectViewModel) applyFilters() {
	currentPath := ""
	if m.diff != nil && m.currentFile < len(m.diff.Files) {
		currentPath = getFilePath(m.diff.Files[m.currentFile])
	}
	m.diff = filterDiff(m.sourceDiff, m.hiddenExtensions, m.ignoreWhitespace)
	m.currentFile = 0
	m.currentLineIdx = 0
	if m.diff != nil {
		for i, file := range m.diff.Files {
			if getFilePath(file) == currentPath {
				m.currentFile = i
			}
		}
	}
	m.updateViewport()
}

func (m *PRInspectViewModel) SwitchToDiff() {
	m.mode = PRInspectModeDiff
	m.updateViewport()
}

func (m *PRInspectViewModel) SwitchToDescription() {
	m.mode = PRInspectModeDescription
	m.updateViewport()
}

func (m *PRInspectViewModel) GetMode() PRInspectMode {
	return m.mode
}

func (m *PRInspectViewModel) GetDiffViewMode() DiffViewMode {
	return m.diffViewMode
}

func (m *PRInspectViewModel) ToggleDiffViewMode() {
	if m.diffViewMode == DiffViewModeFull {
		m.diffViewMode = DiffViewModeCompact
	} else {
		m.diffViewMode = DiffViewModeFull
	}
	m.updateViewport()
}

func (m *PRInspectViewModel) NextFile() {
	if m.diff != nil && m.currentFile < len(m.diff.Files)-1 {
		m.currentFile++
		m.currentLineIdx = 0
		m.updateViewport()
	}
}

func (m *PRInspectViewModel) PrevFile() {
	if m.currentFile > 0 {
		m.currentFile--
		m.currentLineIdx = 0
		m.updateViewport()
	}
}

func (m *PRInspectViewModel) ToggleComments() {
	m.showComments = !m.showComments
	m.updateViewport()
}

func (m *PRInspectViewModel) NextLine() {
	if m.diff == nil || len(m.diff.Files) == 0 {
		return
	}
	file := m.diff.Files[m.currentFile]
	totalLines := m.countTotalLines(file)
	if m.currentLineIdx < totalLines-1 {
		m.currentLineIdx++
		m.updateViewport()
		m.ensureLineVisible()
	}
}

func (m *PRInspectViewModel) PrevLine() {
	if m.currentLineIdx > 0 {
		m.currentLineIdx--
		m.updateViewport()
		m.ensureLineVisible()
	}
}

func (m *PRInspectViewModel) clampViewportOffset() {
	if m.viewport.YOffset < 0 {
		m.viewport.YOffset = 0
	}

	maxOffset := m.contentLines - m.viewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.viewport.YOffset > maxOffset {
		m.viewport.YOffset = maxOffset
	}
}

func (m *PRInspectViewModel) countTotalLines(file domain.FileDiff) int {
	count := 0
	for _, hunk := range file.Hunks {
		count += len(hunk.Lines)
	}
	return count
}

func (m *PRInspectViewModel) hunkHasChanges(hunk domain.DiffHunk) bool {
	for _, line := range hunk.Lines {
		if line.Type == "add" || line.Type == "delete" {
			return true
		}
	}
	return false
}

func (m *PRInspectViewModel) GetCurrentLineInfo() *domain.DiffLine {
	if m.diff == nil || len(m.diff.Files) == 0 {
		return nil
	}

	file := m.diff.Files[m.currentFile]
	lineIdx := 0
	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			if lineIdx == m.currentLineIdx {
				return &line
			}
			lineIdx++
		}
	}
	return nil
}

// CurrentCommentTarget is the diff line under the cursor, which an inline comment would go on.
func (m *PRInspectViewModel) CurrentCommentTarget() (CommentTarget, bool) {
	if m.diff == nil || len(m.diff.Files) == 0 {
		return CommentTarget{}, false
	}
	return commentTargetAt(m.diff.Files[m.currentFile], m.currentLineIdx)
}

// AddPendingComment queues a comment on the line under the cursor, to be posted with the review.
func (m *PRInspectViewModel) AddPendingComment(body string) {
	target, ok := m.CurrentCommentTarget()
	if !ok {
		return
	}
	m.pendingComments = append(m.pendingComments, domain.Comment{
		Body:     body,
		FilePath: target.FilePath,
		Line:     target.Line,
		Side:     target.Side,
	})
}

func (m *PRInspectViewModel) GetPendingComments() []domain.Comment {
	return m.pendingComments
}

func (m *PRInspectViewModel) ClearPendingComments() {
	m.pendingComments = []domain.Comment{}
}

func (m *PRInspectViewModel) GetPendingCommentCount() int {
	return len(m.pendingComments)
}

func (m *PRInspectViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

const (
	commitHeaderFormat   = "Commit %s: %s"
	commitsTitle         = "Commits"
	loadingCommitsText   = "Loading commits..."
	noCommitsText        = "This pull request has no commits."
	commitIDWidth        = 9
	commitAuthorWidth    = 18
	commitAgeWidth       = 14
	minCommitSubject     = 20
	allFilesFilteredText = "Every file is hidden by the extension filter (F)."
	prInspectStatusRows  = 2
	// fileHeaderPadding is the file header's padding on both sides.
	fileHeaderPadding   = 2
	diffViewModeFull    = "full"
	diffViewModeCompact = "compact"
)

var prInspectStatusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

// View frames the content in the main window, beside the work items pane when it is open.
func (m *PRInspectViewModel) View() string {
	content := m.diffHeader + m.viewport.View() + "\n\n" + prInspectStatusStyle.Render(m.statusLine())
	if !m.showsWorkItemsPane() {
		return RenderMainWindow(content, m.width, m.height)
	}
	return m.split.Render(content, m.workItemPanes(), m.width, m.height)
}

// statusLine reports the diff state the key hints in the top bar cannot show.
func (m *PRInspectViewModel) statusLine() string {
	if m.mode != PRInspectModeDiff {
		return ""
	}
	viewMode := diffViewModeFull
	if m.diffViewMode == DiffViewModeCompact {
		viewMode = diffViewModeCompact
	}
	status := "Diff view: " + viewMode
	if m.ignoreWhitespace {
		status += " | whitespace changes hidden"
	}
	if hidden := m.hiddenExtensionNames(); len(hidden) > 0 && m.sourceDiff != nil {
		status += fmt.Sprintf(" | %d/%d files, hiding %s", len(m.diff.Files), len(m.sourceDiff.Files), strings.Join(hidden, ", "))
	}
	if pending := m.GetPendingCommentCount(); pending > 0 {
		status += fmt.Sprintf(" | %d pending comments", pending)
	}
	if motion := m.PendingMotion(); motion != "" {
		status += " | " + motion
	}
	if search := m.search.View(m.searchPosition()); search != "" {
		status += " | " + search
	}
	return status
}

func (m *PRInspectViewModel) hiddenExtensionNames() []string {
	var names []string
	for _, extension := range m.Extensions() {
		if m.hiddenExtensions[extension.Extension] {
			names = append(names, extension.Extension)
		}
	}
	return names
}

func (m *PRInspectViewModel) updateViewport() {
	m.diffHeader = m.renderDiffHeader()
	m.viewport.Height = max(m.height-prInspectStatusRows-strings.Count(m.diffHeader, "\n"), minPaneSize)
	m.viewport.Width = m.contentWidth()
	m.mdRenderer.SetWidth(m.viewport.Width)

	var b strings.Builder

	switch m.mode {
	case PRInspectModeDescription:
		if m.pr != nil {
			b.WriteString(m.renderPRHeader())
		}
	case PRInspectModeCommits:
		b.WriteString(m.renderCommits())
	case PRInspectModeDiff:
		if m.diff != nil && len(m.diff.Files) > 0 {
			b.WriteString(m.renderDiff())
		} else if m.sourceDiff != nil && len(m.sourceDiff.Files) > 0 {
			b.WriteString(listMutedStyle.Render(allFilesFilteredText))
		}
	}

	content := b.String()
	m.contentLines = strings.Count(content, "\n") + 1
	m.viewport.SetContent(content)
}

func (m *PRInspectViewModel) renderPRHeader() string {
	if m.pr == nil {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true)

	b.WriteString(titleStyle.Render(m.pr.Title))
	b.WriteString("\n")

	metaStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	meta := fmt.Sprintf("%s #%d | %s → %s | by %s",
		m.pr.Repository.FullName,
		m.pr.Number,
		m.pr.SourceBranch,
		m.pr.TargetBranch,
		m.pr.Author.Username,
	)
	b.WriteString(metaStyle.Render(meta))
	b.WriteString("\n")

	statusStyle := lipgloss.NewStyle()
	statusText := string(m.pr.Status)
	switch m.pr.Status {
	case domain.PRStatusOpen:
		statusStyle = statusStyle.Foreground(lipgloss.Color("#10B981"))
	case domain.PRStatusClosed:
		statusStyle = statusStyle.Foreground(lipgloss.Color("#EF4444"))
	case domain.PRStatusMerged:
		statusStyle = statusStyle.Foreground(lipgloss.Color("#7C3AED"))
	}

	if m.pr.IsDraft {
		statusText += " (draft)"
	}

	b.WriteString(statusStyle.Render(statusText))
	b.WriteString("\n")

	if m.pr.Description != "" {
		dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#374151"))
		divider := strings.Repeat("─", max(m.viewport.Width-4, 0))
		b.WriteString("\n")
		b.WriteString(dividerStyle.Render(divider))
		b.WriteString("\n\n")
		b.WriteString(m.mdRenderer.Render(m.pr.Description))
	}

	return b.String()
}

var fileHeaderStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#7C3AED")).
	Bold(true).
	Background(lipgloss.Color("#1F2937")).
	Padding(0, 1)

// renderDiffHeader names the shown file (and commit), with a blank row under it. It stays above
// the scrolling diff, so the file name is always in view.
func (m *PRInspectViewModel) renderDiffHeader() string {
	if m.mode != PRInspectModeDiff || m.diff == nil || len(m.diff.Files) == 0 {
		return ""
	}
	width := m.contentWidth()
	var b strings.Builder
	if m.viewingCommit != nil {
		b.WriteString(listTitleStyle.Render(truncateString(fmt.Sprintf(commitHeaderFormat, m.viewingCommit.ShortID(), m.viewingCommit.Subject()), width)))
		b.WriteString("\n")
	}
	header := fmt.Sprintf("File %d/%d: %s", m.currentFile+1, len(m.diff.Files), getFilePath(m.diff.Files[m.currentFile]))
	b.WriteString(fileHeaderStyle.Render(truncateString(header, max(width-fileHeaderPadding, minPaneSize))))
	b.WriteString("\n\n")
	return b.String()
}

func (m *PRInspectViewModel) renderDiff() string {
	if m.diff == nil || len(m.diff.Files) == 0 {
		logger.Log("PRInspectView: renderDiff - No diff available (diff nil: %v, files: %d)", m.diff == nil, 0)
		return "No diff available"
	}

	logger.Log("PRInspectView: renderDiff - Rendering file %d of %d", m.currentFile+1, len(m.diff.Files))

	var b strings.Builder
	file := m.diff.Files[m.currentFile]

	logger.Log("PRInspectView: renderDiff - File has %d hunks", len(file.Hunks))

	m.cursorRow = noCursorRow
	gutterDigits := diffGutterDigits(file)
	lineIdx := 0
	for hunkIdx, hunk := range file.Hunks {
		hunkHeaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3B82F6"))

		hasVisibleLines := m.diffViewMode == DiffViewModeFull || m.hunkHasChanges(hunk)
		if hasVisibleLines {
			b.WriteString(hunkHeaderStyle.Render(hunk.Header))
			b.WriteString("\n")
		}

		logger.Log("PRInspectView: renderDiff - Hunk %d has %d lines", hunkIdx+1, len(hunk.Lines))

		for _, line := range hunk.Lines {
			if !m.isLineShown(line) {
				lineIdx++
				continue
			}
			if lineIdx == m.currentLineIdx {
				m.cursorRow = strings.Count(b.String(), "\n")
			}
			b.WriteString(m.renderDiffLine(line, lineIdx, gutterDigits))
			b.WriteString("\n")
			lineIdx++
		}

		if hasVisibleLines {
			b.WriteString("\n")
		}
	}

	if m.showComments {
		b.WriteString(m.renderComments(getFilePath(file)))
	}

	result := b.String()
	logger.Log("PRInspectView: renderDiff - Generated %d bytes of content", len(result))
	return result
}

func (m *PRInspectViewModel) renderDiffLine(line domain.DiffLine, lineIdx, gutterDigits int) string {
	style := lipgloss.NewStyle()

	switch line.Type {
	case "add":
		style = style.Foreground(lipgloss.Color("#10B981"))
	case "delete":
		style = style.Foreground(lipgloss.Color("#EF4444"))
	default:
		style = style.Foreground(lipgloss.Color("#6B7280"))
	}

	if m.search.Matches(line.Content) {
		style = style.Inherit(searchMatchStyle)
	}

	isCursor := lineIdx == m.currentLineIdx
	hasPendingComment := m.hasPendingCommentOnLine(line)
	hasSubmittedComment := m.hasSubmittedCommentOnLine(line)

	prefix := ""
	if isCursor {
		prefix = "► "
		style = style.Bold(true).Background(lipgloss.Color("#374151")).Underline(true)
	} else {
		prefix = "  "
	}

	if hasPendingComment {
		prefix += "💬 "
	} else if hasSubmittedComment {
		prefix += "💭 "
	}

	return renderDiffGutter(line, gutterDigits) + style.Render(prefix+line.Content)
}

func (m *PRInspectViewModel) hasPendingCommentOnLine(line domain.DiffLine) bool {
	return m.hasCommentOnLine(m.pendingComments, line)
}

func (m *PRInspectViewModel) hasSubmittedCommentOnLine(line domain.DiffLine) bool {
	return m.hasCommentOnLine(m.comments, line)
}

func (m *PRInspectViewModel) hasCommentOnLine(comments []domain.Comment, line domain.DiffLine) bool {
	if m.diff == nil || len(m.diff.Files) == 0 {
		return false
	}
	filePath := getFilePath(m.diff.Files[m.currentFile])
	for _, comment := range comments {
		if comment.FilePath == filePath && commentIsOnLine(comment, line) {
			return true
		}
	}
	return false
}

func (m *PRInspectViewModel) renderComments(filePath string) string {
	var b strings.Builder

	b.WriteString("\n")
	commentHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Bold(true)
	b.WriteString(commentHeaderStyle.Render("Comments"))
	b.WriteString("\n\n")

	relevantComments := []domain.Comment{}
	for _, comment := range m.comments {
		if comment.FilePath == filePath || comment.FilePath == "" {
			relevantComments = append(relevantComments, comment)
		}
	}

	if len(relevantComments) == 0 {
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true).
			Render("No comments"))
		return b.String()
	}

	for _, comment := range relevantComments {
		authorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)

		b.WriteString(authorStyle.Render(comment.Author.Username))
		if comment.Line > 0 {
			b.WriteString(fmt.Sprintf(" on line %d", comment.Line))
		}
		b.WriteString(":\n")

		commentStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FAFB")).
			PaddingLeft(2)
		b.WriteString(commentStyle.Render(comment.Body))
		b.WriteString("\n\n")
	}

	return b.String()
}

func (m *PRInspectViewModel) GetCurrentFileDiffText() string {
	if m.diff == nil || len(m.diff.Files) == 0 {
		return ""
	}
	return m.generateFileDiffText(m.diff.Files[m.currentFile])
}

func (m *PRInspectViewModel) GetAllFilesDiffText() string {
	if m.diff == nil || len(m.diff.Files) == 0 {
		return ""
	}

	var b strings.Builder
	for i, file := range m.diff.Files {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("=== %s ===\n", getFilePath(file)))
		b.WriteString(m.generateFileDiffText(file))
	}
	return b.String()
}

func (m *PRInspectViewModel) generateFileDiffText(file domain.FileDiff) string {
	var b strings.Builder

	for _, hunk := range file.Hunks {
		hasVisibleLines := m.diffViewMode == DiffViewModeFull || m.hunkHasChanges(hunk)
		if hasVisibleLines {
			b.WriteString(hunk.Header)
			b.WriteString("\n")
		}

		for _, line := range hunk.Lines {
			if m.diffViewMode == DiffViewModeCompact && line.Type == "context" {
				continue
			}
			b.WriteString(line.Content)
			b.WriteString("\n")
		}
	}

	return b.String()
}

func getFilePath(file domain.FileDiff) string {
	if file.NewPath != "" {
		return file.NewPath
	}
	return file.OldPath
}

func (m *PRInspectViewModel) renderCommits() string {
	var b strings.Builder
	title := commitsTitle
	if m.pr != nil {
		title = fmt.Sprintf("#%d %s / %s", m.pr.Number, m.pr.Title, commitsTitle)
	}
	b.WriteString(renderListTitle(title))
	b.WriteString("\n\n")

	switch {
	case !m.commitsLoaded:
		b.WriteString(listMutedStyle.Render(loadingCommitsText))
	case len(m.commits) == 0:
		b.WriteString(listMutedStyle.Render(noCommitsText))
	default:
		columns := []listColumn{
			{"Commit", commitIDWidth},
			{"Message", flexColumnWidth(m.width, commitIDWidth+commitAuthorWidth+commitAgeWidth+3, minCommitSubject)},
			{"Author", commitAuthorWidth},
			{"Age", commitAgeWidth},
		}
		b.WriteString(renderListRows(columns, len(m.commits), m.commitCursor, func(i int) listRow {
			commit := m.commits[i]
			return listRow{
				icon:   renderListIcon("●", lipgloss.Color("214")),
				values: []string{commit.ShortID(), commit.Subject(), commit.Author.Username, formatOptionalAge(commit.Date)},
				style:  listRowStyle,
			}
		}))
	}
	return b.String()
}
