package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Theme holds all colors used throughout the UI.
type Theme struct {
	Red            lipgloss.Color
	Orange         lipgloss.Color
	Cyan           lipgloss.Color
	Gray           lipgloss.Color
	Green          lipgloss.Color
	Purple         lipgloss.Color
	White          lipgloss.Color
	Bg             lipgloss.Color
	DarkBg         lipgloss.Color
	MatchBg        lipgloss.Color
	MatchFg        lipgloss.Color
	FillBackground bool // if true, paint the viewport with DarkBg
}

// DarkTheme is the default Dracula-inspired dark palette.
var DarkTheme = Theme{
	Red:     lipgloss.Color("#FF5555"),
	Orange:  lipgloss.Color("#FFB86C"),
	Cyan:    lipgloss.Color("#8BE9FD"),
	Gray:    lipgloss.Color("#6272A4"),
	Green:   lipgloss.Color("#50FA7B"),
	Purple:  lipgloss.Color("#BD93F9"),
	White:   lipgloss.Color("#F8F8F2"),
	Bg:      lipgloss.Color("#44475A"),
	DarkBg:  lipgloss.Color("#282A36"),
	MatchBg: lipgloss.Color("#F1FA8C"),
	MatchFg: lipgloss.Color("#282A36"),
}

// LightTheme is a light palette with dark text on light backgrounds.
var LightTheme = Theme{
	Red:            lipgloss.Color("#D32F2F"),
	Orange:         lipgloss.Color("#E65100"),
	Cyan:           lipgloss.Color("#00838F"),
	Gray:           lipgloss.Color("#757575"),
	Green:          lipgloss.Color("#2E7D32"),
	Purple:         lipgloss.Color("#7B1FA2"),
	White:          lipgloss.Color("#212121"),
	Bg:             lipgloss.Color("#E0E0E0"),
	DarkBg:         lipgloss.Color("#FAFAFA"),
	MatchBg:        lipgloss.Color("#FFF176"),
	MatchFg:        lipgloss.Color("#212121"),
	FillBackground: true,
}

// Package-level color variables (referenced by other UI files).
var (
	colorRed    lipgloss.Color
	colorOrange lipgloss.Color
	colorCyan   lipgloss.Color
	colorGray   lipgloss.Color
	colorGreen  lipgloss.Color
	colorPurple lipgloss.Color
	colorWhite  lipgloss.Color
	colorBg     lipgloss.Color
	colorDarkBg lipgloss.Color
)

// StyleBase is the base style — empty for dark theme, has background for light.
// Use StyleBase.Copy() when creating inline styles in rendering code.
var StyleBase lipgloss.Style

// Style variables (referenced by other UI files).
var (
	StyleError     lipgloss.Style
	StyleWarn      lipgloss.Style
	StyleInfo      lipgloss.Style
	StyleDebug     lipgloss.Style
	StyleFatal     lipgloss.Style
	StyleBorder    lipgloss.Style
	StyleStatusBar lipgloss.Style
	StyleQueryBar  lipgloss.Style
	StyleHighlight lipgloss.Style
	StyleDim       lipgloss.Style
	StyleTitle     lipgloss.Style
	StyleMatch     lipgloss.Style
	StyleHistBar   lipgloss.Style
	StyleHistError lipgloss.Style
	StyleHistLabel lipgloss.Style

	themeFillBg bool // whether to paint the viewport background
)

func init() {
	ApplyTheme(DarkTheme)
}

// ApplyTheme sets all package-level color and style variables from the given theme.
func ApplyTheme(t Theme) {
	colorRed = t.Red
	colorOrange = t.Orange
	colorCyan = t.Cyan
	colorGray = t.Gray
	colorGreen = t.Green
	colorPurple = t.Purple
	colorWhite = t.White
	colorBg = t.Bg
	colorDarkBg = t.DarkBg
	themeFillBg = t.FillBackground

	if t.FillBackground {
		StyleBase = lipgloss.NewStyle().Background(colorDarkBg)
	} else {
		StyleBase = lipgloss.NewStyle()
	}

	// Log levels
	StyleError = StyleBase.Copy().Foreground(colorRed).Bold(true)
	StyleWarn = StyleBase.Copy().Foreground(colorOrange)
	StyleInfo = StyleBase.Copy().Foreground(colorCyan)
	StyleDebug = StyleBase.Copy().Foreground(colorGray)
	StyleFatal = StyleBase.Copy().Foreground(colorRed).Bold(true).Underline(true)

	// UI chrome — these have their own backgrounds, no need for StyleBase
	StyleBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorGray)
	StyleStatusBar = lipgloss.NewStyle().Background(colorBg).Foreground(colorWhite).Padding(0, 1)
	StyleQueryBar = lipgloss.NewStyle().Background(colorDarkBg).Border(lipgloss.NormalBorder()).BorderForeground(colorPurple).Padding(0, 1)
	StyleHighlight = lipgloss.NewStyle().Background(colorBg).Foreground(colorWhite)
	StyleDim = StyleBase.Copy().Foreground(colorGray)
	StyleTitle = StyleBase.Copy().Foreground(colorPurple).Bold(true)
	StyleMatch = lipgloss.NewStyle().Background(t.MatchBg).Foreground(t.MatchFg).Bold(true)

	// Histogram
	StyleHistBar = StyleBase.Copy().Foreground(colorGreen)
	StyleHistError = StyleBase.Copy().Foreground(colorRed)
	StyleHistLabel = StyleBase.Copy().Foreground(colorGray)
}

// RenderAppBackground pads content to fill the viewport when the theme requires it.
// For dark theme, returns content unchanged.
func RenderAppBackground(content string, width, height int) string {
	if !themeFillBg || width <= 0 {
		return content
	}
	// Build a reusable padding string — appended to every line to ensure
	// any trailing unstyled gaps are covered. In altscreen mode extra
	// characters beyond the terminal width are harmlessly clipped.
	pad := StyleBase.Render(strings.Repeat(" ", width))
	lines := strings.Split(content, "\n")
	// Pad to full height
	for len(lines) < height {
		lines = append(lines, pad)
	}
	// Append background padding to every line
	for i := range lines {
		lines[i] = lines[i] + pad
	}
	return strings.Join(lines, "\n")
}

// padLine pads a rendered line to the given width using the theme background.
func padLine(line string, width int) string {
	visLen := lipgloss.Width(line)
	if visLen < width {
		return line + StyleBase.Render(strings.Repeat(" ", width-visLen))
	}
	return line
}

// DetectTheme returns LightTheme or DarkTheme based on the terminal background.
func DetectTheme() Theme {
	if !termenv.HasDarkBackground() {
		return LightTheme
	}
	return DarkTheme
}

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
		return StyleBase.Copy()
	}
}
