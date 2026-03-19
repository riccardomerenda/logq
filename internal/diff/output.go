package diff

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
)

// WriteDiff writes a diff report to w. Format is "json" or "" (table).
func WriteDiff(w io.Writer, result Result, format string, topN int, threshold float64) error {
	if format == "json" {
		return writeDiffJSON(w, result, topN, threshold)
	}
	return writeDiffTable(w, result, topN, threshold)
}

func writeDiffTable(w io.Writer, r Result, topN int, threshold float64) error {
	fmt.Fprintf(w, "logq diff — %s vs %s\n\n", r.LeftName, r.RightName)

	// Summary
	fmt.Fprintf(w, "  %-20s %8s %8s %8s\n", "", "Before", "After", "Change")
	fmt.Fprintf(w, "  %-20s %8d %8d %8s\n", "Records", r.LeftCount, r.RightCount, FormatChange(r.LeftCount, r.RightCount))
	fmt.Fprintf(w, "  %-20s %8d %8d %8s\n", "Patterns", r.LeftPatterns, r.RightPatterns, FormatChange(r.LeftPatterns, r.RightPatterns))
	fmt.Fprintln(w)

	// Level distribution
	if len(r.Levels) > 0 {
		fmt.Fprintln(w, "  Level Distribution")
		fmt.Fprintf(w, "  %-20s %8s %8s %8s\n", "Level", "Before", "After", "Change")
		for _, l := range r.Levels {
			fmt.Fprintf(w, "  %-20s %8d %8d %8s\n", l.Level, l.LeftCount, l.RightCount, FormatChange(l.LeftCount, l.RightCount))
		}
		fmt.Fprintln(w)
	}

	// New patterns
	newPatterns := r.NewPatterns
	if topN > 0 && topN < len(newPatterns) {
		newPatterns = newPatterns[:topN]
	}
	fmt.Fprintf(w, "  New Patterns (%d only in %s)\n", len(r.NewPatterns), r.RightName)
	if len(r.NewPatterns) == 0 {
		fmt.Fprintln(w, "    (none)")
	} else {
		for _, p := range newPatterns {
			fmt.Fprintf(w, "  %6d  %s\n", p.RightCount, p.Template)
		}
	}
	fmt.Fprintln(w)

	// Gone patterns
	gonePatterns := r.GonePatterns
	if topN > 0 && topN < len(gonePatterns) {
		gonePatterns = gonePatterns[:topN]
	}
	fmt.Fprintf(w, "  Gone Patterns (%d only in %s)\n", len(r.GonePatterns), r.LeftName)
	if len(r.GonePatterns) == 0 {
		fmt.Fprintln(w, "    (none)")
	} else {
		for _, p := range gonePatterns {
			fmt.Fprintf(w, "  %6d  %s\n", p.LeftCount, p.Template)
		}
	}
	fmt.Fprintln(w)

	// Changed patterns (above threshold)
	var filteredChanged []PatternDiff
	for _, p := range r.Changed {
		pct := math.Abs(ChangePercent(p.LeftCount, p.RightCount))
		if pct >= threshold {
			filteredChanged = append(filteredChanged, p)
		}
	}
	if topN > 0 && topN < len(filteredChanged) {
		filteredChanged = filteredChanged[:topN]
	}
	fmt.Fprintf(w, "  Changed Patterns (>%.0f%% change)\n", threshold)
	if len(filteredChanged) == 0 {
		fmt.Fprintln(w, "    (none)")
	} else {
		fmt.Fprintf(w, "  %8s %8s %8s  %s\n", "Before", "After", "Change", "Template")
		for _, p := range filteredChanged {
			fmt.Fprintf(w, "  %8d %8d %8s  %s\n", p.LeftCount, p.RightCount, FormatChange(p.LeftCount, p.RightCount), p.Template)
		}
	}

	return nil
}

func writeDiffJSON(w io.Writer, r Result, topN int, threshold float64) error {
	type jsonSide struct {
		Name     string `json:"name"`
		Records  int    `json:"records"`
		Patterns int    `json:"patterns"`
	}
	type jsonLevel struct {
		Level  string `json:"level"`
		Left   int    `json:"left"`
		Right  int    `json:"right"`
		Change string `json:"change"`
	}
	type jsonPattern struct {
		Template string `json:"template"`
		Left     int    `json:"left,omitempty"`
		Right    int    `json:"right,omitempty"`
		Change   string `json:"change,omitempty"`
	}
	type jsonOutput struct {
		Left    jsonSide      `json:"left"`
		Right   jsonSide      `json:"right"`
		Levels  []jsonLevel   `json:"levels"`
		New     []jsonPattern `json:"new_patterns"`
		Gone    []jsonPattern `json:"gone_patterns"`
		Changed []jsonPattern `json:"changed_patterns"`
	}

	out := jsonOutput{
		Left:    jsonSide{Name: r.LeftName, Records: r.LeftCount, Patterns: r.LeftPatterns},
		Right:   jsonSide{Name: r.RightName, Records: r.RightCount, Patterns: r.RightPatterns},
		Levels:  make([]jsonLevel, 0, len(r.Levels)),
		New:     make([]jsonPattern, 0),
		Gone:    make([]jsonPattern, 0),
		Changed: make([]jsonPattern, 0),
	}

	for _, l := range r.Levels {
		out.Levels = append(out.Levels, jsonLevel{
			Level: l.Level, Left: l.LeftCount, Right: l.RightCount,
			Change: FormatChange(l.LeftCount, l.RightCount),
		})
	}

	newPatterns := r.NewPatterns
	if topN > 0 && topN < len(newPatterns) {
		newPatterns = newPatterns[:topN]
	}
	for _, p := range newPatterns {
		out.New = append(out.New, jsonPattern{Template: p.Template, Right: p.RightCount})
	}

	gonePatterns := r.GonePatterns
	if topN > 0 && topN < len(gonePatterns) {
		gonePatterns = gonePatterns[:topN]
	}
	for _, p := range gonePatterns {
		out.Gone = append(out.Gone, jsonPattern{Template: p.Template, Left: p.LeftCount})
	}

	for _, p := range r.Changed {
		pct := math.Abs(ChangePercent(p.LeftCount, p.RightCount))
		if pct >= threshold {
			out.Changed = append(out.Changed, jsonPattern{
				Template: p.Template, Left: p.LeftCount, Right: p.RightCount,
				Change: FormatChange(p.LeftCount, p.RightCount),
			})
		}
	}
	if topN > 0 && topN < len(out.Changed) {
		out.Changed = out.Changed[:topN]
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
