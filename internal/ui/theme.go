package ui

import "github.com/charmbracelet/lipgloss"

// Dracula-inspired color palette
var (
	colorRed     = lipgloss.Color("#FF5555")
	colorOrange  = lipgloss.Color("#FFB86C")
	colorCyan    = lipgloss.Color("#8BE9FD")
	colorGray    = lipgloss.Color("#6272A4")
	colorGreen   = lipgloss.Color("#50FA7B")
	colorPurple  = lipgloss.Color("#BD93F9")
	colorWhite   = lipgloss.Color("#F8F8F2")
	colorBg      = lipgloss.Color("#44475A")
	colorDarkBg  = lipgloss.Color("#282A36")

	// Log levels
	StyleError = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	StyleWarn  = lipgloss.NewStyle().Foreground(colorOrange)
	StyleInfo  = lipgloss.NewStyle().Foreground(colorCyan)
	StyleDebug = lipgloss.NewStyle().Foreground(colorGray)
	StyleFatal = lipgloss.NewStyle().Foreground(colorRed).Bold(true).Underline(true)

	// UI chrome
	StyleBorder    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorGray)
	StyleStatusBar = lipgloss.NewStyle().Background(colorBg).Foreground(colorWhite).Padding(0, 1)
	StyleQueryBar  = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(colorPurple).Padding(0, 1)
	StyleHighlight = lipgloss.NewStyle().Background(colorBg).Foreground(colorWhite)
	StyleDim       = lipgloss.NewStyle().Foreground(colorGray)
	StyleTitle     = lipgloss.NewStyle().Foreground(colorPurple).Bold(true)
	StyleMatch     = lipgloss.NewStyle().Background(lipgloss.Color("#F1FA8C")).Foreground(lipgloss.Color("#282A36")).Bold(true)

	// Histogram
	StyleHistBar   = lipgloss.NewStyle().Foreground(colorGreen)
	StyleHistError = lipgloss.NewStyle().Foreground(colorRed)
	StyleHistLabel = lipgloss.NewStyle().Foreground(colorGray)
)

// LevelStyle returns the appropriate style for a log level.
func LevelStyle(level string) lipgloss.Style {
	switch level {
	case "error":
		return StyleError
	case "warn":
		return StyleWarn
	case "info":
		return StyleInfo
	case "debug":
		return StyleDebug
	case "fatal":
		return StyleFatal
	default:
		return lipgloss.NewStyle()
	}
}
