package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/riccardomerenda/logq/internal/parser"
)

// DetailView renders a full-screen overlay showing all fields of a record.
type DetailView struct {
	record     *parser.Record
	width      int
	height     int
	copyMsg    string
	highlights []HighlightTerm
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

// View renders the detail overlay.
func (d *DetailView) View() string {
	if d.record == nil {
		return ""
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

	valStyle := lipgloss.NewStyle().Foreground(colorWhite)
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
	hint := "  Press Escape to close, c to copy raw"
	if d.copyMsg != "" {
		hint = "  " + d.copyMsg
		d.copyMsg = ""
	}
	b.WriteString(StyleDim.Render(hint))

	content := b.String()

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple).
		Padding(1, 2).
		Width(d.width - 4).
		Render(content)
}
