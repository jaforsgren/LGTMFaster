package views

import (
	"path"
	"strings"

	"github.com/johanforsgren/lgtmfaster/internal/logger"
)

const pathSeparator = "/"

// diffExclusions are the patterns of the files that can be hidden from a diff (Settings.DiffExclusions),
// lower-cased since they match case-insensitively.
type diffExclusions []string

// newDiffExclusions keeps the valid patterns; an invalid glob is logged and left out rather than
// failing the diff.
func newDiffExclusions(patterns []string) diffExclusions {
	var exclusions diffExclusions
	for _, pattern := range patterns {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if strings.Trim(pattern, pathSeparator) == "" {
			continue
		}
		if _, err := path.Match(pattern, ""); err != nil {
			logger.LogError("DIFF_EXCLUSIONS", pattern, err)
			continue
		}
		exclusions = append(exclusions, pattern)
	}
	return exclusions
}

// excludes reports whether a diff path matches a pattern. Patterns are globs (path.Match):
//   - ending in "/": a folder at any depth, so "tests/" hides "Api/tests/FooTests.cs"
//   - containing "/" otherwise: the whole path, so "src/*.props" hides "src/Directory.Build.props"
//   - else: the file name, so "*.test.*" hides "web/app.test.ts"
func (e diffExclusions) excludes(filePath string) bool {
	filePath = strings.ToLower(strings.TrimPrefix(filePath, pathSeparator))
	folder, name := path.Split(filePath)
	var folders []string
	if folder != "" {
		folders = strings.Split(strings.TrimSuffix(folder, pathSeparator), pathSeparator)
	}

	for _, pattern := range e {
		var matched bool
		switch {
		case strings.HasSuffix(pattern, pathSeparator):
			matched = containsFolders(folders, strings.Split(strings.Trim(pattern, pathSeparator), pathSeparator))
		case strings.Contains(pattern, pathSeparator):
			matched = globMatches(strings.TrimPrefix(pattern, pathSeparator), filePath)
		default:
			matched = globMatches(pattern, name)
		}
		if matched {
			return true
		}
	}
	return false
}

// containsFolders reports whether the wanted folder names appear in a row among the path's folders.
func containsFolders(folders, wanted []string) bool {
	for start := 0; start+len(wanted) <= len(folders); start++ {
		if allGlobsMatch(wanted, folders[start:start+len(wanted)]) {
			return true
		}
	}
	return false
}

func allGlobsMatch(patterns, names []string) bool {
	for i, pattern := range patterns {
		if !globMatches(pattern, names[i]) {
			return false
		}
	}
	return true
}

// globMatches ignores path.Match's error, since newDiffExclusions only keeps valid patterns.
func globMatches(pattern, name string) bool {
	matched, _ := path.Match(pattern, name)
	return matched
}
