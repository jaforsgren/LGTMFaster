package views

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

// WorkItemPickerAction is what a key press in the work item picker asks the app to do.
type WorkItemPickerAction int

const (
	WorkItemPickerActionNone WorkItemPickerAction = iota
	WorkItemPickerActionLink
	WorkItemPickerActionCancel
)

const (
	workItemPickerTitleFormat = "Link a work item to #%d %s"
	workItemPickerHelp        = "type to filter | ↑/↓: move | enter: link | esc: cancel"
	workItemPickerFilterHint  = "number to link any work item, or id, type, title or state"
	loadingAssignedText       = "Loading your open bugs, tasks and user stories..."
	noAssignedText            = "Nothing to link: no open bugs, tasks or user stories are assigned to you."
	noAssignedMatchText       = "No work items match the filter."
	alreadyLinkedFormat       = "#%d is already linked."
	linkByNumberText          = "link this work item by its number"
	linkByNumberIcon          = "+"
	workItemIcon              = "●"
	workItemPickerInset       = 4
	// workItemPickerChromeRows is the box's border and padding, title, filter, header and help.
	workItemPickerChromeRows = 12
	workItemIDWidth          = 8
	workItemTypeWidth        = 11
	workItemStateWidth       = 12
	minWorkItemTitleWidth    = 20
	workItemColumnSpacing    = 3
)

var workItemPickerBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(mainWindowBorderColor).Padding(1, 2)

// WorkItemPickerViewModel picks one of the user's work items to link to a pull request.
type WorkItemPickerViewModel struct {
	pr        domain.PullRequest
	workItems []domain.WorkItem
	linked    map[int]bool
	visible   []domain.WorkItem
	// typed is the work item number the filter holds, offered first even when it is not assigned
	// to the user; 0 when the filter is not a number.
	typed  int
	loaded bool
	filter listFilter
	cursor listCursor
	active bool
	width  int
	height int
}

func NewWorkItemPickerView() *WorkItemPickerViewModel {
	return &WorkItemPickerViewModel{filter: newListFilter(workItemPickerFilterHint), cursor: newListCursor()}
}

func (m *WorkItemPickerViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.cursor.SetVisibleRows(height - workItemPickerChromeRows)
}

// Activate opens the picker for the pull request and the work items already linked to it, empty
// until SetWorkItems.
func (m *WorkItemPickerViewModel) Activate(pr domain.PullRequest, linked map[int]bool) {
	m.pr = pr
	m.workItems = nil
	m.linked = linked
	m.visible = nil
	m.typed = 0
	m.loaded = false
	m.filter.Clear()
	m.filter.Activate()
	m.cursor.Reset()
	m.active = true
}

func (m *WorkItemPickerViewModel) Deactivate() {
	m.active = false
	m.filter.Deactivate()
}

func (m *WorkItemPickerViewModel) IsActive() bool {
	return m.active
}

// PRKey is the pull request the picker links to.
func (m *WorkItemPickerViewModel) PRKey() string {
	return m.pr.Key()
}

// SetWorkItems lists the work items that are not linked yet.
func (m *WorkItemPickerViewModel) SetWorkItems(workItems []domain.WorkItem) {
	m.workItems = make([]domain.WorkItem, 0, len(workItems))
	for _, workItem := range workItems {
		if !m.linked[workItem.ID] {
			m.workItems = append(m.workItems, workItem)
		}
	}
	m.loaded = true
	m.rebuild()
}

func (m *WorkItemPickerViewModel) Selected() *domain.WorkItem {
	if index := m.cursor.Index(); index < len(m.visible) {
		return &m.visible[index]
	}
	return nil
}

func (m *WorkItemPickerViewModel) HandleKey(msg tea.KeyMsg) (WorkItemPickerAction, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.Selected() == nil {
			return WorkItemPickerActionNone, nil
		}
		return WorkItemPickerActionLink, nil
	case "esc":
		return WorkItemPickerActionCancel, nil
	case "down", "ctrl+n":
		m.cursor.MoveDown(len(m.visible))
		return WorkItemPickerActionNone, nil
	case "up", "ctrl+p":
		m.cursor.MoveUp()
		return WorkItemPickerActionNone, nil
	}
	cmd := m.filter.Update(msg)
	m.rebuild()
	m.cursor.Reset()
	return WorkItemPickerActionNone, cmd
}

func (m *WorkItemPickerViewModel) rebuild() {
	m.typed = typedWorkItemID(m.filter.Text())
	m.visible = m.visible[:0]
	if m.typed != 0 && !m.linked[m.typed] {
		m.visible = append(m.visible, m.typedWorkItem())
	}
	for _, workItem := range m.workItems {
		if workItem.ID != m.typed && m.filter.Matches(strconv.Itoa(workItem.ID), workItem.Type, workItem.Title, workItem.State) {
			m.visible = append(m.visible, workItem)
		}
	}
	m.cursor.Clamp(len(m.visible))
}

// typedWorkItemID is the one work item number the text holds, or 0.
func typedWorkItemID(text string) int {
	ids, err := domain.ParseWorkItemIDs(text)
	if err != nil || len(ids) != 1 {
		return 0
	}
	return ids[0]
}

// typedWorkItem is the assigned work item with the typed number, or one known only by it.
func (m *WorkItemPickerViewModel) typedWorkItem() domain.WorkItem {
	for _, workItem := range m.workItems {
		if workItem.ID == m.typed {
			return workItem
		}
	}
	return domain.WorkItem{ID: m.typed}
}

func (m *WorkItemPickerViewModel) View() string {
	if !m.active {
		return ""
	}
	innerWidth := max(m.width-workItemPickerInset, minPaneSize)

	var b strings.Builder
	b.WriteString(listTitleStyle.Render(truncateString(fmt.Sprintf(workItemPickerTitleFormat, m.pr.Number, m.pr.Title), innerWidth)))
	b.WriteString("\n\n")
	b.WriteString(m.filter.View())
	b.WriteString("\n\n")

	switch {
	case m.typed != 0 && m.linked[m.typed]:
		b.WriteString(listMutedStyle.Render(fmt.Sprintf(alreadyLinkedFormat, m.typed)))
	case len(m.visible) > 0:
		b.WriteString(m.renderRows(innerWidth))
	case !m.loaded:
		b.WriteString(listMutedStyle.Render(loadingAssignedText))
	case len(m.workItems) == 0:
		b.WriteString(listMutedStyle.Render(noAssignedText))
	default:
		b.WriteString(listMutedStyle.Render(noAssignedMatchText))
	}
	b.WriteString("\n")
	b.WriteString(listMutedStyle.Render(workItemPickerHelp))
	return workItemPickerBoxStyle.Width(innerWidth).Render(b.String())
}

func (m *WorkItemPickerViewModel) renderRows(innerWidth int) string {
	columns := []listColumn{
		{"#", workItemIDWidth},
		{"Type", workItemTypeWidth},
		{"Title", flexColumnWidth(innerWidth, workItemIDWidth+workItemTypeWidth+workItemStateWidth+workItemColumnSpacing, minWorkItemTitleWidth)},
		{"State", workItemStateWidth},
	}
	return renderListRows(columns, len(m.visible), m.cursor, func(i int) listRow {
		workItem := m.visible[i]
		if isKnownOnlyByNumber(workItem) {
			return listRow{
				icon:   renderListIcon(linkByNumberIcon, workItemTypeColor(workItem.Type)),
				values: []string{strconv.Itoa(workItem.ID), "", listMutedStyle.Render(linkByNumberText), ""},
				style:  listRowStyle,
			}
		}
		return listRow{
			icon:   renderListIcon(workItemIcon, workItemTypeColor(workItem.Type)),
			values: []string{strconv.Itoa(workItem.ID), workItem.Type, workItem.Title, workItem.State},
			style:  listRowStyle,
		}
	})
}

func isKnownOnlyByNumber(workItem domain.WorkItem) bool {
	return workItem == domain.WorkItem{ID: workItem.ID}
}
