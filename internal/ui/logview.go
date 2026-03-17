package ui

import (
	"fmt"
	"strings"

	"github.com/riccardomerenda/logq/internal/parser"
)

// LogView renders the scrollable log lines panel.
type LogView struct {
	records        []parser.Record
	results        []int // filtered record indices
	offset         int   // scroll offset
	cursor         int   // selected line (relative to results)
	width          int
	height         int
	highlights     []HighlightTerm
	columns        []string // column mode field names
	colWidths      []int    // computed column widths
	traceOriginIdx int      // record index that initiated trace (-1 when inactive)
}

// NewLogView creates a new log view.
func NewLogView() LogView {
	return LogView{traceOriginIdx: -1}
}

// SetTraceOrigin sets the record index that originated a trace filter.
// Pass -1 to clear.
func (lv *LogView) SetTraceOrigin(idx int) {
	lv.traceOriginIdx = idx
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
	if len(lv.columns) > 0 {
		lv.computeColumnWidths()
	}
}

// SetHighlights sets the terms to highlight in log output.
func (lv *LogView) SetHighlights(h []HighlightTerm) {
	lv.highlights = h
}

// SetColumns sets the column names for table rendering mode.
func (lv *LogView) SetColumns(cols []string) {
	lv.columns = cols
	lv.computeColumnWidths()
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
		emptyLine := StyleBase.Render(strings.Repeat(" ", lv.width))
		msg := padLine(StyleDim.Render("  No matching records"), lv.width)
		var b strings.Builder
		b.WriteString(msg)
		for i := 1; i < lv.height; i++ {
			b.WriteString("\n")
			b.WriteString(emptyLine)
		}
		return b.String()
	}

	// Column mode
	if len(lv.columns) > 0 {
		return lv.viewColumns()
	}

	var b strings.Builder
	end := lv.offset + lv.height
	if end > len(lv.results) {
		end = len(lv.results)
	}

	traceMarker := StyleBase.Copy().Foreground(colorPurple).Bold(true)
	for i := lv.offset; i < end; i++ {
		recIdx := lv.results[i]
		r := lv.records[recIdx]
		line := formatLogLine(r, lv.width-2, lv.highlights)

		// Trace origin gutter marker
		gutter := "  "
		if recIdx == lv.traceOriginIdx {
			gutter = traceMarker.Render("> ")
		}

		if i == lv.cursor {
			line = StyleHighlight.Width(lv.width - 2).Render(line)
		} else {
			line = padLine(line, lv.width-2)
		}
		line = gutter + line

		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Pad remaining lines with background
	rendered := end - lv.offset
	emptyLine := StyleBase.Render(strings.Repeat(" ", lv.width))
	for i := rendered; i < lv.height; i++ {
		b.WriteString("\n")
		b.WriteString(emptyLine)
	}

	return b.String()
}

func formatLogLine(r parser.Record, maxWidth int, highlights []HighlightTerm) string {
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
		srcStyle := StyleBase.Copy().Foreground(colorCyan)
		parts = append(parts, srcStyle.Render("<")+highlightText(src, highlights, srcStyle, "source")+srcStyle.Render(">"))
		usedWidth += len(src) + 2 + 2
	}

	// Service
	if svc, ok := r.Fields["service"]; ok {
		parts = append(parts, StyleDim.Render("[")+highlightText(svc, highlights, StyleDim, "service")+StyleDim.Render("]"))
		usedWidth += len(svc) + 2 + 2
	}

	// Extra fields — collect for width calculation
	skip := map[string]bool{
		"timestamp": true, "ts": true, "time": true, "@timestamp": true,
		"level": true, "lvl": true, "severity": true,
		"message": true, "msg": true, "body": true,
		"service": true, "source": true, "datetime": true, "t": true, "loglevel": true, "text": true,
	}

	type fieldPair struct{ key, val string }
	var extras []fieldPair
	extraWidth := 0
	for k, v := range r.Fields {
		if !skip[k] {
			extras = append(extras, fieldPair{k, v})
			extraWidth += 2 + len(k) + 1 + len(v) // "  k=v"
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
		remaining := maxWidth - usedWidth - extraWidth - 1
		if remaining < 20 {
			// Not enough room for extras, drop them and give full width to message
			remaining = maxWidth - usedWidth - 1
			extras = nil
		}
		if remaining > 0 && len(msg) > remaining {
			msg = msg[:remaining-1] + "…"
		}
		msgStyle := StyleBase.Copy().Foreground(colorWhite)
		parts = append(parts, highlightText(msg, highlights, msgStyle, "message"))
	}

	sep := StyleBase.Render("  ")
	line := strings.Join(parts, sep)

	// Render extras with highlighting on values
	for _, e := range extras {
		line += StyleDim.Render("  "+e.key+"=") + highlightText(e.val, highlights, StyleDim, e.key)
	}

	return line
}

// viewColumns renders the log view in column/table mode.
func (lv *LogView) viewColumns() string {
	var b strings.Builder

	// Header row
	header := lv.formatTableRow(lv.columns, lv.colWidths, true)
	b.WriteString(StyleTitle.Render(header))
	b.WriteString("\n")

	// Data rows (height-1 because header takes one line)
	dataHeight := lv.height - 1
	end := lv.offset + dataHeight
	if end > len(lv.results) {
		end = len(lv.results)
	}

	for i := lv.offset; i < end; i++ {
		recIdx := lv.results[i]
		r := lv.records[recIdx]
		vals := lv.getColumnValues(r)
		line := lv.formatTableRow(vals, lv.colWidths, false)

		if i == lv.cursor {
			line = StyleHighlight.Width(lv.width).Render(line)
		}

		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Pad remaining lines
	rendered := end - lv.offset + 1 // +1 for header
	for i := rendered; i < lv.height; i++ {
		b.WriteString("\n")
	}

	return b.String()
}

// getColumnValues extracts field values for each column from a record.
// Handles pseudo-columns: timestamp, level, message.
func (lv *LogView) getColumnValues(r parser.Record) []string {
	vals := make([]string, len(lv.columns))
	for i, col := range lv.columns {
		switch col {
		case "timestamp", "time", "ts":
			if !r.Timestamp.IsZero() {
				vals[i] = r.Timestamp.Format("2006-01-02 15:04:05")
			}
		case "level":
			vals[i] = strings.ToUpper(r.Level)
		case "message", "msg":
			vals[i] = r.Message
			// Strip to first line
			if idx := strings.IndexByte(vals[i], '\n'); idx >= 0 {
				vals[i] = vals[i][:idx]
			}
		default:
			vals[i] = r.Fields[col]
		}
	}
	return vals
}

// formatTableRow formats a row of values with fixed column widths.
func (lv *LogView) formatTableRow(vals []string, widths []int, isHeader bool) string {
	var parts []string
	for i, v := range vals {
		w := 10
		if i < len(widths) {
			w = widths[i]
		}
		if len(v) > w {
			v = v[:w-1] + "…"
		}
		parts = append(parts, fmt.Sprintf("%-*s", w, v))
	}
	return strings.Join(parts, "  ")
}

// computeColumnWidths auto-sizes columns by sampling the first 100 visible records.
func (lv *LogView) computeColumnWidths() {
	if len(lv.columns) == 0 {
		lv.colWidths = nil
		return
	}

	lv.colWidths = make([]int, len(lv.columns))
	// Start with header widths
	for i, col := range lv.columns {
		lv.colWidths[i] = len(col)
	}

	// Sample first 100 results
	sampleSize := 100
	if sampleSize > len(lv.results) {
		sampleSize = len(lv.results)
	}
	for j := 0; j < sampleSize; j++ {
		r := lv.records[lv.results[j]]
		vals := lv.getColumnValues(r)
		for i, v := range vals {
			if len(v) > lv.colWidths[i] {
				lv.colWidths[i] = len(v)
			}
		}
	}

	// Cap each column width and compute total
	maxColWidth := 50
	totalWidth := 0
	for i := range lv.colWidths {
		if lv.colWidths[i] > maxColWidth {
			lv.colWidths[i] = maxColWidth
		}
		if lv.colWidths[i] < 4 {
			lv.colWidths[i] = 4
		}
		totalWidth += lv.colWidths[i] + 2 // +2 for separator
	}

	// If total exceeds available width, shrink proportionally
	if totalWidth > lv.width && lv.width > 0 {
		ratio := float64(lv.width) / float64(totalWidth)
		for i := range lv.colWidths {
			lv.colWidths[i] = int(float64(lv.colWidths[i]) * ratio)
			if lv.colWidths[i] < 4 {
				lv.colWidths[i] = 4
			}
		}
	}
}
