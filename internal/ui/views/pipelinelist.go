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
	// favoriteSectionPrefix keeps the IDs of the favorites section's copies apart from the main
	// tree's, so folding or selecting one copy leaves the other alone.
	favoriteSectionPrefix = "favorites:"
	runIDSeparator        = "/"
	maxExpandedRuns       = 10
	loadingRunsText       = "Loading runs..."
	noRunsText            = "This pipeline has no runs yet."
	noFavoritesText       = "No favorites yet. Press f on a pipeline, repository or project to add one."
	favoritesOnlyHint     = "  favorites only"
	// pipelineListChromeRows leaves room for the divider between the favorites and the rest.
	pipelineListChromeRows = listChromeRows + listDividerRows
)

type PipelineNodeKind int

const (
	PipelineNodeProject PipelineNodeKind = iota
	PipelineNodeRepository
	PipelineNodePipeline
	PipelineNodeRun
	PipelineNodePlaceholder
)

type PipelineNode struct {
	Kind PipelineNodeKind
	// Key is the domain key used for favorites and view history; ID is unique per visible row.
	Key           string
	ID            string
	ParentID      string
	Name          string
	Depth         int
	Status        domain.PipelineStatus
	PipelineCount int
	LastRunAt     time.Time
	LastViewedAt  time.Time
	Pipeline      *domain.Pipeline
	Run           *domain.PipelineRun
	Children      []*PipelineNode
}

func (n *PipelineNode) IsGroup() bool {
	return n.Kind == PipelineNodeProject || n.Kind == PipelineNodeRepository
}

// IsExpandable reports whether the node folds: groups over their children, pipelines over runs.
func (n *PipelineNode) IsExpandable() bool {
	return n.IsGroup() || n.Kind == PipelineNodePipeline
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
	favoriteRoots   []*PipelineNode
	visibleNodes    []*PipelineNode
	// mainSectionStart is the index of the first main-tree row, which has the divider above it.
	mainSectionStart int
	favoritesOnly    bool
	expanded         map[string]bool
	favorites        map[string]bool
	lastViewed       map[string]time.Time
	sortMode         domain.PipelineSortMode
	previewRuns      map[string][]domain.PipelineRun
	previewPending   map[string]bool
	cursor           listCursor
	filter           listFilter
	split            splitView
	width            int
	height           int
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

// GetSelectedPipeline is the selected pipeline, or the pipeline of the selected run.
func (m *PipelineListViewModel) GetSelectedPipeline() *domain.Pipeline {
	node := m.GetSelectedNode()
	if node == nil {
		return nil
	}
	return node.Pipeline
}

func (m *PipelineListViewModel) GetSelectedRun() *domain.PipelineRun {
	node := m.GetSelectedNode()
	if node == nil {
		return nil
	}
	return node.Run
}

func (m *PipelineListViewModel) ToggleFavoritesOnly() {
	m.favoritesOnly = !m.favoritesOnly
	m.rebuild()
}

func (m *PipelineListViewModel) IsFavoritesOnly() bool {
	return m.favoritesOnly
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
			return append([]string{node.ID}, inner...), true
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

// ToggleSelected folds/unfolds a group or a pipeline's runs; on a run it folds the pipeline and
// selects it.
func (m *PipelineListViewModel) ToggleSelected() {
	node := m.GetSelectedNode()
	if node == nil {
		return
	}
	if node.IsExpandable() {
		m.expanded[node.ID] = !m.expanded[node.ID]
		m.rebuildKeepingSelection(node.ID)
		return
	}
	m.expanded[node.ParentID] = false
	m.rebuildKeepingSelection(node.ParentID)
}

// ToggleAll expands every group unless all are already expanded, in which case it folds them all.
func (m *PipelineListViewModel) ToggleAll() {
	var selectedKey string
	if node := m.GetSelectedNode(); node != nil {
		selectedKey = node.ID
	}

	groupKeys := collectGroupIDs(append(m.favoriteRoots, m.roots...))
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
		selectedKey = rootIDOf(node, append(m.favoriteRoots, m.roots...))
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

// ShownRunPipelines are the pipelines whose runs are on screen: the selected one while the details
// pane is open, and every unfolded one.
func (m *PipelineListViewModel) ShownRunPipelines() []domain.Pipeline {
	var shown []domain.Pipeline
	if pipeline := m.GetSelectedPipeline(); m.split.IsOpen() && pipeline != nil {
		shown = append(shown, *pipeline)
	}
	for _, node := range m.visibleNodes {
		if node.Kind == PipelineNodePipeline && m.expanded[node.ID] {
			shown = append(shown, *node.Pipeline)
		}
	}
	return shown
}

// PendingRunRequests returns the shown pipelines whose runs are not fetched yet, and marks them in
// flight so cursor movement does not refetch them.
func (m *PipelineListViewModel) PendingRunRequests() []domain.Pipeline {
	var requests []domain.Pipeline
	for _, pipeline := range m.ShownRunPipelines() {
		key := pipeline.FavoriteKey()
		if _, loaded := m.previewRuns[key]; loaded || m.previewPending[key] {
			continue
		}
		m.previewPending[key] = true
		requests = append(requests, pipeline)
	}
	return requests
}

func (m *PipelineListViewModel) SetPreviewRuns(pipelineKey string, runs []domain.PipelineRun) {
	delete(m.previewPending, pipelineKey)
	m.previewRuns[pipelineKey] = runs
	m.rebuild()
}

// PreviewRunsFailed forgets the in-flight request, so the runs are fetched again next time they
// are shown instead of loading forever.
func (m *PipelineListViewModel) PreviewRunsFailed(pipelineKey string) {
	delete(m.previewPending, pipelineKey)
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
	var selectedID string
	if node := m.GetSelectedNode(); node != nil {
		selectedID = node.ID
	}
	m.rebuildKeepingSelection(selectedID)
}

func (m *PipelineListViewModel) rebuildKeepingSelection(selectedID string) {
	var filtered []domain.Pipeline
	for _, pipeline := range m.sourcePipelines {
		if m.filter.Matches(pipeline.Project, pipeline.Repository, PipelineDisplayName(pipeline)) {
			filtered = append(filtered, pipeline)
		}
	}

	m.roots = buildPipelineTree(filtered, groupingFor(m.sortMode))
	m.annotate(m.roots)
	m.sortTree(m.roots)
	m.favoriteRoots = m.favoriteSection(m.roots)
	m.sortTree(m.favoriteRoots)

	// While filtering every group is unfolded so matches are never hidden inside a folded group.
	showAll := m.filter.Text() != ""
	m.visibleNodes = m.visibleNodes[:0]
	m.flatten(m.favoriteRoots, showAll)
	m.mainSectionStart = len(m.visibleNodes)
	if !m.favoritesOnly {
		m.flatten(m.roots, showAll)
	}

	for i, node := range m.visibleNodes {
		if node.ID == selectedID {
			m.cursor.SetIndex(i, len(m.visibleNodes))
			return
		}
	}
	m.cursor.Clamp(len(m.visibleNodes))
}

func (m *PipelineListViewModel) flatten(nodes []*PipelineNode, showAll bool) {
	for _, node := range nodes {
		m.visibleNodes = append(m.visibleNodes, node)
		switch {
		case node.IsGroup() && (showAll || m.expanded[node.ID]):
			m.flatten(node.Children, showAll)
		case node.Kind == PipelineNodePipeline && m.expanded[node.ID]:
			m.visibleNodes = append(m.visibleNodes, m.runNodes(node)...)
		}
	}
}

// runNodes are the rows under an unfolded pipeline: its latest runs, or a row saying why there
// are none to show yet.
func (m *PipelineListViewModel) runNodes(pipelineNode *PipelineNode) []*PipelineNode {
	child := func(id, key, name string) *PipelineNode {
		return &PipelineNode{
			ID:       pipelineNode.ID + runIDSeparator + id,
			Key:      key,
			ParentID: pipelineNode.ID,
			Name:     name,
			Depth:    pipelineNode.Depth + 1,
			Pipeline: pipelineNode.Pipeline,
			Kind:     PipelineNodePlaceholder,
		}
	}

	runs, loaded := m.previewRuns[pipelineNode.Key]
	if !loaded {
		return []*PipelineNode{child(loadingRunsText, "", loadingRunsText)}
	}
	if len(runs) == 0 {
		return []*PipelineNode{child(noRunsText, "", noRunsText)}
	}

	nodes := make([]*PipelineNode, 0, min(len(runs), maxExpandedRuns))
	for i := range runs[:min(len(runs), maxExpandedRuns)] {
		run := &runs[i]
		node := child(run.FavoriteKey(), run.FavoriteKey(), valueOrPlaceholder(run.Number))
		node.Kind = PipelineNodeRun
		node.Status = run.Status
		node.Run = run
		nodes = append(nodes, node)
	}
	return nodes
}

// favoriteSection copies every favorite node, with its subtree, to the top level of the favorites
// section. The copies keep the annotations of the main tree.
func (m *PipelineListViewModel) favoriteSection(roots []*PipelineNode) []*PipelineNode {
	var section []*PipelineNode
	var collect func(nodes []*PipelineNode)
	collect = func(nodes []*PipelineNode) {
		for _, node := range nodes {
			if m.favorites[node.Key] {
				section = append(section, copyToFavoriteSection(node, "", 0))
			}
			collect(node.Children)
		}
	}
	collect(roots)
	return section
}

// favoriteSectionName names a node at the top of the favorites section, where the groups that
// would otherwise say which project and repository it belongs to are not shown.
func favoriteSectionName(node *PipelineNode) string {
	switch node.Kind {
	case PipelineNodePipeline:
		return fmt.Sprintf("%s  (%s/%s)", PipelineDisplayName(*node.Pipeline), node.Pipeline.Project, node.Pipeline.Repository)
	case PipelineNodeRepository:
		if pipeline := firstPipeline(node); pipeline != nil {
			return pipeline.Project + "/" + pipeline.Repository
		}
	}
	return node.Name
}

func firstPipeline(node *PipelineNode) *domain.Pipeline {
	if node.Pipeline != nil {
		return node.Pipeline
	}
	for _, child := range node.Children {
		if pipeline := firstPipeline(child); pipeline != nil {
			return pipeline
		}
	}
	return nil
}

func copyToFavoriteSection(node *PipelineNode, parentID string, depth int) *PipelineNode {
	clone := *node
	clone.ID = favoriteSectionPrefix + node.ID
	clone.ParentID = parentID
	clone.Depth = depth
	if depth == 0 {
		clone.Name = favoriteSectionName(node)
	}
	clone.Children = make([]*PipelineNode, len(node.Children))
	for i, child := range node.Children {
		clone.Children[i] = copyToFavoriteSection(child, clone.ID, depth+1)
	}
	return &clone
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
				group = &PipelineNode{Kind: kind, Key: key, ID: key, Name: name, Depth: depth}
				if parent != nil {
					group.ParentID = parent.ID
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
			ID:       pipeline.FavoriteKey(),
			Name:     name,
			Depth:    len(levels),
			Pipeline: pipeline,
		}
		if parent != nil {
			leaf.ParentID = parent.ID
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

func collectGroupIDs(nodes []*PipelineNode) []string {
	var ids []string
	for _, node := range nodes {
		if node.IsGroup() {
			ids = append(ids, node.ID)
			ids = append(ids, collectGroupIDs(node.Children)...)
		}
	}
	return ids
}

// rootIDOf finds the top-level row a node sits under; run rows are found through their pipeline.
func rootIDOf(node *PipelineNode, roots []*PipelineNode) string {
	id := node.ID
	if node.ParentID != "" && !node.IsExpandable() {
		id = node.ParentID
	}
	for _, root := range roots {
		if root.ID == id || containsID(root.Children, id) {
			return root.ID
		}
	}
	return id
}

func containsID(nodes []*PipelineNode, id string) bool {
	for _, node := range nodes {
		if node.ID == id || containsID(node.Children, id) {
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
	const columnSpacing = 5
	fixed := lastRunWidth + runStageColumnWidth + branchWidth + ageWidth + runCommitColumnWidth + columnSpacing
	return []listColumn{
		{"Name", flexColumnWidth(listWidth, fixed, minNameWidth)},
		{"Last run", lastRunWidth},
		runStageColumn,
		{"Branch", branchWidth},
		{"Age", ageWidth},
		runCommitColumn,
	}
}

func (m *PipelineListViewModel) View() string {
	var b strings.Builder

	b.WriteString(renderListTitle(pipelinesTitle))
	b.WriteString(listMutedStyle.Render("  sorted by " + PipelineSortModeLabel(m.sortMode)))
	if m.favoritesOnly {
		b.WriteString(listMutedStyle.Render(favoritesOnlyHint))
	}
	b.WriteString("\n\n")

	if len(m.visibleNodes) == 0 {
		if m.favoritesOnly && m.filter.Text() == "" {
			b.WriteString(listMutedStyle.Render(noFavoritesText))
		} else if m.filter.Text() != "" {
			b.WriteString(listMutedStyle.Render("No pipelines match the filter."))
		} else {
			b.WriteString(listMutedStyle.Render("No pipelines loaded."))
		}
		b.WriteString("\n")
	} else {
		b.WriteString(renderListRows(m.columns(), len(m.visibleNodes), m.cursor, func(i int) listRow {
			row := m.nodeRow(m.visibleNodes[i])
			row.dividerAbove = i == m.mainSectionStart && i > 0
			return row
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
	if node.IsExpandable() {
		marker = collapsedMarker
		if m.expanded[node.ID] || (node.IsGroup() && m.filter.Text() != "") {
			marker = expandedMarker
		}
	}

	indent := strings.Repeat(treeIndent, node.Depth)
	if node.Kind == PipelineNodePlaceholder {
		return listRow{
			icon:   renderListIcon("", ""),
			values: []string{indent + leafMarker + node.Name, "", "", "", "", ""},
			style:  listMutedStyle,
		}
	}

	name := indent + marker + favoritePrefix(m.favorites[node.Key]) + node.Name
	values := []string{name, emptyCellPlaceholder, emptyCellPlaceholder, emptyCellPlaceholder, emptyCellPlaceholder, emptyCellPlaceholder}
	switch {
	case node.IsGroup():
		values[0] = fmt.Sprintf("%s (%d)", name, node.PipelineCount)
	case node.Run != nil:
		values = append(values[:1], runCells(*node.Run, PipelineStatusLabel(node.Run.Status))...)
	case node.Pipeline.LatestRun != nil:
		latest := *node.Pipeline.LatestRun
		values = append(values[:1], runCells(latest, valueOrPlaceholder(latest.Number))...)
	}

	return listRow{
		icon:   renderPipelineStatusIcon(node.Status),
		values: values,
		style:  listRowStyle,
	}
}

// runCells fills the columns after the name for a run row or a pipeline's latest run.
func runCells(run domain.PipelineRun, lastRun string) []string {
	return []string{lastRun, runStageLabel(run), valueOrPlaceholder(run.SourceBranch), formatOptionalAge(run.QueuedAt), runCommitLabel(run)}
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
	case PipelineNodeRun:
		return detailPane{title: "Run " + node.Name, lines: runDetailLines(*node.Run)}
	default:
		return detailPane{title: "Runs of " + PipelineDisplayName(*node.Pipeline), lines: m.previewRunLines(node.Pipeline)}
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
		lines = append(lines, renderDetailRow(run.Status, fmt.Sprintf("%s  %s  %s  %s",
			valueOrPlaceholder(run.Number), valueOrPlaceholder(run.SourceBranch), formatOptionalAge(run.QueuedAt),
			truncateString(run.CommitMessage, runCommitColumnWidth))))
	}
	return lines
}

func runDetailLines(run domain.PipelineRun) []string {
	details := []struct {
		label string
		value string
	}{
		{"Status", renderPipelineStatusLabel(run.Status, stageStatusWidth)},
		{"Stage", runStageLabel(run)},
		{"Branch", valueOrPlaceholder(run.SourceBranch)},
		{"Commit", truncateString(runCommitLabel(run), runCommitDetailWidth)},
		{"Requested by", valueOrPlaceholder(run.RequestedBy.Username)},
		{"Queued", formatOptionalAge(run.QueuedAt)},
		{"Duration", formatElapsed(run.StartedAt, run.FinishedAt)},
	}
	lines := make([]string, 0, len(details))
	for _, detail := range details {
		lines = append(lines, renderDetailLabel(detail.label, runDetailLabelWidth)+detailValueStyle.Render(detail.value))
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
