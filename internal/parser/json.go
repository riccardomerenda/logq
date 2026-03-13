package parser

import (
	"encoding/json"
	"fmt"
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
