package parser

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// parseJSON attempts to parse a JSON line into a Record.
// Returns (record, true) on success, (zero, false) on failure.
func parseJSON(line string, lineNumber int) (Record, bool) {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return Record{}, false
	}

	fields := make(map[string]string, len(raw))
	flatten("", raw, fields)

	r := Record{
		LineNumber: lineNumber,
		Fields:     fields,
		Raw:        line,
	}
	extractWellKnown(&r)
	return r, true
}

// flatten recursively flattens a nested map into dot-notation keys.
func flatten(prefix string, m map[string]interface{}, out map[string]string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]interface{}:
			flatten(key, val, out)
		case []interface{}:
			b, _ := json.Marshal(val)
			out[key] = string(b)
		case nil:
			// skip null values
		case float64:
			// Preserve integer formatting when possible
			if val == float64(int64(val)) {
				out[key] = fmt.Sprintf("%d", int64(val))
			} else {
				out[key] = fmt.Sprintf("%g", val)
			}
		case bool:
			out[key] = fmt.Sprintf("%t", val)
		case string:
			out[key] = val
		default:
			out[key] = fmt.Sprintf("%v", val)
		}
	}
}

// FormatRecord formats a Record as a single display line:
// {timestamp} {level} [{service}] {message} {remaining key=value pairs}
func FormatRecord(r Record) string {
	var b strings.Builder

	// Timestamp
	if !r.Timestamp.IsZero() {
		b.WriteString(r.Timestamp.Format("15:04:05.000"))
	} else {
		b.WriteString("            ")
	}
	b.WriteString("  ")

	// Level
	level := strings.ToUpper(r.Level)
	if level == "" {
		level = "   "
	}
	// Pad to 5 chars
	b.WriteString(fmt.Sprintf("%-5s", level))
	b.WriteString("  ")

	// Service (if present)
	if svc, ok := r.Fields["service"]; ok {
		b.WriteString("[")
		b.WriteString(svc)
		b.WriteString("]  ")
	}

	// Message
	b.WriteString(r.Message)

	// Remaining fields sorted
	skip := map[string]bool{}
	for _, k := range timestampKeys {
		skip[k] = true
	}
	for _, k := range levelKeys {
		skip[k] = true
	}
	for _, k := range messageKeys {
		skip[k] = true
	}
	skip["service"] = true

	var extra []string
	for k := range r.Fields {
		if !skip[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)

	for _, k := range extra {
		b.WriteString("  ")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(r.Fields[k])
	}

	return b.String()
}
