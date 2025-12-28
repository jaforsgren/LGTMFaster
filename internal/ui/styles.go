package ui

import "github.com/charmbracelet/lipgloss"

var (
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#10B981")
	errorColor     = lipgloss.Color("#EF4444")
	warningColor   = lipgloss.Color("#F59E0B")
	infoColor      = lipgloss.Color("#3B82F6")
	mutedColor     = lipgloss.Color("#6B7280")
	backgroundColor = lipgloss.Color("#1F2937")
	foregroundColor = lipgloss.Color("#F9FAFB")

	authoredColor  = lipgloss.Color("#10B981")
	assignedColor  = lipgloss.Color("#F59E0B")
	otherColor     = lipgloss.Color("#6B7280")
)

var (
	BaseStyle = lipgloss.NewStyle().
			Foreground(foregroundColor)

	TopBarStyle = lipgloss.NewStyle().
			Foreground(foregroundColor).
			Background(primaryColor).
			Bold(true).
			Padding(0, 1)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(foregroundColor).
			Background(lipgloss.Color("#374151")).
			Padding(0, 1)

	CommandBarStyle = lipgloss.NewStyle().
			Foreground(foregroundColor).
			Background(backgroundColor).
			Padding(0, 1).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(primaryColor)

	TitleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			MarginTop(1).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(foregroundColor).
				Background(primaryColor).
				Bold(true).
				Padding(0, 1)

	UnselectedItemStyle = lipgloss.NewStyle().
				Foreground(foregroundColor).
				Padding(0, 1)

	AuthoredPRStyle = lipgloss.NewStyle().
			Foreground(authoredColor).
			Bold(true)

	AssignedPRStyle = lipgloss.NewStyle().
			Foreground(assignedColor).
			Bold(true)

	OtherPRStyle = lipgloss.NewStyle().
			Foreground(otherColor)

	DraftPRStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	DiffAddStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	DiffDeleteStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	DiffContextStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(infoColor)

	HelpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			MarginTop(1)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)

	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			MarginBottom(0)

	ActiveProviderStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	InactiveProviderStyle = lipgloss.NewStyle().
				Foreground(mutedColor)
)

func GetCategoryStyle(category string) lipgloss.Style {
	switch category {
	case "authored":
		return AuthoredPRStyle
	case "assigned":
		return AssignedPRStyle
	default:
		return OtherPRStyle
	}
}

func GetDiffLineStyle(lineType string) lipgloss.Style {
	switch lineType {
	case "add":
		return DiffAddStyle
	case "delete":
		return DiffDeleteStyle
	default:
		return DiffContextStyle
	}
}
