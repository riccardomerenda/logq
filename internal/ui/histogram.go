package ui

import (
	"fmt"
	"strings"

	"github.com/riccardomerenda/logq/internal/index"
)

// Histogram renders a vertical time-based bar chart.
type Histogram struct {
	buckets []index.HistogramBucket
	width   int
	height  int
	cursor  int
	focused bool
}

// NewHistogram creates a new histogram.
func NewHistogram() Histogram {
	return Histogram{}
}

// SetSize updates the histogram dimensions.
func (h *Histogram) SetSize(w, he int) {
	h.width = w
	h.height = he
}

// SetBuckets updates the histogram data.
func (h *Histogram) SetBuckets(buckets []index.HistogramBucket) {
	h.buckets = buckets
	if h.cursor >= len(buckets) {
		h.cursor = 0
	}
}

// SetFocused sets focus state.
func (h *Histogram) SetFocused(f bool) {
	h.focused = f
}

// ScrollUp moves the histogram cursor up.
func (h *Histogram) ScrollUp() {
	if h.cursor > 0 {
		h.cursor--
	}
}

// ScrollDown moves the histogram cursor down.
func (h *Histogram) ScrollDown() {
	if h.cursor < len(h.buckets)-1 {
		h.cursor++
	}
}

// SelectedBucket returns the currently focused bucket, or nil.
func (h *Histogram) SelectedBucket() *index.HistogramBucket {
	if len(h.buckets) == 0 || h.cursor < 0 || h.cursor >= len(h.buckets) {
		return nil
	}
	return &h.buckets[h.cursor]
}

// View renders the histogram.
func (h *Histogram) View() string {
	if len(h.buckets) == 0 || h.width < 10 {
		return StyleDim.Render("No data")
	}

	// Find max count for scaling
	maxCount := 0
	for _, b := range h.buckets {
		if b.Count > maxCount {
			maxCount = b.Count
		}
	}
	if maxCount == 0 {
		return StyleDim.Render("No data")
	}

	// Label width: "HH:MM " = 6 chars, count width varies
	labelWidth := 6
	barMaxWidth := h.width - labelWidth - 6 // space for count
	if barMaxWidth < 3 {
		barMaxWidth = 3
	}

	var b strings.Builder
	titleLine := padLine(StyleTitle.Render("  Timeline"), h.width)
	b.WriteString(titleLine + "\n")

	displayCount := h.height - 2
	if displayCount > len(h.buckets) {
		displayCount = len(h.buckets)
	}

	for i := 0; i < displayCount; i++ {
		bucket := h.buckets[i]
		label := bucket.Start.Format("15:04")
		barLen := (bucket.Count * barMaxWidth) / maxCount
		if barLen == 0 && bucket.Count > 0 {
			barLen = 1
		}

		// Build bar: errors in red, rest in green
		errorBarLen := 0
		if bucket.Errors > 0 && bucket.Count > 0 {
			errorBarLen = (bucket.Errors * barLen) / bucket.Count
			if errorBarLen == 0 {
				errorBarLen = 1
			}
		}
		normalBarLen := barLen - errorBarLen

		bar := StyleHistBar.Render(strings.Repeat("█", normalBarLen)) +
			StyleHistError.Render(strings.Repeat("█", errorBarLen))

		count := fmt.Sprintf("%d", bucket.Count)

		sp := StyleBase.Render(" ")
		line := StyleHistLabel.Render(label) + sp + bar + sp + StyleDim.Render(count)

		if h.focused && i == h.cursor {
			line = StyleHighlight.Width(h.width).Render(
				label + " " + strings.Repeat("█", barLen) + " " + count,
			)
		} else {
			line = padLine(line, h.width)
		}

		b.WriteString(line)
		if i < displayCount-1 {
			b.WriteString("\n")
		}
	}

	// Pad remaining lines with background
	emptyLine := StyleBase.Render(strings.Repeat(" ", h.width))
	for i := displayCount; i < h.height-2; i++ {
		b.WriteString("\n")
		b.WriteString(emptyLine)
	}

	return b.String()
}
