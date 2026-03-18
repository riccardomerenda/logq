package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
)

// PatternResult holds a message template and its occurrence count.
type PatternResult struct {
	Template string `json:"template"`
	Count    int    `json:"count"`
}

// WritePatterns writes pattern clustering results to w in the given format.
func WritePatterns(w io.Writer, results []PatternResult, format Format) error {
	switch format {
	case FormatJSON:
		return writePatternsJSON(w, results)
	case FormatCSV:
		return writePatternsCSV(w, results)
	default:
		return writePatternsTable(w, results)
	}
}

func writePatternsTable(w io.Writer, results []PatternResult) error {
	if len(results) == 0 {
		return nil
	}
	header := fmt.Sprintf("%6s  %s\n", "COUNT", "TEMPLATE")
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}
	for _, r := range results {
		line := fmt.Sprintf("%6d  %s\n", r.Count, r.Template)
		if _, err := io.WriteString(w, line); err != nil {
			return err
		}
	}
	return nil
}

func writePatternsJSON(w io.Writer, results []PatternResult) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

func writePatternsCSV(w io.Writer, results []PatternResult) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write([]string{"template", "count"}); err != nil {
		return err
	}
	for _, r := range results {
		if err := cw.Write([]string{r.Template, fmt.Sprintf("%d", r.Count)}); err != nil {
			return err
		}
	}
	return nil
}
