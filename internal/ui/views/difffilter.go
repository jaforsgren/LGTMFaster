package views

import (
	"path"
	"sort"
	"strings"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

const (
	noExtension       = "(none)"
	diffLineTypeCtx   = "context"
	contextLinePrefix = " "
)

// ExtensionCount is a file extension in a diff and how many of its files have it.
type ExtensionCount struct {
	Extension string
	Files     int
}

// FileExtension is a path's extension, like ".go", or noExtension for files such as Makefile.
func FileExtension(filePath string) string {
	if extension := path.Ext(filePath); extension != "" {
		return strings.ToLower(extension)
	}
	return noExtension
}

// DiffExtensions lists the extensions of the diff's files, in name order.
func DiffExtensions(diff *domain.Diff) []ExtensionCount {
	if diff == nil {
		return nil
	}
	counts := make(map[string]int)
	for _, file := range diff.Files {
		counts[FileExtension(getFilePath(file))]++
	}
	extensions := make([]ExtensionCount, 0, len(counts))
	for extension, files := range counts {
		extensions = append(extensions, ExtensionCount{Extension: extension, Files: files})
	}
	sort.Slice(extensions, func(i, j int) bool { return extensions[i].Extension < extensions[j].Extension })
	return extensions
}

// filterDiff returns the diff without the files of hidden extensions and, when ignoreWhitespace is
// set, without changes that only touch whitespace. The source diff is not modified.
func filterDiff(diff *domain.Diff, hiddenExtensions map[string]bool, ignoreWhitespace bool) *domain.Diff {
	if diff == nil {
		return nil
	}
	filtered := &domain.Diff{}
	for _, file := range diff.Files {
		if hiddenExtensions[FileExtension(getFilePath(file))] {
			continue
		}
		if ignoreWhitespace {
			file = withoutWhitespaceChanges(file)
		}
		filtered.Files = append(filtered.Files, file)
	}
	return filtered
}

func withoutWhitespaceChanges(file domain.FileDiff) domain.FileDiff {
	hunks := make([]domain.DiffHunk, len(file.Hunks))
	for i, hunk := range file.Hunks {
		hunks[i] = domain.DiffHunk{Header: hunk.Header, Lines: collapseWhitespaceChanges(hunk.Lines)}
	}
	file.Hunks = hunks
	return file
}

// collapseWhitespaceChanges walks each block of deleted lines followed by added lines. A deleted and
// an added line in the same position that differ only in whitespace become one context line, and
// added or deleted blank lines are dropped.
func collapseWhitespaceChanges(lines []domain.DiffLine) []domain.DiffLine {
	var out []domain.DiffLine
	for i := 0; i < len(lines); {
		if lines[i].Type != diffLineTypeDelete && lines[i].Type != diffLineTypeAdd {
			out = append(out, lines[i])
			i++
			continue
		}

		deletesEnd := i
		for deletesEnd < len(lines) && lines[deletesEnd].Type == diffLineTypeDelete {
			deletesEnd++
		}
		addsEnd := deletesEnd
		for addsEnd < len(lines) && lines[addsEnd].Type == diffLineTypeAdd {
			addsEnd++
		}
		out = append(out, collapseChangeBlock(lines[i:deletesEnd], lines[deletesEnd:addsEnd])...)
		i = addsEnd
	}
	return out
}

func collapseChangeBlock(deletes, adds []domain.DiffLine) []domain.DiffLine {
	paired := make(map[int]bool)
	var out []domain.DiffLine
	for i, deleted := range deletes {
		if i < len(adds) && normalizeWhitespace(deleted.Content) == normalizeWhitespace(adds[i].Content) {
			paired[i] = true
			out = append(out, domain.DiffLine{
				Type:    diffLineTypeCtx,
				Content: contextLinePrefix + lineText(adds[i].Content),
				OldLine: deleted.OldLine,
				NewLine: adds[i].NewLine,
			})
			continue
		}
		if normalizeWhitespace(deleted.Content) != "" {
			out = append(out, deleted)
		}
	}
	for i, added := range adds {
		if !paired[i] && normalizeWhitespace(added.Content) != "" {
			out = append(out, added)
		}
	}
	return out
}

// lineText is a diff line without its "+", "-" or " " marker.
func lineText(content string) string {
	if content == "" {
		return ""
	}
	return content[1:]
}

func normalizeWhitespace(content string) string {
	return strings.Join(strings.Fields(lineText(content)), "")
}
