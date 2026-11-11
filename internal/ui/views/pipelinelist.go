package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	pipelineFolderSeparator = `\`
	pipelineListChromeRows  = 12
	treeIndent              = "  "
	expandedMarker          = "▾ "
	collapsedMarker         = "▸ "
	leafMarker              = "  "
)

type PipelineNodeKind int

const (
	PipelineNodeProject PipelineNodeKind = iota
	PipelineNodeRepository
	PipelineNodePipeline
)

type PipelineNode struct {
	Kind          PipelineNodeKind
	Key           string
	ParentKey     string
	Name          string
	Depth         int
	Status        domain.PipelineStatus
	PipelineCount int
	Pipeline      *domain.Pipeline
	Children      []*PipelineNode
}

func (n *PipelineNode) IsGroup() bool {
	return n.Kind != PipelineNodePipeline
}

// groupStatusPriority decides which child status a folded group surfaces: anything needing
// attention (approval, failure) outranks in-flight work, which outranks settled outcomes.
var groupStatusPriority = []domain.PipelineStatus{
	domain.PipelineStatusWaitingForApproval,
	domain.PipelineStatusFailed,
	domain.PipelineStatusRunning,
	domain.PipelineStatusQueued,
	domain.PipelineStatusPartiallySucceeded,
	domain.PipelineStatusSucceeded,
	domain.PipelineStatusCanceled,
	domain.PipelineStatusSkipped,
}

type PipelineListViewModel struct {
	sourcePipelines []domain.Pipeline
	roots           []*PipelineNode
	visibleNodes    []*PipelineNode
	expanded        map[string]bool
	favorites       map[string]bool
	previewRuns     map[string][]domain.PipelineRun
	previewPending  map[string]bool
	cursor          listCursor
	filter          listFilter
	pane            splitPane
	width           int
	height          int
}

func NewPipelineListView() *PipelineListViewModel {
	return &PipelineListViewModel{
		expanded:       make(map[string]bool),
		favorites:      make(map[string]bool),
		previewRuns:    make(map[string][]domain.PipelineRun),
		previewPending: make(map[string]bool),
		cursor:         newListCursor(),
		filter:         newListFilter("Filter by project, repository, or pipeline name..."),
	}
}

func (m *PipelineListViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.cursor.SetVisibleRows(height - pipelineListChromeRows)
}

func (m *PipelineListViewModel) SetPipelines(pipelines []domain.Pipeline) {
	m.sourcePipelines = append([]domain.Pipeline(nil), pipelines...)
	m.rebuild()
}

func (m *PipelineListViewModel) SetFavorites(favorites map[string]bool) {
	m.favorites = favorites
	m.rebuild()
}

func (m *PipelineListViewModel) HasPipelines() bool {
	return len(m.sourcePipelines) > 0
}

func (m *PipelineListViewModel) PipelineCount() int {
	return len(m.sourcePipelines)
}

func (m *PipelineListViewModel) GetSelectedNode() *PipelineNode {
	idx := m.cursor.Index()
	if idx < 0 || idx >= len(m.visibleNodes) {
		return nil
	}
	return m.visibleNodes[idx]
}

func (m *PipelineListViewModel) GetSelectedPipeline() *domain.Pipeline {
	node := m.GetSelectedNode()
	if node == nil {
		return nil
	}
	return node.Pipeline
}

func (m *PipelineListViewModel) MoveUp() {
	m.cursor.MoveUp()
}

func (m *PipelineListViewModel) MoveDown() {
	m.cursor.MoveDown(len(m.visibleNodes))
}

// ToggleSelected folds/unfolds a group; on a pipeline it folds the parent repository and selects it.
func (m *PipelineListViewModel) ToggleSelected() {
	node := m.GetSelectedNode()
	if node == nil {
		return
	}
	if node.IsGroup() {
		m.expanded[node.Key] = !m.expanded[node.Key]
		m.rebuildKeepingSelection(node.Key)
		return
	}
	m.expanded[node.ParentKey] = false
	m.rebuildKeepingSelection(node.ParentKey)
}

// ToggleAll expands every group unless all are already expanded, in which case it folds them all.
func (m *PipelineListViewModel) ToggleAll() {
	var selectedKey string
	if node := m.GetSelectedNode(); node != nil {
		selectedKey = node.Key
	}

	groupKeys := collectGroupKeys(m.roots)
	allExpanded := true
	for _, key := range groupKeys {
		if !m.expanded[key] {
			allExpanded = false
			break
		}
	}

	m.expanded = make(map[string]bool)
	if !allExpanded {
		for _, key := range groupKeys {
			m.expanded[key] = true
		}
	} else if node := m.GetSelectedNode(); node != nil {
		selectedKey = rootKeyOf(node, m.roots)
	}
	m.rebuildKeepingSelection(selectedKey)
}

func (m *PipelineListViewModel) TogglePreview() {
	m.pane.Toggle()
}

func (m *PipelineListViewModel) IsPreviewOpen() bool {
	return m.pane.IsOpen()
}

// NextPreviewRequest returns the selected pipeline when the open pane needs its runs fetched,
// and marks the request in flight so cursor movement does not refetch it.
func (m *PipelineListViewModel) NextPreviewRequest() *domain.Pipeline {
	pipeline := m.GetSelectedPipeline()
	if !m.pane.IsOpen() || pipeline == nil {
		return nil
	}
	key := pipeline.FavoriteKey()
	if _, loaded := m.previewRuns[key]; loaded || m.previewPending[key] {
		return nil
	}
	m.previewPending[key] = true
	return pipeline
}

func (m *PipelineListViewModel) SetPreviewRuns(pipelineKey string, runs []domain.PipelineRun) {
	delete(m.previewPending, pipelineKey)
	m.previewRuns[pipelineKey] = runs
}

func (m *PipelineListViewModel) ResetPreview() {
	m.previewRuns = make(map[string][]domain.PipelineRun)
	m.previewPending = make(map[string]bool)
}

func (m *PipelineListViewModel) ActivateFilter() {
	m.filter.Activate()
}

func (m *PipelineListViewModel) DeactivateFilter() {
	m.filter.Deactivate()
}

func (m *PipelineListViewModel) ClearFilter() {
	m.filter.Clear()
	m.rebuild()
}

func (m *PipelineListViewModel) IsFiltering() bool {
	return m.filter.IsActive()
}

func (m *PipelineListViewModel) GetFilterText() string {
	return m.filter.Text()
}

func (m *PipelineListViewModel) UpdateFilter(msg tea.Msg) tea.Cmd {
	cmd := m.filter.Update(msg)
	m.rebuild()
	m.cursor.Reset()
	return cmd
}

func (m *PipelineListViewModel) rebuild() {
	var selectedKey string
	if node := m.GetSelectedNode(); node != nil {
		selectedKey = node.Key
	}
	m.rebuildKeepingSelection(selectedKey)
}

func (m *PipelineListViewModel) rebuildKeepingSelection(selectedKey string) {
	var filtered []domain.Pipeline
	for _, pipeline := range m.sourcePipelines {
		if m.filter.Matches(pipeline.Project, pipeline.Repository, PipelineDisplayName(pipeline)) {
			filtered = append(filtered, pipeline)
		}
	}

	m.roots = buildPipelineTree(filtered)
	m.sortTree(m.roots)

	// While filtering every group is unfolded so matches are never hidden inside a folded group.
	showAll := m.filter.Text() != ""
	m.visibleNodes = m.visibleNodes[:0]
	m.flatten(m.roots, showAll)

	for i, node := range m.visibleNodes {
		if node.Key == selectedKey {
			m.cursor.SetIndex(i, len(m.visibleNodes))
			return
		}
	}
	m.cursor.Clamp(len(m.visibleNodes))
}

func (m *PipelineListViewModel) flatten(nodes []*PipelineNode, showAll bool) {
	for _, node := range nodes {
		m.visibleNodes = append(m.visibleNodes, node)
		if node.IsGroup() && (showAll || m.expanded[node.Key]) {
			m.flatten(node.Children, showAll)
		}
	}
}

func (m *PipelineListViewModel) sortTree(nodes []*PipelineNode) {
	sort.SliceStable(nodes, func(i, j int) bool {
		a, b := nodes[i], nodes[j]
		if m.favorites[a.Key] != m.favorites[b.Key] {
			return m.favorites[a.Key]
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	for _, node := range nodes {
		m.sortTree(node.Children)
	}
}

func buildPipelineTree(pipelines []domain.Pipeline) []*PipelineNode {
	var roots []*PipelineNode
	projects := make(map[string]*PipelineNode)
	repos := make(map[string]*PipelineNode)

	for i := range pipelines {
		pipeline := &pipelines[i]
		projectKey := domain.ProjectFavoriteKey(pipeline.PATID, pipeline.Project)
		project, ok := projects[projectKey]
		if !ok {
			project = &PipelineNode{Kind: PipelineNodeProject, Key: projectKey, Name: pipeline.Project}
			projects[projectKey] = project
			roots = append(roots, project)
		}

		repoKey := domain.RepositoryFavoriteKey(pipeline.PATID, pipeline.Project, pipeline.Repository)
		repo, ok := repos[repoKey]
		if !ok {
			repo = &PipelineNode{Kind: PipelineNodeRepository, Key: repoKey, ParentKey: projectKey, Name: pipeline.Repository, Depth: 1}
			repos[repoKey] = repo
			project.Children = append(project.Children, repo)
		}

		status := domain.PipelineStatusUnknown
		if pipeline.LatestRun != nil {
			status = pipeline.LatestRun.Status
		}
		repo.Children = append(repo.Children, &PipelineNode{
			Kind:      PipelineNodePipeline,
			Key:       pipeline.FavoriteKey(),
			ParentKey: repoKey,
			Name:      PipelineDisplayName(*pipeline),
			Depth:     2,
			Status:    status,
			Pipeline:  pipeline,
		})
		repo.PipelineCount++
		project.PipelineCount++
	}

	for _, project := range roots {
		for _, repo := range project.Children {
			repo.Status = aggregateStatus(repo.Children)
		}
		project.Status = aggregateStatus(project.Children)
	}
	return roots
}

func aggregateStatus(children []*PipelineNode) domain.PipelineStatus {
	present := make(map[domain.PipelineStatus]bool, len(children))
	for _, child := range children {
		present[child.Status] = true
	}
	for _, status := range groupStatusPriority {
		if present[status] {
			return status
		}
	}
	return domain.PipelineStatusUnknown
}

func collectGroupKeys(nodes []*PipelineNode) []string {
	var keys []string
	for _, node := range nodes {
		if node.IsGroup() {
			keys = append(keys, node.Key)
			keys = append(keys, collectGroupKeys(node.Children)...)
		}
	}
	return keys
}

func rootKeyOf(node *PipelineNode, roots []*PipelineNode) string {
	for _, root := range roots {
		if root.Key == node.Key || containsKey(root.Children, node.Key) {
			return root.Key
		}
	}
	return node.Key
}

func containsKey(nodes []*PipelineNode, key string) bool {
	for _, node := range nodes {
		if node.Key == key || containsKey(node.Children, key) {
			return true
		}
	}
	return false
}

// PipelineDisplayName prefixes the pipeline name with its folder; the root folder is omitted.
func PipelineDisplayName(pipeline domain.Pipeline) string {
	folder := strings.Trim(pipeline.Folder, pipelineFolderSeparator)
	if folder == "" {
		return pipeline.Name
	}
	return folder + pipelineFolderSeparator + pipeline.Name
}

func (m *PipelineListViewModel) columns() []listColumn {
	const (
		lastRunWidth = 18
		branchWidth  = 20
		ageWidth     = 14
		minNameWidth = 24
	)
	listWidth := m.pane.ListWidth(m.width)
	if m.pane.IsOpen() {
		return []listColumn{
			{"Name", flexColumnWidth(listWidth, lastRunWidth+1, minNameWidth)},
			{"Last run", lastRunWidth},
		}
	}
	const columnSpacing = 3
	return []listColumn{
		{"Name", flexColumnWidth(listWidth, lastRunWidth+branchWidth+ageWidth+columnSpacing, minNameWidth)},
		{"Last run", lastRunWidth},
		{"Branch", branchWidth},
		{"Age", ageWidth},
	}
}

func (m *PipelineListViewModel) View() string {
	var b strings.Builder

	b.WriteString(pipelineTitleStyle.Render("Pipelines"))
	b.WriteString("\n\n")

	if len(m.visibleNodes) == 0 {
		if m.filter.Text() != "" {
			b.WriteString(pipelineMutedStyle.Render("No pipelines match the filter."))
		} else {
			b.WriteString(pipelineMutedStyle.Render("No pipelines loaded."))
		}
		b.WriteString("\n")
	} else {
		columns := m.columns()
		b.WriteString(renderListHeader(columns))
		b.WriteString("\n")

		start, end := m.cursor.VisibleRange(len(m.visibleNodes))
		for i := start; i < end; i++ {
			b.WriteString(m.renderNode(m.visibleNodes[i], columns, i == m.cursor.Index()))
			b.WriteString("\n")
		}
		if len(m.visibleNodes) > end-start {
			b.WriteString(renderScrollInfo(start, end, len(m.visibleNodes)))
			b.WriteString("\n")
		}
	}

	if m.filter.IsActive() {
		b.WriteString("\n")
		b.WriteString(m.filter.View())
	}

	b.WriteString("\n")
	b.WriteString(pipelineMutedStyle.Render(m.helpText()))

	title, lines := m.previewContent()
	return m.pane.Render(b.String(), title, lines, m.width, m.cursor.visibleRows)
}

func (m *PipelineListViewModel) renderNode(node *PipelineNode, columns []listColumn, selected bool) string {
	marker := leafMarker
	if node.IsGroup() {
		marker = collapsedMarker
		if m.expanded[node.Key] || m.filter.Text() != "" {
			marker = expandedMarker
		}
	}

	name := strings.Repeat(treeIndent, node.Depth) + marker + favoritePrefix(m.favorites[node.Key]) + node.Name
	lastRun, branch, age := emptyCellPlaceholder, emptyCellPlaceholder, emptyCellPlaceholder
	if node.IsGroup() {
		name = fmt.Sprintf("%s (%d)", name, node.PipelineCount)
	} else if run := node.Pipeline.LatestRun; run != nil {
		lastRun = valueOrPlaceholder(run.Number)
		branch = valueOrPlaceholder(run.SourceBranch)
		age = formatOptionalAge(run.QueuedAt)
	}

	values := []string{name, lastRun, branch, age}
	return renderListRow(node.Status, columns, values[:len(columns)], selected)
}

func (m *PipelineListViewModel) previewContent() (string, []string) {
	node := m.GetSelectedNode()
	if node == nil {
		return "Details", []string{pipelineMutedStyle.Render("Nothing selected.")}
	}

	switch node.Kind {
	case PipelineNodeProject:
		var lines []string
		for _, repo := range node.Children {
			lines = append(lines, renderDetailRow(repo.Status, fmt.Sprintf("%s (%d)", repo.Name, repo.PipelineCount)))
			for _, pipeline := range repo.Children {
				lines = append(lines, treeIndent+renderDetailRow(pipeline.Status, pipeline.Name))
			}
		}
		return "Pipelines in " + node.Name, lines
	case PipelineNodeRepository:
		var lines []string
		for _, pipeline := range node.Children {
			lines = append(lines, renderDetailRow(pipeline.Status, pipeline.Name+latestRunSuffix(pipeline.Pipeline)))
		}
		return "Pipelines in " + node.Name, lines
	default:
		return "Runs of " + node.Name, m.previewRunLines(node.Pipeline)
	}
}

func (m *PipelineListViewModel) previewRunLines(pipeline *domain.Pipeline) []string {
	runs, loaded := m.previewRuns[pipeline.FavoriteKey()]
	if !loaded {
		return []string{pipelineMutedStyle.Render("Loading runs...")}
	}
	if len(runs) == 0 {
		return []string{pipelineMutedStyle.Render("This pipeline has no runs yet.")}
	}
	lines := make([]string, 0, len(runs))
	for _, run := range runs {
		lines = append(lines, renderDetailRow(run.Status, fmt.Sprintf("%s  %s  %s",
			valueOrPlaceholder(run.Number), valueOrPlaceholder(run.SourceBranch), formatOptionalAge(run.QueuedAt))))
	}
	return lines
}

func latestRunSuffix(pipeline *domain.Pipeline) string {
	if pipeline == nil || pipeline.LatestRun == nil {
		return ""
	}
	return fmt.Sprintf("  %s  %s", valueOrPlaceholder(pipeline.LatestRun.Number), formatOptionalAge(pipeline.LatestRun.QueuedAt))
}

func (m *PipelineListViewModel) helpText() string {
	if m.filter.IsActive() {
		return "Type to filter | Enter/Esc: Close"
	}
	help := "Enter: Open/Fold | z/Z: Fold | f: Favorite | Tab: Details | ctrl+o: Open | r: Refresh | /: Filter | q: Back"
	if m.filter.Text() != "" {
		help = strings.Replace(help, "q: Back", "Esc: Clear filter | q: Back", 1)
	}
	return help
}
