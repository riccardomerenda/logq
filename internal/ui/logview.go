package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/riccardomerenda/logq/internal/parser"
)

// LogView renders the scrollable log lines panel.
type LogView struct {
	records []parser.Record
	results []int // filtered record indices
	offset  int   // scroll offset
	cursor  int   // selected line (relative to results)
	width   int
	height  int
}

// NewLogView creates a new log view.
func NewLogView() LogView {
	return LogView{}
}

// SetSize updates the viewport dimensions.
func (lv *LogView) SetSize(w, h int) {
	lv.width = w
	lv.height = h
}

// SetResults updates the filtered results.
func (lv *LogView) SetResults(records []parser.Record, results []int) {
	lv.records = records
	lv.results = results
	// Reset cursor if out of bounds
	if lv.cursor >= len(results) {
		lv.cursor = 0
		lv.offset = 0
	}
}

// ScrollUp moves the cursor up.
func (lv *LogView) ScrollUp(n int) {
	lv.cursor -= n
	if lv.cursor < 0 {
		lv.cursor = 0
	}
	lv.ensureVisible()
}

// ScrollDown moves the cursor down.
func (lv *LogView) ScrollDown(n int) {
	lv.cursor += n
	if lv.cursor >= len(lv.results) {
		lv.cursor = len(lv.results) - 1
	}
	if lv.cursor < 0 {
		lv.cursor = 0
	}
	lv.ensureVisible()
}

// GoToStart jumps to the first record.
func (lv *LogView) GoToStart() {
	lv.cursor = 0
	lv.offset = 0
}

// GoToEnd jumps to the last record.
func (lv *LogView) GoToEnd() {
	if len(lv.results) > 0 {
		lv.cursor = len(lv.results) - 1
	}
	lv.ensureVisible()
}

// SelectedRecordIndex returns the index of the currently selected record,
// or -1 if nothing is selected.
func (lv *LogView) SelectedRecordIndex() int {
	if len(lv.results) == 0 || lv.cursor < 0 || lv.cursor >= len(lv.results) {
		return -1
	}
	return lv.results[lv.cursor]
}

func (lv *LogView) ensureVisible() {
	if lv.cursor < lv.offset {
		lv.offset = lv.cursor
	}
	if lv.cursor >= lv.offset+lv.height {
		lv.offset = lv.cursor - lv.height + 1
	}
}

// View renders the log view.
func (lv *LogView) View() string {
	if len(lv.results) == 0 {
		msg := StyleDim.Render("  No matching records")
		return msg + strings.Repeat("\n", max(0, lv.height-1))
	}

	var b strings.Builder
	end := lv.offset + lv.height
	if end > len(lv.results) {
		end = len(lv.results)
	}

	for i := lv.offset; i < end; i++ {
		recIdx := lv.results[i]
		r := lv.records[recIdx]
		line := formatLogLine(r, lv.width)

		if i == lv.cursor {
			line = StyleHighlight.Width(lv.width).Render(line)
		}

		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Pad remaining lines
	rendered := end - lv.offset
	for i := rendered; i < lv.height; i++ {
		b.WriteString("\n")
	}

	return b.String()
}

func formatLogLine(r parser.Record, maxWidth int) string {
	var parts []string
	usedWidth := 0

	// Timestamp
	if !r.Timestamp.IsZero() {
		ts := r.Timestamp.Format("15:04:05")
		parts = append(parts, StyleDim.Render(ts))
		usedWidth += len(ts) + 2 // +2 for separator
	}

	// Level badge
	if r.Level != "" {
		badge := fmt.Sprintf("%-5s", strings.ToUpper(r.Level))
		parts = append(parts, LevelStyle(r.Level).Render(badge))
		usedWidth += 5 + 2
	}

	// Source file indicator (multi-file mode)
	if src, ok := r.Fields["source"]; ok {
		srcStr := "<" + src + ">"
		parts = append(parts, lipgloss.NewStyle().Foreground(colorCyan).Render(srcStr))
		usedWidth += len(srcStr) + 2
	}

	// Service
	if svc, ok := r.Fields["service"]; ok {
		svcStr := "[" + svc + "]"
		parts = append(parts, StyleDim.Render(svcStr))
		usedWidth += len(svcStr) + 2
	}

	// Extra fields
	skip := map[string]bool{
		"timestamp": true, "ts": true, "time": true, "@timestamp": true,
		"level": true, "lvl": true, "severity": true,
		"message": true, "msg": true, "body": true,
		"service": true, "source": true, "datetime": true, "t": true, "loglevel": true, "text": true,
	}

	var extraStr string
	for k, v := range r.Fields {
		if !skip[k] {
			extraStr += "  " + k + "=" + v
		}
	}

	// Message — first line only, truncated to remaining width
	if r.Message != "" {
		msg := r.Message
		// Strip to first line only
		if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
			msg = msg[:idx]
		}
		// Reserve space for extras, then truncate message to fit
		remaining := maxWidth - usedWidth - len(extraStr) - 1
		if remaining < 20 {
			// Not enough room for extras, drop them and give full width to message
			remaining = maxWidth - usedWidth - 1
			extraStr = ""
		}
		if remaining > 0 && len(msg) > remaining {
			msg = msg[:remaining-1] + "…"
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(colorWhite).Render(msg))
	}

	line := strings.Join(parts, "  ")
	if extraStr != "" {
		line += StyleDim.Render(extraStr)
	}

	return line
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
