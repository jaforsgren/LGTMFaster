package common

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

var hunkHeaderRegex = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func ParseUnifiedDiff(diffText string) *domain.Diff {
	lines := strings.Split(diffText, "\n")
	files := []domain.FileDiff{}
	var currentFile *domain.FileDiff
	var currentHunk *domain.DiffHunk
	oldLine, newLine := 0, 0

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			if currentFile != nil && currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
				currentHunk = nil
			}
			if currentFile != nil {
				files = append(files, *currentFile)
			}
			currentFile = &domain.FileDiff{
				Hunks: []domain.DiffHunk{},
			}
		} else if strings.HasPrefix(line, "---") {
			if currentFile != nil {
				path := strings.TrimPrefix(line, "--- ")
				if path != "/dev/null" {
					currentFile.OldPath = strings.TrimPrefix(path, "a/")
				} else {
					currentFile.IsNew = true
				}
			}
		} else if strings.HasPrefix(line, "+++") {
			if currentFile != nil {
				path := strings.TrimPrefix(line, "+++ ")
				if path != "/dev/null" {
					currentFile.NewPath = strings.TrimPrefix(path, "b/")
				} else {
					currentFile.IsDeleted = true
				}
			}
		} else if strings.HasPrefix(line, "@@") {
			if currentFile != nil && currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}
			currentHunk = &domain.DiffHunk{
				Header: line,
				Lines:  []domain.DiffLine{},
			}
			matches := hunkHeaderRegex.FindStringSubmatch(line)
			if len(matches) >= 3 {
				oldLine, _ = strconv.Atoi(matches[1])
				newLine, _ = strconv.Atoi(matches[2])
			}
		} else if currentHunk != nil {
			diffLine := domain.DiffLine{Content: line}
			if strings.HasPrefix(line, "+") {
				diffLine.Type = "add"
				diffLine.NewLine = newLine
				newLine++
			} else if strings.HasPrefix(line, "-") {
				diffLine.Type = "delete"
				diffLine.OldLine = oldLine
				oldLine++
			} else if line != "" {
				diffLine.Type = "context"
				diffLine.OldLine = oldLine
				diffLine.NewLine = newLine
				oldLine++
				newLine++
			}
			if line != "" || diffLine.Type != "" {
				currentHunk.Lines = append(currentHunk.Lines, diffLine)
			}
		}
	}

	if currentFile != nil && currentHunk != nil {
		currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
	}
	if currentFile != nil {
		files = append(files, *currentFile)
	}

	return &domain.Diff{Files: files}
}
