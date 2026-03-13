package parser

import (
	"strconv"
	"strings"
	"time"
)

// Common timestamp formats, ordered by frequency.
var timestampFormats = []string{
	time.RFC3339Nano,              // 2006-01-02T15:04:05.999999999Z07:00
	time.RFC3339,                  // 2006-01-02T15:04:05Z07:00
	"2006-01-02 15:04:05.000",    // common in app logs
	"2006-01-02 15:04:05",        // common in app logs
	"2006-01-02T15:04:05",        // ISO without timezone
	"02/Jan/2006:15:04:05 -0700", // nginx/apache
	time.UnixDate,                // Mon Jan _2 15:04:05 MST 2006
	time.RubyDate,                // Mon Jan 02 15:04:05 -0700 2006
	"Jan _2 15:04:05",            // syslog
	"2006/01/02 15:04:05",        // alternative date format
}

// ParseTimestamp attempts to parse a timestamp string.
// Tries common formats, then Unix epoch (seconds and milliseconds).
func ParseTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	// Try common string formats
	for _, format := range timestampFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	// Try Unix epoch (numeric)
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		if f > 1_000_000_000_000 {
			// Milliseconds
			sec := int64(f / 1000)
			nsec := int64((f - float64(sec)*1000) * 1_000_000)
			return time.Unix(sec, nsec).UTC(), nil
		}
		if f > 1_000_000_000 {
			// Seconds
			sec := int64(f)
			nsec := int64((f - float64(sec)) * 1_000_000_000)
			return time.Unix(sec, nsec).UTC(), nil
		}
	}

	return time.Time{}, &time.ParseError{Value: s, Message: "unrecognized timestamp format"}
}
