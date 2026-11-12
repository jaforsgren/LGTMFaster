package ui

import (
	"os"
	"testing"
)

// TestMain keeps every test from handing URLs, files or folders to the system: nothing a test
// "opens" may launch a browser or file manager on the developer's machine.
func TestMain(m *testing.M) {
	openExternal = func(string) error { return nil }
	openFolder = func(string) error { return nil }
	os.Exit(m.Run())
}

// recordOpened makes openExternal collect what it is asked to open, for the duration of the test.
func recordOpened(t *testing.T) *[]string {
	t.Helper()
	var opened []string
	previous := openExternal
	openExternal = func(target string) error {
		opened = append(opened, target)
		return nil
	}
	t.Cleanup(func() { openExternal = previous })
	return &opened
}
