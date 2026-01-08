package common

import (
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestParseUnifiedDiff(t *testing.T) {
	tests := []struct {
		name     string
		diffText string
		want     *domain.Diff
	}{
		{
			name:     "empty diff",
			diffText: "",
			want:     &domain.Diff{Files: []domain.FileDiff{}},
		},
		{
			name: "simple diff with add",
			diffText: `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 line 1
 line 2
+new line
 line 3`,
			want: &domain.Diff{
				Files: []domain.FileDiff{
					{
						OldPath: "file.txt",
						NewPath: "file.txt",
						Hunks: []domain.DiffHunk{
							{
								Header: "@@ -1,3 +1,4 @@",
								Lines: []domain.DiffLine{
									{Content: " line 1", Type: "context", OldLine: 1, NewLine: 1},
									{Content: " line 2", Type: "context", OldLine: 2, NewLine: 2},
									{Content: "+new line", Type: "add", NewLine: 3},
									{Content: " line 3", Type: "context", OldLine: 3, NewLine: 4},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "new file",
			diffText: `diff --git a/newfile.txt b/newfile.txt
--- /dev/null
+++ b/newfile.txt
@@ -0,0 +1,2 @@
+line 1
+line 2`,
			want: &domain.Diff{
				Files: []domain.FileDiff{
					{
						NewPath: "newfile.txt",
						IsNew:   true,
						Hunks: []domain.DiffHunk{
							{
								Header: "@@ -0,0 +1,2 @@",
								Lines: []domain.DiffLine{
									{Content: "+line 1", Type: "add", NewLine: 1},
									{Content: "+line 2", Type: "add", NewLine: 2},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "deleted file",
			diffText: `diff --git a/oldfile.txt b/oldfile.txt
--- a/oldfile.txt
+++ /dev/null
@@ -1,2 +0,0 @@
-line 1
-line 2`,
			want: &domain.Diff{
				Files: []domain.FileDiff{
					{
						OldPath:   "oldfile.txt",
						IsDeleted: true,
						Hunks: []domain.DiffHunk{
							{
								Header: "@@ -1,2 +0,0 @@",
								Lines: []domain.DiffLine{
									{Content: "-line 1", Type: "delete", OldLine: 1},
									{Content: "-line 2", Type: "delete", OldLine: 2},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "github format with index line",
			diffText: "diff --git a/README.md b/README.md\n" +
				"index 1234567..89abcdef 100644\n" +
				"--- a/README.md\n" +
				"+++ b/README.md\n" +
				"@@ -1,5 +1,6 @@\n" +
				" # LGTMFaster\n" +
				" \n" +
				" A tool for reviewing PRs\n" +
				"+Now with multiple PAT support!\n" +
				" \n" +
				" ## Installation",
			want: &domain.Diff{
				Files: []domain.FileDiff{
					{
						OldPath: "README.md",
						NewPath: "README.md",
						Hunks: []domain.DiffHunk{
							{
								Header: "@@ -1,5 +1,6 @@",
								Lines: []domain.DiffLine{
									{Content: " # LGTMFaster", Type: "context", OldLine: 1, NewLine: 1},
									{Content: " ", Type: "context", OldLine: 2, NewLine: 2},
									{Content: " A tool for reviewing PRs", Type: "context", OldLine: 3, NewLine: 3},
									{Content: "+Now with multiple PAT support!", Type: "add", NewLine: 4},
									{Content: " ", Type: "context", OldLine: 4, NewLine: 5},
									{Content: " ## Installation", Type: "context", OldLine: 5, NewLine: 6},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple files",
			diffText: `diff --git a/file1.txt b/file1.txt
--- a/file1.txt
+++ b/file1.txt
@@ -1,1 +1,2 @@
 line 1
+added line
diff --git a/file2.txt b/file2.txt
--- a/file2.txt
+++ b/file2.txt
@@ -1,2 +1,1 @@
 line 1
-removed line`,
			want: &domain.Diff{
				Files: []domain.FileDiff{
					{
						OldPath: "file1.txt",
						NewPath: "file1.txt",
						Hunks: []domain.DiffHunk{
							{
								Header: "@@ -1,1 +1,2 @@",
								Lines: []domain.DiffLine{
									{Content: " line 1", Type: "context", OldLine: 1, NewLine: 1},
									{Content: "+added line", Type: "add", NewLine: 2},
								},
							},
						},
					},
					{
						OldPath: "file2.txt",
						NewPath: "file2.txt",
						Hunks: []domain.DiffHunk{
							{
								Header: "@@ -1,2 +1,1 @@",
								Lines: []domain.DiffLine{
									{Content: " line 1", Type: "context", OldLine: 1, NewLine: 1},
									{Content: "-removed line", Type: "delete", OldLine: 2},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseUnifiedDiff(tt.diffText)
			if len(got.Files) != len(tt.want.Files) {
				t.Errorf("ParseUnifiedDiff() files count = %v, want %v", len(got.Files), len(tt.want.Files))
				return
			}
			for i := range got.Files {
				gotFile := got.Files[i]
				wantFile := tt.want.Files[i]

				if gotFile.OldPath != wantFile.OldPath {
					t.Errorf("File %d OldPath = %v, want %v", i, gotFile.OldPath, wantFile.OldPath)
				}
				if gotFile.NewPath != wantFile.NewPath {
					t.Errorf("File %d NewPath = %v, want %v", i, gotFile.NewPath, wantFile.NewPath)
				}
				if gotFile.IsNew != wantFile.IsNew {
					t.Errorf("File %d IsNew = %v, want %v", i, gotFile.IsNew, wantFile.IsNew)
				}
				if gotFile.IsDeleted != wantFile.IsDeleted {
					t.Errorf("File %d IsDeleted = %v, want %v", i, gotFile.IsDeleted, wantFile.IsDeleted)
				}
				if len(gotFile.Hunks) != len(wantFile.Hunks) {
					t.Errorf("File %d hunks count = %v, want %v", i, len(gotFile.Hunks), len(wantFile.Hunks))
					continue
				}
				for j := range gotFile.Hunks {
					gotHunk := gotFile.Hunks[j]
					wantHunk := wantFile.Hunks[j]

					if gotHunk.Header != wantHunk.Header {
						t.Errorf("File %d Hunk %d Header = %v, want %v", i, j, gotHunk.Header, wantHunk.Header)
					}
					if len(gotHunk.Lines) != len(wantHunk.Lines) {
						t.Errorf("File %d Hunk %d lines count = %v, want %v", i, j, len(gotHunk.Lines), len(wantHunk.Lines))
						continue
					}
					for k := range gotHunk.Lines {
						gotLine := gotHunk.Lines[k]
						wantLine := wantHunk.Lines[k]

						if gotLine.Content != wantLine.Content {
							t.Errorf("File %d Hunk %d Line %d Content = %v, want %v", i, j, k, gotLine.Content, wantLine.Content)
						}
						if gotLine.Type != wantLine.Type {
							t.Errorf("File %d Hunk %d Line %d Type = %v, want %v", i, j, k, gotLine.Type, wantLine.Type)
						}
						if gotLine.OldLine != wantLine.OldLine {
							t.Errorf("File %d Hunk %d Line %d OldLine = %v, want %v", i, j, k, gotLine.OldLine, wantLine.OldLine)
						}
						if gotLine.NewLine != wantLine.NewLine {
							t.Errorf("File %d Hunk %d Line %d NewLine = %v, want %v", i, j, k, gotLine.NewLine, wantLine.NewLine)
						}
					}
				}
			}
		})
	}
}
