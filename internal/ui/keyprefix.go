package ui

import tea "github.com/charmbracelet/bubbletea"

// completePrefix runs the two-key sequence the pending prefix and key make and reports whether
// they made one; either way the prefix is dropped.
func (m Model) completePrefix(key string) (Model, tea.Cmd, bool) {
	prefix := m.pendingPrefix
	m.pendingPrefix = ""
	switch prefix {
	case keyYankPrefix:
		return m.completeYank(key)
	case keyNewPRPrefix:
		return m.completeNewPR(key)
	}
	return m, nil, false
}
