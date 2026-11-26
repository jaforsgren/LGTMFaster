package editor

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCommand_RunsEditorWithItsArguments(t *testing.T) {
	t.Setenv(EnvVar, "code --wait")

	cmd := Command("notes.md")

	if want := []string{"code", "--wait", "notes.md"}; !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("expected %v, got %v", want, cmd.Args)
	}
}

func TestCommand_FallsBackToTheDefaultEditor(t *testing.T) {
	t.Setenv(EnvVar, "")

	if cmd := Command("notes.md"); cmd.Args[0] != Default {
		t.Errorf("expected %s, got %v", Default, cmd.Args)
	}
}

func TestEdit_ReturnsWhatTheEditorSaved(t *testing.T) {
	script := filepath.Join(t.TempDir(), "editor.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf edited >> \"$1\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvVar, script)

	text, err := Edit("start ")

	if err != nil || text != "start edited" {
		t.Errorf("expected %q, got %q, %v", "start edited", text, err)
	}
}

func TestEdit_AbortsWhenTheEditorFails(t *testing.T) {
	t.Setenv(EnvVar, "false")

	if _, err := Edit(""); err == nil || !strings.Contains(err.Error(), "nothing was saved") {
		t.Errorf("expected the edit aborted, got %v", err)
	}
}
