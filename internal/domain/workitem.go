package domain

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

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

// workItemIDSeparators split a typed list of work item numbers: "1234, 5678" or "#1234 #5678".
const workItemIDSeparators = ", "

const workItemIDPrefix = "#"

// ParseWorkItemIDs reads the work item numbers typed as "1234, #5678", each once, in order. Empty
// text is no numbers.
func ParseWorkItemIDs(text string) ([]int, error) {
	fields := strings.FieldsFunc(text, func(r rune) bool { return strings.ContainsRune(workItemIDSeparators, r) })
	ids := make([]int, 0, len(fields))
	for _, field := range fields {
		id, err := strconv.Atoi(strings.TrimPrefix(field, workItemIDPrefix))
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("%q is not a work item number", field)
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
