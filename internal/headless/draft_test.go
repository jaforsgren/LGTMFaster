package headless

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func noEditor(t *testing.T) Editor {
	return func(string) (string, error) {
		t.Error("expected the editor not opened")
		return "", nil
	}
}

func TestDescription_ReadsTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "description.md")
	if err := os.WriteFile(path, []byte("From the file"), 0o600); err != nil {
		t.Fatal(err)
	}

	description, err := (&Flags{DescriptionFile: path}).description(noEditor(t))

	if err != nil || description != "From the file" {
		t.Errorf("expected the file's text, got %q, %v", description, err)
	}
}

func TestDescription_IsWrittenInTheEditor(t *testing.T) {
	description, err := (&Flags{DescriptionEditor: true}).description(func(string) (string, error) { return "Written", nil })

	if err != nil || description != "Written" {
		t.Errorf("expected the edited text, got %q, %v", description, err)
	}
}

func TestDescription_AbortsWhenTheEditorFails(t *testing.T) {
	aborted := errors.New("aborted")

	_, err := (&Flags{DescriptionEditor: true}).description(func(string) (string, error) { return "", aborted })

	if !errors.Is(err, aborted) {
		t.Errorf("expected the editor's error, got %v", err)
	}
}

func TestDescription_FallsBackToTheFlag(t *testing.T) {
	description, _ := (&Flags{Description: "Given"}).description(noEditor(t))

	if description != "Given" {
		t.Errorf("expected the -d text, got %q", description)
	}
}

func TestDraft_RejectsWorkItemsThatAreNotNumbers(t *testing.T) {
	_, err := Draft(t.TempDir(), &Flags{WorkItems: "login"}, noEditor(t))

	if err == nil || !strings.Contains(err.Error(), "-a") {
		t.Errorf("expected the -a value refused, got %v", err)
	}
}

func TestDraft_NeedsAGitRepository(t *testing.T) {
	_, err := Draft(t.TempDir(), &Flags{}, noEditor(t))

	if err == nil || !strings.Contains(err.Error(), "not in a git repository") {
		t.Errorf("expected a git repository asked for, got %v", err)
	}
}
