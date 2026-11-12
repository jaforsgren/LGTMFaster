package domain

// WorkItem is tracked work a pull request can be linked to: an Azure DevOps work item, or a GitHub
// issue. Description is plain text.
type WorkItem struct {
	ID          int
	Type        string
	Title       string
	State       string
	Description string
	URL         string
}
