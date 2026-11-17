package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

// MergeRunsAction is what a key press in the merge runs list asks the app to do.
type MergeRunsAction int

const (
	MergeRunsActionNone MergeRunsAction = iota
	MergeRunsActionOpen
	MergeRunsActionOpenBrowser
	MergeRunsActionCancel
)

const (
	mergeRunsTitleFormat = "Runs of #%d %s, merged as %s"
	mergeRunsHelp        = "↑/↓: move | enter: open run | ctrl+o: open in browser | esc: close"
	loadingMergeRunsList = "Looking for the runs that built the merge commit..."
	noMergeRunsList      = "No pipeline run built the merge commit on the target branch."
	mergeRunsInset       = 4
	// mergeRunsChromeRows is the box's border and padding, title, header and help.
	mergeRunsChromeRows = 10
	shortCommitLength   = 8
)

var mergeRunsBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(mainWindowBorderColor).Padding(1, 2)

// MergeRunsViewModel lists the pipeline runs that built a merged pull request's merge commit.
type MergeRunsViewModel struct {
	pr     domain.PullRequest
	runs   []domain.PipelineRun
	loaded bool
	cursor listCursor
	active bool
	width  int
	height int
}

func NewMergeRunsView() *MergeRunsViewModel {
	return &MergeRunsViewModel{cursor: newListCursor()}
}

func (m *MergeRunsViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.cursor.SetVisibleRows(height - mergeRunsChromeRows)
}

// Activate opens the list for the pull request, loading until SetRuns.
func (m *MergeRunsViewModel) Activate(pr domain.PullRequest) {
	m.pr = pr
	m.runs = nil
	m.loaded = false
	m.cursor.Reset()
	m.active = true
}

func (m *MergeRunsViewModel) Deactivate() {
	m.active = false
}

func (m *MergeRunsViewModel) IsActive() bool {
	return m.active
}

// PRKey is the pull request whose runs are listed.
func (m *MergeRunsViewModel) PRKey() string {
	return m.pr.Key()
}

func (m *MergeRunsViewModel) SetRuns(runs []domain.PipelineRun) {
	m.runs = runs
	m.loaded = true
	m.cursor.Clamp(len(m.runs))
}

func (m *MergeRunsViewModel) Selected() *domain.PipelineRun {
	if index := m.cursor.Index(); index >= 0 && index < len(m.runs) {
		return &m.runs[index]
	}
	return nil
}

func (m *MergeRunsViewModel) HandleKey(msg tea.KeyMsg) MergeRunsAction {
	switch msg.String() {
	case "enter":
		if m.Selected() != nil {
			return MergeRunsActionOpen
		}
	case "ctrl+o":
		if m.Selected() != nil {
			return MergeRunsActionOpenBrowser
		}
	case "esc", "q":
		return MergeRunsActionCancel
	case "down", "j", "ctrl+n":
		m.cursor.MoveDown(len(m.runs))
	case "up", "k", "ctrl+p":
		m.cursor.MoveUp()
	}
	return MergeRunsActionNone
}

func (m *MergeRunsViewModel) View() string {
	if !m.active {
		return ""
	}
	innerWidth := max(m.width-mergeRunsInset, minPaneSize)

	var b strings.Builder
	commit := m.pr.MergeCommitID[:min(len(m.pr.MergeCommitID), shortCommitLength)]
	b.WriteString(listTitleStyle.Render(truncateString(fmt.Sprintf(mergeRunsTitleFormat, m.pr.Number, m.pr.Title, commit), innerWidth)))
	b.WriteString("\n\n")

	switch {
	case !m.loaded:
		b.WriteString(listMutedStyle.Render(loadingMergeRunsList))
	case len(m.runs) == 0:
		b.WriteString(listMutedStyle.Render(noMergeRunsList))
	default:
		b.WriteString(renderListRows(pipelineRunColumns(innerWidth), len(m.runs), m.cursor, func(i int) listRow {
			return pipelineRunRow(m.runs[i])
		}))
	}
	b.WriteString("\n")
	b.WriteString(listMutedStyle.Render(mergeRunsHelp))
	return mergeRunsBoxStyle.Width(innerWidth).Render(b.String())
}
