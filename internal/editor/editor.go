// Package editor runs the user's $EDITOR on a file.
package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	EnvVar  = "EDITOR"
	Default = "nvim"
	// TempFilePattern names the markdown files text is edited in.
	TempFilePattern = "lgtmfaster_*.md"
)

// Command opens the file in $EDITOR, which may carry arguments ("code --wait").
func Command(path string) *exec.Cmd {
	args := strings.Fields(os.Getenv(EnvVar))
	if len(args) == 0 {
		args = []string{Default}
	}
	return exec.Command(args[0], append(args[1:], path)...)
}

// Edit lets the user write text in $EDITOR, attached to this terminal, starting from initial. An
// editor that exits with an error (vim's :cq) aborts the edit.
func Edit(initial string) (string, error) {
	file, err := os.CreateTemp("", TempFilePattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	path := file.Name()
	defer os.Remove(path)

	_, err = file.WriteString(initial)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	cmd := Command(path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor %s exited with an error, nothing was saved: %w", cmd.Args[0], err)
	}

	edited, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read the edited file: %w", err)
	}
	return string(edited), nil
}
