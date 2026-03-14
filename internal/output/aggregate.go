package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/riccardomerenda/logq/internal/parser"
)

// GroupResult holds a field value and its count.
type GroupResult struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// GroupBy counts records per unique value of the given field.
// Results are sorted descending by count.
func GroupBy(records []parser.Record, ids []int, field string) []GroupResult {
	counts := make(map[string]int)
	for _, id := range ids {
		val := records[id].Fields[field]
		if val == "" {
			val = "(empty)"
		}
		counts[val]++
	}

	groups := make([]GroupResult, 0, len(counts))
	for v, c := range counts {
		groups = append(groups, GroupResult{Value: v, Count: c})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Count != groups[j].Count {
			return groups[i].Count > groups[j].Count
		}
		return groups[i].Value < groups[j].Value
	})
	return groups
}

// TopN returns the first n results, or all if n <= 0.
func TopN(groups []GroupResult, n int) []GroupResult {
	if n <= 0 || n >= len(groups) {
		return groups
	}
	return groups[:n]
}

// WriteGroups writes aggregation results to w in the given format.
func WriteGroups(w io.Writer, groups []GroupResult, format Format) error {
	switch format {
	case FormatJSON:
		return writeGroupsJSON(w, groups)
	case FormatCSV:
		return writeGroupsCSV(w, groups)
	default:
		return writeGroupsTable(w, groups)
	}
}

func writeGroupsTable(w io.Writer, groups []GroupResult) error {
	if len(groups) == 0 {
		return nil
	}
	// Find max value width for alignment
	maxWidth := 5 // minimum "VALUE"
	for _, g := range groups {
		if len(g.Value) > maxWidth {
			maxWidth = len(g.Value)
		}
	}

	header := fmt.Sprintf("%-*s  %s\n", maxWidth, "VALUE", "COUNT")
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}
	for _, g := range groups {
		line := fmt.Sprintf("%-*s  %d\n", maxWidth, g.Value, g.Count)
		if _, err := io.WriteString(w, line); err != nil {
			return err
		}
	}
	return nil
}

func writeGroupsJSON(w io.Writer, groups []GroupResult) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(groups)
}

func writeGroupsCSV(w io.Writer, groups []GroupResult) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write([]string{"value", "count"}); err != nil {
		return err
	}
	for _, g := range groups {
		if err := cw.Write([]string{g.Value, fmt.Sprintf("%d", g.Count)}); err != nil {
			return err
		}
	}
	return nil
}
