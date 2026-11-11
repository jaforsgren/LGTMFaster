package views

import "github.com/charmbracelet/lipgloss"

const (
	paneBorderSize        = 2
	paneVerticalPadding   = 1
	paneHorizontalPadding = 2
	minPaneSize           = 1
)

var (
	mainWindowBorderColor  = lipgloss.Color("81")
	detailPaneBorderColor  = lipgloss.Color("5")
	focusedPaneBorderColor = lipgloss.Color("11")

	mainWindowStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(mainWindowBorderColor).
			Padding(paneVerticalPadding, paneHorizontalPadding)
)

// MainWindowContentSize is the room left for a view's content once the main window's border and
// padding are taken from an outer area of outerWidth x outerHeight.
func MainWindowContentSize(outerWidth, outerHeight int) (int, int) {
	return innerSize(outerWidth, paneHorizontalPadding), innerSize(outerHeight, paneVerticalPadding)
}

// RenderMainWindow frames content that was laid out for innerWidth x innerHeight.
func RenderMainWindow(content string, innerWidth, innerHeight int) string {
	return renderPane(content, innerWidth, innerHeight, mainWindowBorderColor)
}

func innerSize(outer, padding int) int {
	return max(outer-paneBorderSize-2*padding, minPaneSize)
}

func outerSize(inner, padding int) int {
	return inner + paneBorderSize + 2*padding
}

// renderPane truncates content to the pane's inner size first, so long lines are cut rather than
// wrapped and the pane never grows past its layout.
func renderPane(content string, innerWidth, innerHeight int, borderColor lipgloss.Color) string {
	inner := lipgloss.NewStyle().
		MaxWidth(innerWidth).
		MaxHeight(innerHeight).
		Render(content)

	return mainWindowStyle.
		BorderForeground(borderColor).
		Width(innerWidth + 2*paneHorizontalPadding).
		Height(innerHeight + 2*paneVerticalPadding).
		Render(inner)
}
