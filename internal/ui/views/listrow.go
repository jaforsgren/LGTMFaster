package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const (
	listIconColumnWidth  = 3
	selectedRowMarker    = "> "
	unselectedRowMarker  = "  "
	emptyCellPlaceholder = "-"
	breadcrumbSeparator  = " / "
	listTitleRows        = 2
	listHeaderRows       = 1
	listScrollInfoRows   = 1
	listFilterRows       = 2
	listDividerRows      = 1
	listDividerGlyph     = "─"
	// listChromeRows is every row of a list view that is not a list row.
	listChromeRows = listTitleRows + listHeaderRows + listScrollInfoRows + listFilterRows
)

var (
	listHeaderStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	listSelectedRowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Background(lipgloss.Color("#000000")).Bold(true)
	listRowStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	listMutedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	listTitleStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	listDividerStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	detailLabelStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	detailValueStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
)

type listColumn struct {
	title string
	width int
}

// listRow is one rendered line of a list: a coloured icon, then a cell per column in style. A
// dividerAbove row is separated from the rows before it by a discreet rule.
type listRow struct {
	icon         string
	values       []string
	style        lipgloss.Style
	dividerAbove bool
}

// renderListTitle renders the view's location as a breadcrumb, like "/ Pipelines / deploy".
func renderListTitle(segments ...string) string {
	return listTitleStyle.Render(strings.TrimSpace(breadcrumbSeparator + strings.Join(segments, breadcrumbSeparator)))
}

func renderListIcon(icon string, color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(padToWidth(icon, listIconColumnWidth))
}

func renderDetailLabel(label string, width int) string {
	return detailLabelStyle.Render(padToWidth(label+":", width))
}

func renderListHeader(columns []listColumn) string {
	cells := make([]string, len(columns))
	for i, column := range columns {
		cells[i] = padToWidth(truncateString(column.title, column.width), column.width)
	}
	return unselectedRowMarker + padToWidth("", listIconColumnWidth) + " " +
		listHeaderStyle.Render(strings.Join(cells, " "))
}

func renderListRow(row listRow, columns []listColumn, selected bool) string {
	cells := make([]string, len(columns))
	for i, column := range columns {
		cells[i] = padToWidth(truncateString(row.values[i], column.width), column.width)
	}

	marker := unselectedRowMarker
	style := row.style
	if selected {
		marker = selectedRowMarker
		style = listSelectedRowStyle
	}
	return style.Render(marker) + row.icon + " " + style.Render(strings.Join(cells, " "))
}

// renderListRows renders the header and the cursor's window of rows, with a position indicator
// when not every row fits.
func renderListRows(columns []listColumn, count int, cursor listCursor, rowAt func(index int) listRow) string {
	var b strings.Builder
	b.WriteString(renderListHeader(columns))
	b.WriteString("\n")

	start, end := cursor.VisibleRange(count)
	for i := start; i < end; i++ {
		row := rowAt(i)
		row.values = row.values[:len(columns)]
		if row.dividerAbove && i > start {
			b.WriteString(listDividerStyle.Render(strings.Repeat(listDividerGlyph, listRowWidth(columns))))
			b.WriteString("\n")
		}
		b.WriteString(renderListRow(row, columns, i == cursor.Index()))
		b.WriteString("\n")
	}
	if count > end-start {
		b.WriteString(renderScrollInfo(start, end, count))
		b.WriteString("\n")
	}
	return b.String()
}

func listRowWidth(columns []listColumn) int {
	width := len(unselectedRowMarker) + listIconColumnWidth + len(columns)
	for _, column := range columns {
		width += column.width
	}
	return width
}

func renderScrollInfo(start, end, count int) string {
	return listMutedStyle.Render(fmt.Sprintf("[%d-%d of %d]", start+1, end, count))
}

// flexColumnWidth sizes the single stretchy column so the row fills the available width.
func flexColumnWidth(totalWidth, fixedWidth, minWidth int) int {
	const rowChrome = len(unselectedRowMarker) + listIconColumnWidth + 1
	return max(minWidth, totalWidth-fixedWidth-rowChrome)
}

func formatOptionalAge(t time.Time) string {
	if t.IsZero() {
		return emptyCellPlaceholder
	}
	return formatAge(t)
}

func valueOrPlaceholder(value string) string {
	if value == "" {
		return emptyCellPlaceholder
	}
	return value
}

func truncateString(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return runewidth.Truncate(s, maxWidth, "")
	}
	return runewidth.Truncate(s, maxWidth-3, "") + "..."
}

func padToWidth(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return runewidth.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-w)
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
