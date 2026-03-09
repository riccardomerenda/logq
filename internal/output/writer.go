package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/riccardomerenda/logq/internal/parser"
)

// Format represents an output format.
type Format string

const (
	FormatRaw  Format = "raw"
	FormatJSON Format = "json"
	FormatCSV  Format = "csv"
)

// ParseFormat parses a format string, returning an error for unknown formats.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "raw", "":
		return FormatRaw, nil
	case "json":
		return FormatJSON, nil
	case "csv":
		return FormatCSV, nil
	default:
		return "", fmt.Errorf("unknown format %q (valid: raw, json, csv)", s)
	}
}

// Write writes the matched records to w in the given format.
func Write(w io.Writer, records []parser.Record, ids []int, format Format) error {
	switch format {
	case FormatRaw:
		return writeRaw(w, records, ids)
	case FormatJSON:
		return writeJSON(w, records, ids)
	case FormatCSV:
		return writeCSV(w, records, ids)
	default:
		return writeRaw(w, records, ids)
	}
}

func writeRaw(w io.Writer, records []parser.Record, ids []int) error {
	for _, id := range ids {
		if _, err := fmt.Fprintln(w, records[id].Raw); err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(w io.Writer, records []parser.Record, ids []int) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, id := range ids {
		if err := enc.Encode(records[id].Fields); err != nil {
			return err
		}
	}
	return nil
}

func writeCSV(w io.Writer, records []parser.Record, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	// Collect all field keys across matched records
	keySet := make(map[string]bool)
	for _, id := range ids {
		for k := range records[id].Fields {
			keySet[k] = true
		}
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header
	if err := cw.Write(keys); err != nil {
		return err
	}

	// Rows
	row := make([]string, len(keys))
	for _, id := range ids {
		for i, k := range keys {
			row[i] = records[id].Fields[k]
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return nil
}
