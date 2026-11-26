package ui

import (
	"testing"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func TestYankPrefix_WaitsForTheKeyNamingWhatToCopy(t *testing.T) {
	m, _ := sortTestModel(t)

	m, _ = pressKey(t, m, keyYankPrefix)

	if m.pendingPrefix != keyYankPrefix {
		t.Fatal("expected y to wait for the next key")
	}
}

func TestYankPrefix_AnotherKeyDropsItAndRunsAsUsual(t *testing.T) {
	m, repository := sortTestModel(t)

	m, _ = pressKey(t, m, keyYankPrefix)
	m, _ = pressKey(t, m, "2")

	if m.pendingPrefix != "" {
		t.Error("expected the yank dropped")
	}
	if repository.settings.PRSortMode != domain.PRSortTitle {
		t.Errorf("expected the key to sort as usual, got %q", repository.settings.PRSortMode)
	}
}

func TestYankPRLink_ReportsAPRWithoutURL(t *testing.T) {
	m, _ := sortTestModel(t)

	m, _ = pressKey(t, m, keyYankPrefix)
	m, _ = pressKey(t, m, keyYankPRLink)

	if m.pendingPrefix != "" {
		t.Error("expected the yank finished")
	}
	if msg := m.statusBar.Message(); msg != noPRURLMessage {
		t.Errorf("expected %q, got %q", noPRURLMessage, msg)
	}
}
