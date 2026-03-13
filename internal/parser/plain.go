package parser

import (
	"regexp"
	"strings"
	"time"
)

// plainTimestampPatterns attempt to extract a timestamp from the start of a
// plain text line. Each entry has a regex and a slice of Go time formats to try.
var plainTimestampPatterns = []struct {
	re      *regexp.Regexp
	formats []string
}{
	{
		// ISO-like: 2026-03-08 10:00:01 or 2026-03-08T10:00:01
		re: regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)`),
		formats: []string{
			"2006-01-02T15:04:05.999999999Z07:00",
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05.000",
			"2006-01-02 15:04:05",
		},
	},
	{
		// Time-only: 12:43:10 or 12:43:10.123
		re:      regexp.MustCompile(`^(\d{1,2}:\d{2}:\d{2}(?:\.\d+)?)`),
		formats: []string{"15:04:05.000", "15:04:05"},
	},
	{
		// Syslog: Jan  2 15:04:05
		re:      regexp.MustCompile(`^([A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})`),
		formats: []string{"Jan _2 15:04:05", "Jan 02 15:04:05"},
	},
	{
		// Nginx/Apache: 08/Mar/2026:15:04:05 -0700
		re:      regexp.MustCompile(`^(\d{2}/[A-Z][a-z]{2}/\d{4}:\d{2}:\d{2}:\d{2}\s+[+-]\d{4})`),
		formats: []string{"02/Jan/2006:15:04:05 -0700"},
	},
}

// levelPattern detects log level keywords near the start of a plain text line.
// Uses word boundaries and looks within the first 100 chars to avoid false
// positives from stack traces or message bodies.
var levelPattern = regexp.MustCompile(`(?i)(?:^|[\s\[\(])(DEBUG|INFO|INFORMATION|WARN|WARNING|ERROR|ERR|FATAL|CRITICAL|PANIC)(?:[\s\]\):]|$)`)

// parsePlain treats the line as unstructured text.
// It attempts to extract a timestamp from the beginning and a log level keyword.
// For multi-line entries, it finds the most meaningful line to use as the message.
func parsePlain(line string, lineNumber int) Record {
	r := Record{
		LineNumber: lineNumber,
		Raw:        line,
		Fields:     map[string]string{},
	}

	// Try to extract a leading timestamp from the first line
	firstLine, rest := splitFirstLine(line)
	trimmed := strings.TrimSpace(firstLine)
	for _, p := range plainTimestampPatterns {
		if m := p.re.FindString(trimmed); m != "" {
			for _, layout := range p.formats {
				if t, err := time.Parse(layout, m); err == nil {
					r.Timestamp = t
					r.Fields["timestamp"] = m
					break
				}
			}
			if !r.Timestamp.IsZero() {
				break
			}
		}
	}

	// Determine the message to display
	r.Message = extractMessage(firstLine, rest, !r.Timestamp.IsZero())
	r.Fields["message"] = r.Message

	// Try to extract a log level keyword from the first 200 chars of the full text
	searchArea := strings.TrimSpace(line)
	if len(searchArea) > 200 {
		searchArea = searchArea[:200]
	}
	if matches := levelPattern.FindStringSubmatch(searchArea); len(matches) > 1 {
		r.Level = NormalizeLevel(matches[1])
		r.Fields["level"] = r.Level
	}

	return r
}

// splitFirstLine splits text into first line and the rest.
func splitFirstLine(text string) (string, string) {
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		return text[:idx], text[idx+1:]
	}
	return text, ""
}

// extractMessage determines the best message to show for a plain text entry.
// For multi-line entries where the first line is just a timestamp, it uses the
// next non-empty, non-indented line as the message summary.
func extractMessage(firstLine, rest string, hasTimestamp bool) string {
	isMultiLine := rest != ""

	if !isMultiLine {
		// Single line: use the whole thing
		return firstLine
	}

	// For multi-line entries: find the first meaningful content line.
	// If the first line is just a timestamp (short, no real content beyond the ts),
	// look for the first substantive line in the rest.
	firstTrimmed := strings.TrimSpace(firstLine)
	firstLineIsTimestampOnly := hasTimestamp && len(firstTrimmed) < 60

	if firstLineIsTimestampOnly {
		// Use the first non-empty line from the rest as the summary message
		for _, candidate := range strings.SplitN(rest, "\n", 5) {
			candidate = strings.TrimSpace(candidate)
			if candidate != "" && candidate != "---" && !strings.HasPrefix(candidate, "----") {
				return candidate
			}
		}
	}

	return firstLine
}

