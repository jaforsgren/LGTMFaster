package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/ui/views"
)

const (
	linkingWorkItemMessage = "Linking #%d %s..."
	linkedWorkItemMessage  = "Linked #%d %s"
	noProviderMessage      = "No provider available for this pull request"
)

type WorkItemsLoadedMsg struct {
	prKey     string
	workItems []domain.WorkItem
	err       error
}

type AssignedWorkItemsLoadedMsg struct {
	prKey     string
	workItems []domain.WorkItem
	err       error
}

type WorkItemLinkedMsg struct {
	prKey    string
	workItem domain.WorkItem
	err      error
}

// loadInspectWorkItems fetches the linked work items once the open pane needs them.
func (m Model) loadInspectWorkItems() tea.Cmd {
	pr := m.prInspect.PendingWorkItemsRequest()
	if pr == nil {
		return nil
	}
	provider := m.getProviderForPR(*pr)
	ctx, inspected := m.ctx, *pr
	return func() tea.Msg {
		if provider == nil {
			return WorkItemsLoadedMsg{prKey: inspected.Key(), err: fmt.Errorf("%s", noProviderMessage)}
		}
		workItems, err := provider.ListLinkedWorkItems(ctx, prIdentifier(inspected))
		return WorkItemsLoadedMsg{prKey: inspected.Key(), workItems: workItems, err: err}
	}
}

func (m Model) handleWorkItemsLoaded(msg WorkItemsLoadedMsg) (Model, tea.Cmd) {
	if pr := m.prInspect.GetPR(); pr == nil || pr.Key() != msg.prKey {
		return m, nil
	}
	if msg.err != nil {
		m.prInspect.ReloadWorkItems()
		m.statusBar.SetMessage(fmt.Sprintf("Failed to load work items: %v", msg.err), true)
		return m, nil
	}
	m.prInspect.SetWorkItems(msg.prKey, msg.workItems)
	return m, nil
}

// handleLinkWorkItemKey opens the picker over the user's open work items.
func handleLinkWorkItemKey(m Model) (Model, tea.Cmd) {
	pr := m.prInspect.GetPR()
	if pr == nil {
		return m, nil
	}
	provider := m.getProviderForPR(*pr)
	if provider == nil {
		m.statusBar.SetMessage(noProviderMessage, true)
		return m, nil
	}
	m.workItemPickerView.Activate(*pr)
	ctx, inspected := m.ctx, *pr
	return m, func() tea.Msg {
		workItems, err := provider.ListAssignedWorkItems(ctx, prIdentifier(inspected))
		return AssignedWorkItemsLoadedMsg{prKey: inspected.Key(), workItems: workItems, err: err}
	}
}

func (m Model) handleAssignedWorkItemsLoaded(msg AssignedWorkItemsLoadedMsg) (Model, tea.Cmd) {
	if !m.workItemPickerView.IsActive() || m.workItemPickerView.PRKey() != msg.prKey {
		return m, nil
	}
	if msg.err != nil {
		m.workItemPickerView.Deactivate()
		m.statusBar.SetMessage(fmt.Sprintf("Failed to load your work items: %v", msg.err), true)
		return m, nil
	}
	m.workItemPickerView.SetWorkItems(msg.workItems, m.prInspect.LinkedWorkItemIDs())
	return m, nil
}

func (m Model) handleWorkItemPickerKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	action, cmd := m.workItemPickerView.HandleKey(msg)
	switch action {
	case views.WorkItemPickerActionCancel:
		m.workItemPickerView.Deactivate()
	case views.WorkItemPickerActionLink:
		return m.linkSelectedWorkItem()
	}
	return m, cmd
}

func (m Model) linkSelectedWorkItem() (Model, tea.Cmd) {
	pr, selected := m.prInspect.GetPR(), m.workItemPickerView.Selected()
	m.workItemPickerView.Deactivate()
	if pr == nil || selected == nil {
		return m, nil
	}
	provider := m.getProviderForPR(*pr)
	if provider == nil {
		m.statusBar.SetMessage(noProviderMessage, true)
		return m, nil
	}
	ctx, inspected, workItem := m.ctx, *pr, *selected
	m.statusBar.SetMessage(fmt.Sprintf(linkingWorkItemMessage, workItem.ID, workItem.Title), false)
	return m, func() tea.Msg {
		err := provider.LinkWorkItem(ctx, prIdentifier(inspected), workItem.ID)
		return WorkItemLinkedMsg{prKey: inspected.Key(), workItem: workItem, err: err}
	}
}

// handleWorkItemLinked reloads the pull request too: on GitHub linking edits its description.
func (m Model) handleWorkItemLinked(msg WorkItemLinkedMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		m.statusBar.SetMessage(fmt.Sprintf("Failed to link #%d: %v", msg.workItem.ID, msg.err), true)
		return m, nil
	}
	m.statusBar.SetMessage(fmt.Sprintf(linkedWorkItemMessage, msg.workItem.ID, msg.workItem.Title), false)
	pr := m.prInspect.GetPR()
	if pr == nil || pr.Key() != msg.prKey {
		return m, nil
	}
	m.prInspect.ReloadWorkItems()
	return m, tea.Batch(m.loadInspectWorkItems(), m.loadPRDetail(*pr))
}
