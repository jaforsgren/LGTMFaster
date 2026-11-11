package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	pipelineFolderSeparator = `\`
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
	LastRunAt     time.Time
	LastViewedAt  time.Time
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
	lastViewed      map[string]time.Time
	sortMode        domain.PipelineSortMode
	previewRuns     map[string][]domain.PipelineRun
	previewPending  map[string]bool
	cursor          listCursor
	filter          listFilter
	split           splitView
	width           int
	height          int
}

func NewPipelineListView() *PipelineListViewModel {
	return &PipelineListViewModel{
		expanded:       make(map[string]bool),
		favorites:      make(map[string]bool),
		lastViewed:     make(map[string]time.Time),
		sortMode:       domain.PipelineSortFavorites,
		previewRuns:    make(map[string][]domain.PipelineRun),
		previewPending: make(map[string]bool),
		cursor:         newListCursor(),
		filter:         newListFilter("Filter by project, repository, or pipeline name..."),
	}
}

func (m *PipelineListViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.cursor.SetVisibleRows(height - listChromeRows)
}

func (m *PipelineListViewModel) SetPipelines(pipelines []domain.Pipeline) {
	m.sourcePipelines = append([]domain.Pipeline(nil), pipelines...)
	m.rebuild()
}

func (m *PipelineListViewModel) SetFavorites(favorites map[string]bool) {
	m.favorites = favorites
	m.rebuild()
}

func (m *PipelineListViewModel) SetLastViewed(lastViewed map[string]time.Time) {
	m.lastViewed = lastViewed
	m.rebuild()
}

func (m *PipelineListViewModel) SetSortMode(mode domain.PipelineSortMode) {
	m.sortMode = mode.OrDefault()
	m.rebuild()
}

func (m *PipelineListViewModel) SortMode() domain.PipelineSortMode {
	return m.sortMode
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

// SelectNode unfolds the groups around the node with key, moves the cursor to it and reports
// whether it exists.
func (m *PipelineListViewModel) SelectNode(key string) bool {
	ancestors, found := ancestorKeys(m.roots, key)
	if !found {
		return false
	}
	for _, ancestor := range ancestors {
		m.expanded[ancestor] = true
	}
	m.rebuildKeepingSelection(key)
	return true
}

// ancestorKeys returns the keys of the groups containing the node with key, outermost first.
func ancestorKeys(nodes []*PipelineNode, key string) ([]string, bool) {
	for _, node := range nodes {
		if node.Key == key {
			return nil, true
		}
		if inner, found := ancestorKeys(node.Children, key); found {
			return append([]string{node.Key}, inner...), true
		}
	}
	return nil, false
}

func (m *PipelineListViewModel) MoveUp() {
	m.cursor.MoveUp()
	m.split.ResetScroll()
}

func (m *PipelineListViewModel) MoveDown() {
	m.cursor.MoveDown(len(m.visibleNodes))
	m.split.ResetScroll()
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
	m.split.Toggle()
}

func (m *PipelineListViewModel) IsPreviewOpen() bool {
	return m.split.IsOpen()
}

func (m *PipelineListViewModel) FocusNextPane() {
	m.split.FocusNext(len(m.detailPanes()))
}

func (m *PipelineListViewModel) FocusPrevPane() {
	m.split.FocusPrev()
}

func (m *PipelineListViewModel) IsPaneFocused() bool {
	return m.split.IsDetailFocused()
}

func (m *PipelineListViewModel) ScrollPane(delta int) {
	m.split.ScrollFocused(m.detailPanes(), delta, m.width, m.height)
}

// NextPreviewRequest returns the selected pipeline when the open pane needs its runs fetched,
// and marks the request in flight so cursor movement does not refetch it.
func (m *PipelineListViewModel) NextPreviewRequest() *domain.Pipeline {
	pipeline := m.GetSelectedPipeline()
	if !m.split.IsOpen() || pipeline == nil {
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

	m.roots = buildPipelineTree(filtered, groupingFor(m.sortMode))
	m.annotate(m.roots)
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

type pipelineGroupLevel int

const (
	groupByProject pipelineGroupLevel = iota
	groupByRepository
)

func groupingFor(mode domain.PipelineSortMode) []pipelineGroupLevel {
	switch mode {
	case domain.PipelineSortFavorites, domain.PipelineSortProject:
		return []pipelineGroupLevel{groupByProject, groupByRepository}
	case domain.PipelineSortRepository:
		return []pipelineGroupLevel{groupByRepository}
	default:
		return nil
	}
}

func buildPipelineTree(pipelines []domain.Pipeline, levels []pipelineGroupLevel) []*PipelineNode {
	var roots []*PipelineNode
	groups := make(map[string]*PipelineNode)
	isFlat := len(levels) == 0

	for i := range pipelines {
		pipeline := &pipelines[i]
		var parent *PipelineNode

		for depth, level := range levels {
			key, kind, name := groupIdentity(pipeline, level, depth == 0)
			group, ok := groups[key]
			if !ok {
				group = &PipelineNode{Kind: kind, Key: key, Name: name, Depth: depth}
				if parent != nil {
					group.ParentKey = parent.Key
					parent.Children = append(parent.Children, group)
				} else {
					roots = append(roots, group)
				}
				groups[key] = group
			}
			parent = group
		}

		name := PipelineDisplayName(*pipeline)
		if isFlat {
			name = fmt.Sprintf("%s  (%s/%s)", name, pipeline.Project, pipeline.Repository)
		}
		leaf := &PipelineNode{
			Kind:     PipelineNodePipeline,
			Key:      pipeline.FavoriteKey(),
			Name:     name,
			Depth:    len(levels),
			Pipeline: pipeline,
		}
		if parent != nil {
			leaf.ParentKey = parent.Key
			parent.Children = append(parent.Children, leaf)
		} else {
			roots = append(roots, leaf)
		}
	}
	return roots
}

// groupIdentity names a repository by "project/repo" when it is a top-level group, since the
// project is otherwise not visible.
func groupIdentity(pipeline *domain.Pipeline, level pipelineGroupLevel, isRoot bool) (string, PipelineNodeKind, string) {
	if level == groupByProject {
		return domain.ProjectFavoriteKey(pipeline.PATID, pipeline.Project), PipelineNodeProject, pipeline.Project
	}
	name := pipeline.Repository
	if isRoot {
		name = pipeline.Project + "/" + pipeline.Repository
	}
	return domain.RepositoryFavoriteKey(pipeline.PATID, pipeline.Project, pipeline.Repository), PipelineNodeRepository, name
}

// annotate fills the sortable/displayed values bottom-up: leaves from their latest run and view
// history, groups as the aggregate of their children.
func (m *PipelineListViewModel) annotate(nodes []*PipelineNode) {
	for _, node := range nodes {
		node.LastViewedAt = m.lastViewed[node.Key]
		if !node.IsGroup() {
			node.Status = domain.PipelineStatusUnknown
			node.PipelineCount = 1
			if run := node.Pipeline.LatestRun; run != nil {
				node.Status = run.Status
				node.LastRunAt = run.QueuedAt
			}
			continue
		}

		m.annotate(node.Children)
		node.Status = aggregateStatus(node.Children)
		for _, child := range node.Children {
			node.PipelineCount += child.PipelineCount
			node.LastRunAt = laterOf(node.LastRunAt, child.LastRunAt)
			node.LastViewedAt = laterOf(node.LastViewedAt, child.LastViewedAt)
		}
	}
}

func laterOf(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func (m *PipelineListViewModel) sortTree(nodes []*PipelineNode) {
	less := m.lessFunc()
	sort.SliceStable(nodes, func(i, j int) bool {
		return less(nodes[i], nodes[j])
	})
	for _, node := range nodes {
		m.sortTree(node.Children)
	}
}

func (m *PipelineListViewModel) lessFunc() func(a, b *PipelineNode) bool {
	byName := func(a, b *PipelineNode) bool {
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	}
	byLastRun := func(a, b *PipelineNode) bool {
		if !a.LastRunAt.Equal(b.LastRunAt) {
			return a.LastRunAt.After(b.LastRunAt)
		}
		return byName(a, b)
	}

	switch m.sortMode {
	case domain.PipelineSortFavorites:
		return func(a, b *PipelineNode) bool {
			if m.favorites[a.Key] != m.favorites[b.Key] {
				return m.favorites[a.Key]
			}
			if !a.LastViewedAt.Equal(b.LastViewedAt) {
				return a.LastViewedAt.After(b.LastViewedAt)
			}
			return byName(a, b)
		}
	case domain.PipelineSortLastRun:
		return byLastRun
	case domain.PipelineSortStatus:
		return func(a, b *PipelineNode) bool {
			if statusRank(a.Status) != statusRank(b.Status) {
				return statusRank(a.Status) < statusRank(b.Status)
			}
			return byLastRun(a, b)
		}
	default:
		return byName
	}
}

func statusRank(status domain.PipelineStatus) int {
	for i, ranked := range groupStatusPriority {
		if ranked == status {
			return i
		}
	}
	return len(groupStatusPriority)
}

func aggregateStatus(children []*PipelineNode) domain.PipelineStatus {
	best := domain.PipelineStatusUnknown
	for _, child := range children {
		if statusRank(child.Status) < statusRank(best) {
			best = child.Status
		}
	}
	return best
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
	listWidth := m.split.ListWidth(m.width)
	if m.split.IsOpen() {
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

	b.WriteString(renderListTitle(pipelinesTitle))
	b.WriteString(listMutedStyle.Render("  sorted by " + PipelineSortModeLabel(m.sortMode)))
	b.WriteString("\n\n")

	if len(m.visibleNodes) == 0 {
		if m.filter.Text() != "" {
			b.WriteString(listMutedStyle.Render("No pipelines match the filter."))
		} else {
			b.WriteString(listMutedStyle.Render("No pipelines loaded."))
		}
		b.WriteString("\n")
	} else {
		b.WriteString(renderListRows(m.columns(), len(m.visibleNodes), m.cursor, func(i int) listRow {
			return m.nodeRow(m.visibleNodes[i])
		}))
	}

	if m.filter.IsActive() {
		b.WriteString("\n")
		b.WriteString(m.filter.View())
	}

	return m.split.Render(b.String(), m.detailPanes(), m.width, m.height)
}

func (m *PipelineListViewModel) nodeRow(node *PipelineNode) listRow {
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

	return listRow{
		icon:   renderPipelineStatusIcon(node.Status),
		values: []string{name, lastRun, branch, age},
		style:  listRowStyle,
	}
}

func (m *PipelineListViewModel) detailPanes() []detailPane {
	return []detailPane{m.detailPane()}
}

func (m *PipelineListViewModel) detailPane() detailPane {
	node := m.GetSelectedNode()
	if node == nil {
		return detailPane{title: "Details", lines: []string{listMutedStyle.Render("Nothing selected.")}}
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
		return detailPane{title: "Pipelines in " + node.Name, lines: lines}
	case PipelineNodeRepository:
		var lines []string
		for _, pipeline := range node.Children {
			lines = append(lines, renderDetailRow(pipeline.Status, pipeline.Name+latestRunSuffix(pipeline.Pipeline)))
		}
		return detailPane{title: "Pipelines in " + node.Name, lines: lines}
	default:
		return detailPane{title: "Runs of " + node.Name, lines: m.previewRunLines(node.Pipeline)}
	}
}

func (m *PipelineListViewModel) previewRunLines(pipeline *domain.Pipeline) []string {
	runs, loaded := m.previewRuns[pipeline.FavoriteKey()]
	if !loaded {
		return []string{listMutedStyle.Render("Loading runs...")}
	}
	if len(runs) == 0 {
		return []string{listMutedStyle.Render("This pipeline has no runs yet.")}
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

var pipelineSortModeLabels = map[domain.PipelineSortMode]string{
	domain.PipelineSortFavorites:  "favorites & recently viewed",
	domain.PipelineSortProject:    "project",
	domain.PipelineSortRepository: "repository",
	domain.PipelineSortPipeline:   "pipeline name",
	domain.PipelineSortLastRun:    "last run",
	domain.PipelineSortStatus:     "status",
}

func PipelineSortModeLabel(mode domain.PipelineSortMode) string {
	return pipelineSortModeLabels[mode.OrDefault()]
}
