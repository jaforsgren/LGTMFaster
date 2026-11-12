package azuredevops

import (
	"context"
	"fmt"
	"strings"

	"github.com/johanforsgren/lgtmfaster/internal/logger"
)

// Azure DevOps names change types by word or by flag value, depending on the endpoint.
var fileChangeTypes = map[string]string{
	"add": changeTypeAdd, "1": changeTypeAdd,
	"edit": changeTypeEdit, "2": changeTypeEdit,
	"delete": changeTypeDelete, "16": changeTypeDelete,
}

const (
	changeTypeAdd    = "add"
	changeTypeEdit   = "edit"
	changeTypeDelete = "delete"
)

// fileChange is one changed file of a pull request iteration or a commit, with the blobs before
// and after the change.
type fileChange struct {
	path             string
	changeType       string
	objectID         string
	originalObjectID string
}

// parseFileChange reads a change entry's untyped item; folders and unknown change types are skipped.
func parseFileChange(changeType string, item interface{}) (fileChange, bool) {
	itemMap, ok := item.(map[string]interface{})
	if !ok || itemMap == nil {
		return fileChange{}, false
	}
	path, _ := itemMap["path"].(string)
	isFolder, _ := itemMap["isFolder"].(bool)
	normalized, known := fileChangeTypes[strings.TrimSpace(changeType)]
	if path == "" || isFolder || !known {
		logger.Log("AzureDevOps: Skipping change %q of %q", changeType, path)
		return fileChange{}, false
	}
	objectID, _ := itemMap["objectId"].(string)
	originalObjectID, _ := itemMap["originalObjectId"].(string)
	return fileChange{path: path, changeType: normalized, objectID: objectID, originalObjectID: originalObjectID}, true
}

// fileDiffText fetches the file's blobs and renders the change as a unified diff.
func (c *Client) fileDiffText(ctx context.Context, projectID, repoID string, change fileChange) (string, error) {
	path := change.path
	var b strings.Builder
	switch change.changeType {
	case changeTypeAdd:
		content, err := c.requiredFileContent(ctx, projectID, repoID, change.objectID)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "diff --git a%s b%s\n--- /dev/null\n+++ b%s\n@@ -0,0 +1,%d @@\n", path, path, path, len(content))
		for _, line := range content {
			b.WriteString("+" + line + "\n")
		}
	case changeTypeDelete:
		content, err := c.requiredFileContent(ctx, projectID, repoID, change.originalObjectID)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "diff --git a%s b%s\n--- a%s\n+++ /dev/null\n@@ -1,%d +0,0 @@\n", path, path, path, len(content))
		for _, line := range content {
			b.WriteString("-" + line + "\n")
		}
	default:
		newContent, err := c.requiredFileContent(ctx, projectID, repoID, change.objectID)
		if err != nil {
			return "", err
		}
		oldContent, err := c.requiredFileContent(ctx, projectID, repoID, change.originalObjectID)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "diff --git a%s b%s\n--- a%s\n+++ b%s\n", path, path, path, path)
		b.WriteString(generateUnifiedDiff(oldContent, newContent))
	}
	return b.String(), nil
}

func (c *Client) requiredFileContent(ctx context.Context, projectID, repoID, objectID string) ([]string, error) {
	if objectID == "" {
		return nil, fmt.Errorf("the change has no blob to read")
	}
	return c.getFileContent(ctx, projectID, repoID, objectID)
}
