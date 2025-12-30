package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
)

type LogsViewModel struct {
	width      int
	height     int
	offset     int
	active     bool
	logs       []logger.LogEntry
	lastUpdate int
}

func NewLogsView() *LogsViewModel {
	return &LogsViewModel{
		active: false,
		offset: 0,
	}
}

func (m *LogsViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *LogsViewModel) Activate() {
	m.active = true
	m.logs = logger.GetLogs()
	m.offset = 0
	if len(m.logs) > m.getVisibleLines() {
		m.offset = len(m.logs) - m.getVisibleLines()
	}
}

func (m *LogsViewModel) Deactivate() {
	m.active = false
	m.offset = 0
}

func (m *LogsViewModel) IsActive() bool {
	return m.active
}

func (m *LogsViewModel) getVisibleLines() int {
	return m.height - 8
}

func (m *LogsViewModel) Update(msg tea.Msg) tea.Cmd {
	if !m.active {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.offset > 0 {
				m.offset--
			}
		case "down", "j":
			maxOffset := len(m.logs) - m.getVisibleLines()
			if maxOffset < 0 {
				maxOffset = 0
			}
			if m.offset < maxOffset {
				m.offset++
			}
		case "pgup":
			m.offset -= m.getVisibleLines()
			if m.offset < 0 {
				m.offset = 0
			}
		case "pgdown":
			m.offset += m.getVisibleLines()
			maxOffset := len(m.logs) - m.getVisibleLines()
			if maxOffset < 0 {
				maxOffset = 0
			}
			if m.offset > maxOffset {
				m.offset = maxOffset
			}
		case "g", "home":
			m.offset = 0
		case "G", "end":
			maxOffset := len(m.logs) - m.getVisibleLines()
			if maxOffset < 0 {
				maxOffset = 0
			}
			m.offset = maxOffset
		}
	}

	return nil
}

func (m *LogsViewModel) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Padding(1, 0)

	b.WriteString(titleStyle.Render(fmt.Sprintf("Session Logs (%d entries)", len(m.logs))))
	b.WriteString("\n\n")

	if len(m.logs) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true)
		b.WriteString(emptyStyle.Render("No logs yet"))
	} else {
		visibleLines := m.getVisibleLines()
		start := m.offset
		end := start + visibleLines
		if end > len(m.logs) {
			end = len(m.logs)
		}

		for i := start; i < end; i++ {
			entry := m.logs[i]
			timestamp := entry.Timestamp.Format("15:04:05.000")

			logColor := "#E5E7EB"
			if strings.Contains(entry.Message, "[ERROR]") {
				logColor = "#EF4444"
			} else if strings.Contains(entry.Message, "[FILE_WRITE]") {
				logColor = "#F59E0B"
			} else if strings.Contains(entry.Message, "[FILE_OPEN]") {
				logColor = "#10B981"
			}

			lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(logColor))
			b.WriteString(lineStyle.Render(fmt.Sprintf("[%s] %s", timestamp, entry.Message)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)

	scrollInfo := ""
	if len(m.logs) > m.getVisibleLines() {
		scrollInfo = fmt.Sprintf(" | Showing %d-%d of %d", m.offset+1, m.offset+m.getVisibleLines(), len(m.logs))
	}

	help := fmt.Sprintf("j/k: Scroll | PgUp/PgDn: Page | g/G: Top/Bottom | Esc: Close%s", scrollInfo)
	b.WriteString(helpStyle.Render(help))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(1, 2).
		Width(m.width - 4)

	return boxStyle.Render(b.String())
}
