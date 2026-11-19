package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEditorCommand_RunsEditorWithItsArguments(t *testing.T) {
	t.Setenv(editorEnvVar, "code --wait")

	cmd := editorCommand("notes.md")

	if want := []string{"code", "--wait", "notes.md"}; !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("expected %v, got %v", want, cmd.Args)
	}
}

func TestEditorCommand_FallsBackToTheDefaultEditor(t *testing.T) {
	t.Setenv(editorEnvVar, "")

	if cmd := editorCommand("notes.md"); cmd.Args[0] != defaultEditor {
		t.Errorf("expected %s, got %v", defaultEditor, cmd.Args)
	}
}

func TestOpenCommentsMarkdown_OpensTheEditorNotTheSystemApp(t *testing.T) {
	opened := recordOpened(t)
	m := sizedTestModel(t, layoutTestWidth, layoutTestHeight)
	t.Cleanup(func() {
		files, _ := filepath.Glob(filepath.Join(os.TempDir(), fmt.Sprintf(commentsMarkdownFilePattern, 0)))
		for _, file := range files {
			os.Remove(file)
		}
	})

	_, cmd := m.openCommentsMarkdown()

	if cmd == nil || len(*opened) != 0 {
		t.Errorf("expected an editor command and nothing handed to the system, got %v", *opened)
	}
}

func TestCommentsMarkdownClosed_RemovesTheFileAndReportsEditorErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "comments.md")
	if err := os.WriteFile(path, []byte("# Comments"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := sizedTestModel(t, layoutTestWidth, layoutTestHeight)

	m, _ = m.handleCommentsMarkdownClosed(CommentsMarkdownClosedMsg{path: path, err: errors.New("exit status 1")})

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected the markdown file to be removed")
	}
	if !strings.Contains(m.View(), "Editor error: exit status 1") {
		t.Error("expected the editor error in the status bar")
	}
}
