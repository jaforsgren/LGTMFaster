package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

type workItemsState int

const (
	workItemsNotLoaded workItemsState = iota
	workItemsLoading
	workItemsLoaded
)

const (
	workItemsPaneTitle       = "Work items"
	workItemsPaneTitleFormat = "Work items (%d)"
	loadingWorkItemsText     = "Loading work items..."
	noWorkItemsText          = "No linked work items. Press l to link one."
	noWorkItemDescription    = "No description."
	workItemHeaderFormat     = "#%d %s"
	workItemStateSeparator   = " · "
	defaultWorkItemColor     = lipgloss.Color("7")
)

// workItemTypeColors tells Azure DevOps work item types and GitHub issues apart at a glance.
var workItemTypeColors = map[string]lipgloss.Color{
	"Bug":        lipgloss.Color("#EF4444"),
	"Task":       lipgloss.Color("#FBBF24"),
	"User Story": lipgloss.Color("#3B82F6"),
	"Issue":      lipgloss.Color("#10B981"),
}

func workItemTypeColor(workItemType string) lipgloss.Color {
	if color, known := workItemTypeColors[workItemType]; known {
		return color
	}
	return defaultWorkItemColor
}

func renderWorkItemHeader(workItem domain.WorkItem) string {
	header := lipgloss.NewStyle().Foreground(workItemTypeColor(workItem.Type)).Bold(true).
		Render(fmt.Sprintf(workItemHeaderFormat, workItem.ID, workItem.Type))
	if workItem.State == "" {
		return header
	}
	return header + listMutedStyle.Render(workItemStateSeparator+workItem.State)
}

// The work items pane sits beside the description only; the diff and commits keep the whole window.
func (m *PRInspectViewModel) showsWorkItemsPane() bool {
	return m.mode == PRInspectModeDescription
}

func (m *PRInspectViewModel) TogglePreview() {
	m.split.Toggle()
	m.updateViewport()
}

// ClosePreview hides the pane, which every pull request opens without.
func (m *PRInspectViewModel) ClosePreview() {
	m.split.Close()
	m.updateViewport()
}

func (m *PRInspectViewModel) IsPreviewOpen() bool {
	return m.split.IsOpen()
}

func (m *PRInspectViewModel) FocusNextPane() {
	m.split.FocusNext(len(m.workItemPanes()))
}

func (m *PRInspectViewModel) FocusPrevPane() {
	m.split.FocusPrev()
}

func (m *PRInspectViewModel) IsPaneFocused() bool {
	return m.showsWorkItemsPane() && m.split.IsDetailFocused()
}

func (m *PRInspectViewModel) ScrollPane(delta int) {
	m.split.ScrollFocused(m.workItemPanes(), delta, m.width, m.height)
}

// PendingWorkItemsRequest returns the pull request when the open pane needs its work items, and
// marks the request in flight so it is made once.
func (m *PRInspectViewModel) PendingWorkItemsRequest() *domain.PullRequest {
	if m.pr == nil || !m.split.IsOpen() || m.workItemsState != workItemsNotLoaded {
		return nil
	}
	m.workItemsState = workItemsLoading
	return m.pr
}

func (m *PRInspectViewModel) SetWorkItems(prKey string, workItems []domain.WorkItem) {
	if m.pr == nil || m.pr.Key() != prKey {
		return
	}
	m.workItems = workItems
	m.workItemsState = workItemsLoaded
	m.split.ResetScroll()
}

// ReloadWorkItems forgets the work items so they are fetched again, after a failure or a new link.
func (m *PRInspectViewModel) ReloadWorkItems() {
	m.workItems = nil
	m.workItemsState = workItemsNotLoaded
}

func (m *PRInspectViewModel) LinkedWorkItemIDs() map[int]bool {
	ids := make(map[int]bool, len(m.workItems))
	for _, workItem := range m.workItems {
		ids[workItem.ID] = true
	}
	return ids
}

func (m *PRInspectViewModel) resetWorkItems() {
	m.ReloadWorkItems()
	m.split.ResetScroll()
}

// contentWidth is the width the viewport lays its content out in, narrower beside the pane.
func (m *PRInspectViewModel) contentWidth() int {
	if m.showsWorkItemsPane() {
		return m.split.ListWidth(m.width)
	}
	return m.width
}

func (m *PRInspectViewModel) workItemPanes() []detailPane {
	switch {
	case m.workItemsState != workItemsLoaded:
		return []detailPane{{title: workItemsPaneTitle, lines: []string{listMutedStyle.Render(loadingWorkItemsText)}}}
	case len(m.workItems) == 0:
		return []detailPane{{title: workItemsPaneTitle, lines: []string{listMutedStyle.Render(noWorkItemsText)}}}
	}
	_, width := splitWidths(m.width)
	var lines []string
	for i, workItem := range m.workItems {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, renderWorkItemHeader(workItem))
		lines = append(lines, wrapLines(detailValueStyle.Bold(true), workItem.Title, width)...)
		description := strings.TrimSpace(workItem.Description)
		if description == "" {
			lines = append(lines, listMutedStyle.Render(noWorkItemDescription))
			continue
		}
		lines = append(lines, wrapLines(listRowStyle, description, width)...)
	}
	return []detailPane{{title: fmt.Sprintf(workItemsPaneTitleFormat, len(m.workItems)), lines: lines}}
}

// wrapLines wraps text to width so a pane scrolls by visual line.
func wrapLines(style lipgloss.Style, text string, width int) []string {
	return strings.Split(style.Width(width).Render(text), "\n")
}
