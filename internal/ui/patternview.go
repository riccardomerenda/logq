package ui

import (
	"fmt"
	"strings"

	"github.com/riccardomerenda/logq/internal/pattern"
)

// PatternView renders the cluster list in pattern mode.
type PatternView struct {
	clusters []pattern.Cluster
	cursor   int
	offset   int
	width    int
	height   int
	total    int // total records (for percentage)
}

// NewPatternView creates a new pattern view.
func NewPatternView() PatternView {
	return PatternView{}
}

// SetClusters updates the cluster data.
func (pv *PatternView) SetClusters(clusters []pattern.Cluster, total int) {
	pv.clusters = clusters
	pv.total = total
	if pv.cursor >= len(clusters) {
		pv.cursor = 0
		pv.offset = 0
	}
}

// SetSize updates the viewport dimensions.
func (pv *PatternView) SetSize(w, h int) {
	pv.width = w
	pv.height = h
}

// ScrollUp moves the cursor up.
func (pv *PatternView) ScrollUp(n int) {
	pv.cursor -= n
	if pv.cursor < 0 {
		pv.cursor = 0
	}
	pv.ensureVisible()
}

// ScrollDown moves the cursor down.
func (pv *PatternView) ScrollDown(n int) {
	pv.cursor += n
	if pv.cursor >= len(pv.clusters) {
		pv.cursor = len(pv.clusters) - 1
	}
	if pv.cursor < 0 {
		pv.cursor = 0
	}
	pv.ensureVisible()
}

// GoToStart jumps to the first cluster.
func (pv *PatternView) GoToStart() {
	pv.cursor = 0
	pv.offset = 0
}

// GoToEnd jumps to the last cluster.
func (pv *PatternView) GoToEnd() {
	if len(pv.clusters) > 0 {
		pv.cursor = len(pv.clusters) - 1
	}
	pv.ensureVisible()
}

// SelectedCluster returns the cluster under the cursor, or nil.
func (pv *PatternView) SelectedCluster() *pattern.Cluster {
	if len(pv.clusters) == 0 || pv.cursor < 0 || pv.cursor >= len(pv.clusters) {
		return nil
	}
	return &pv.clusters[pv.cursor]
}

func (pv *PatternView) ensureVisible() {
	if pv.cursor < pv.offset {
		pv.offset = pv.cursor
	}
	if pv.cursor >= pv.offset+pv.height {
		pv.offset = pv.cursor - pv.height + 1
	}
}

// View renders the pattern view.
func (pv *PatternView) View() string {
	if len(pv.clusters) == 0 {
		emptyLine := StyleBase.Render(strings.Repeat(" ", pv.width))
		msg := padLine(StyleDim.Render("  No patterns found"), pv.width)
		var b strings.Builder
		b.WriteString(msg)
		for i := 1; i < pv.height; i++ {
			b.WriteString("\n")
			b.WriteString(emptyLine)
		}
		return b.String()
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("  %6s  %-5s  %s", "COUNT", "%", "TEMPLATE")
	headerLine := StyleTitle.Width(pv.width).Render(header)
	b.WriteString(headerLine)
	b.WriteString("\n")

	dataHeight := pv.height - 1 // header takes one line
	end := pv.offset + dataHeight
	if end > len(pv.clusters) {
		end = len(pv.clusters)
	}

	countStyle := StyleBase.Copy().Foreground(colorCyan).Bold(true)
	pctStyle := StyleDim
	tmplStyle := StyleBase.Copy().Foreground(colorWhite)

	for i := pv.offset; i < end; i++ {
		c := pv.clusters[i]
		pct := float64(c.Count) / float64(pv.total) * 100
		if pv.total == 0 {
			pct = 0
		}

		line := "  " + countStyle.Render(fmt.Sprintf("%6d", c.Count)) +
			"  " + pctStyle.Render(fmt.Sprintf("%4.0f%%", pct)) +
			"  " + tmplStyle.Render(truncate(c.Template, pv.width-20))

		if i == pv.cursor {
			line = StyleHighlight.Width(pv.width).Render(line)
		} else {
			line = padLine(line, pv.width)
		}

		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Pad remaining lines
	rendered := end - pv.offset + 1 // +1 for header
	emptyLine := StyleBase.Render(strings.Repeat(" ", pv.width))
	for i := rendered; i < pv.height; i++ {
		b.WriteString("\n")
		b.WriteString(emptyLine)
	}

	return b.String()
}

func truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if len(s) > maxWidth {
		return s[:maxWidth-1] + "…"
	}
	return s
}
