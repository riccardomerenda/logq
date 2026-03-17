package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/riccardomerenda/logq/internal/parser"
	"github.com/riccardomerenda/logq/internal/trace"
)

// DetailView renders a full-screen overlay showing all fields of a record.
type DetailView struct {
	record     *parser.Record
	width      int
	height     int
	copyMsg    string
	highlights []HighlightTerm

	// Pick mode for trace ID selection
	pickMode   bool
	pickItems  []trace.IDField
	pickCursor int
}

// NewDetailView creates a new detail view.
func NewDetailView() DetailView {
	return DetailView{}
}

// SetRecord sets the record to display.
func (d *DetailView) SetRecord(r *parser.Record) {
	d.record = r
}

// SetHighlights sets the terms to highlight in the detail view.
func (d *DetailView) SetHighlights(h []HighlightTerm) {
	d.highlights = h
}

// SetSize updates the overlay dimensions.
func (d *DetailView) SetSize(w, h int) {
	d.width = w
	d.height = h
}

// EnterPickMode shows a selection list of trace ID fields.
func (d *DetailView) EnterPickMode(items []trace.IDField) {
	d.pickMode = true
	d.pickItems = items
	d.pickCursor = 0
}

// ExitPickMode returns to normal detail view.
func (d *DetailView) ExitPickMode() {
	d.pickMode = false
	d.pickItems = nil
	d.pickCursor = 0
}

// PickUp moves the pick cursor up.
func (d *DetailView) PickUp() {
	if d.pickCursor > 0 {
		d.pickCursor--
	}
}

// PickDown moves the pick cursor down.
func (d *DetailView) PickDown() {
	if d.pickCursor < len(d.pickItems)-1 {
		d.pickCursor++
	}
}

// PickSelected returns the currently selected trace ID field.
func (d *DetailView) PickSelected() trace.IDField {
	return d.pickItems[d.pickCursor]
}

// View renders the detail overlay.
func (d *DetailView) View() string {
	if d.record == nil {
		return ""
	}

	// Pick mode overlay
	if d.pickMode {
		return d.viewPickMode()
	}

	r := d.record
	innerWidth := d.width - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	var b strings.Builder

	title := fmt.Sprintf(" Record #%d ", r.LineNumber)
	b.WriteString(StyleTitle.Render(title))
	b.WriteString("\n\n")

	// Collect all fields, sorted
	var keys []string
	for k := range r.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Find max key length for alignment
	maxKeyLen := 0
	for _, k := range keys {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	valStyle := StyleBase.Copy().Foreground(colorWhite)
	for _, k := range keys {
		v := r.Fields[k]
		keyStr := fmt.Sprintf("  %-*s", maxKeyLen+2, k)
		b.WriteString(StyleDim.Render(keyStr))
		b.WriteString(highlightText(v, d.highlights, valStyle, k))
		b.WriteString("\n")
	}

	// Raw line
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  Raw:"))
	b.WriteString("\n")

	raw := r.Raw
	if len(raw) > innerWidth*3 {
		raw = raw[:innerWidth*3] + "..."
	}
	b.WriteString(StyleDim.Render("  " + raw))
	b.WriteString("\n\n")
	hint := "  Esc close  c copy  t trace"
	if d.copyMsg != "" {
		hint = "  " + d.copyMsg
	}
	b.WriteString(StyleDim.Render(hint))

	content := b.String()

	return StyleBase.Copy().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple).
		Padding(1, 2).
		Width(d.width - 4).
		Render(content)
}

// viewPickMode renders the trace ID selection menu.
func (d *DetailView) viewPickMode() string {
	var b strings.Builder

	b.WriteString(StyleTitle.Render(" Select trace ID "))
	b.WriteString("\n\n")

	for i, item := range d.pickItems {
		prefix := "  "
		if i == d.pickCursor {
			prefix = StyleBase.Copy().Foreground(colorGreen).Bold(true).Render("> ")
		}
		nameStyle := StyleBase.Copy().Foreground(colorCyan)
		valStyle := StyleBase.Copy().Foreground(colorWhite)

		line := prefix + nameStyle.Render(item.Name) + StyleDim.Render(" = ") + valStyle.Render(item.Value)
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  Up/Down select  Enter confirm  Esc cancel"))

	content := b.String()

	return StyleBase.Copy().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple).
		Padding(1, 2).
		Width(d.width - 4).
		Render(content)
}
