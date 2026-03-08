package parser

import (
	"strings"
	"time"
)

// Record represents a single parsed log line.
type Record struct {
	LineNumber int
	Timestamp  time.Time
	Level      string            // normalized: debug/info/warn/error/fatal
	Message    string
	Fields     map[string]string // all key-value pairs including level, message, etc.
	Raw        string
}

// Parse takes a raw log line and returns a Record.
// Auto-detects format: tries JSON first, then logfmt, then plain text.
func Parse(line string, lineNumber int) Record {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return Record{
			LineNumber: lineNumber,
			Fields:     map[string]string{},
			Raw:        line,
		}
	}

	// Try JSON if line starts with '{'
	if trimmed[0] == '{' {
		if r, ok := parseJSON(trimmed, lineNumber); ok {
			return r
		}
	}

	// Try logfmt if line contains '='
	if strings.Contains(trimmed, "=") {
		if r, ok := parseLogfmt(trimmed, lineNumber); ok {
			return r
		}
	}

	// Fall back to plain text
	return parsePlain(line, lineNumber)
}

// NormalizeLevel maps common level strings to a standard set:
// debug, info, warn, error, fatal.
func NormalizeLevel(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug", "dbg", "d", "trace", "trc":
		return "debug"
	case "info", "inf", "i", "information":
		return "info"
	case "warn", "warning", "wrn", "w":
		return "warn"
	case "error", "err", "e":
		return "error"
	case "fatal", "ftl", "f", "critical", "crit", "panic", "dpanic":
		return "fatal"
	default:
		return strings.ToLower(raw)
	}
}

// Well-known field names for timestamp, level, and message.
var (
	timestampKeys = []string{"timestamp", "ts", "time", "@timestamp", "datetime", "t", "date"}
	levelKeys     = []string{"level", "lvl", "severity", "loglevel"}
	messageKeys   = []string{"message", "msg", "body", "text"}
)

// extractWellKnown looks up well-known fields from a fields map and
// populates the Record's Timestamp, Level, and Message fields.
func extractWellKnown(r *Record) {
	for _, k := range timestampKeys {
		if v, ok := r.Fields[k]; ok && v != "" {
			if t, err := ParseTimestamp(v); err == nil {
				r.Timestamp = t
				break
			}
		}
	}

	for _, k := range levelKeys {
		if v, ok := r.Fields[k]; ok && v != "" {
			r.Level = NormalizeLevel(v)
			break
		}
	}

	for _, k := range messageKeys {
		if v, ok := r.Fields[k]; ok && v != "" {
			r.Message = v
			break
		}
	}
}
