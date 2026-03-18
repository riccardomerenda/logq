package ui

import (
	"fmt"
	"time"
)

// StatusBar renders the bottom status line.
type StatusBar struct {
	matchCount   int
	totalCount   int
	queryTime    time.Duration
	filename     string
	fileSize     string
	following      bool
	traceActive    bool
	patternMode    bool
	bookmarkCount  int
	bookmarkFilter bool
	flashMsg       string // temporary message (cleared after one render)
	width          int
}

// NewStatusBar creates a new status bar.
func NewStatusBar() StatusBar {
	return StatusBar{}
}

// SetSize updates the status bar width.
func (sb *StatusBar) SetSize(w int) {
	sb.width = w
}

// Update refreshes status bar data.
func (sb *StatusBar) Update(matches, total int, qt time.Duration, filename, fileSize string) {
	sb.matchCount = matches
	sb.totalCount = total
	sb.queryTime = qt
	sb.filename = filename
	sb.fileSize = fileSize
}

// View renders the status bar.
func (sb *StatusBar) View() string {
	if sb.flashMsg != "" {
		return StyleStatusBar.Width(sb.width).Render(" " + sb.flashMsg)
	}

	left := fmt.Sprintf(" %d matches / %d total", sb.matchCount, sb.totalCount)
	middle := ""
	if sb.queryTime > 0 {
		middle = fmt.Sprintf("  query: %s", sb.queryTime.Truncate(time.Microsecond))
	}
	right := ""
	if sb.filename != "" {
		right = fmt.Sprintf("  %s", sb.filename)
		if sb.fileSize != "" {
			right += fmt.Sprintf(" (%s)", sb.fileSize)
		}
	}
	follow := ""
	if sb.following {
		follow = "  [following]"
	}
	traceTag := ""
	if sb.traceActive {
		traceTag = "  [trace]"
	}
	patternTag := ""
	if sb.patternMode {
		patternTag = "  [patterns]"
	}
	bookmarkTag := ""
	if sb.bookmarkFilter {
		bookmarkTag = "  [bookmarks only]"
	} else if sb.bookmarkCount > 0 {
		bookmarkTag = fmt.Sprintf("  [%d bookmarks]", sb.bookmarkCount)
	}
	help := "  / filter  p patterns  m mark  s save  q quit"
	if sb.patternMode {
		help = "  p normal  Enter drill  Esc back  q quit"
	} else if sb.traceActive {
		help = "  / filter  T clear trace  s save  q quit"
	}

	content := left + middle + right + follow + traceTag + patternTag + bookmarkTag + help
	return StyleStatusBar.Width(sb.width).Render(content)
}
