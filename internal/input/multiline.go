package input

import (
	"regexp"
	"strings"
)

// timestampStartPattern matches common timestamp prefixes at the start of a line.
// Covers: HH:MM:SS, YYYY-MM-DD, DD/Mon/YYYY, Mon DD, epoch-like numbers, ISO8601, etc.
var timestampStartPattern = regexp.MustCompile(
	`^(?:` +
		`\d{4}[-/]\d{2}[-/]\d{2}` + // 2026-03-08 or 2026/03/08
		`|\d{2}[-/]\d{2}[-/]\d{4}` + // 08-03-2026 or 08/03/2026
		`|\d{1,2}:\d{2}:\d{2}` + // 12:43:10 (time-only)
		`|\d{2}/[A-Z][a-z]{2}/\d{4}` + // 08/Mar/2026 (nginx)
		`|[A-Z][a-z]{2}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}` + // Jan  2 15:04:05 (syslog)
		`|\d{10,13}[\s,]` + // Unix epoch seconds/millis followed by separator
		`)`,
)

// structuredLinePattern matches lines that are clearly structured log entries
// (JSON or logfmt), which should always be treated as new entries.
var structuredLinePattern = regexp.MustCompile(`^[{\[]|^\w+=`)

// continuationPattern matches lines that are clearly continuations:
// indented lines, stack trace markers, or closing braces.
var continuationPattern = regexp.MustCompile(
	`^(?:` +
		`\s+` + // indented (stack traces, wrapped text)
		`|Caused by:` + // Java chained exceptions
		`|\.{3}\s+\d+\s+more` + // Java "... N more"
		`|\}` + // closing brace (JSON block end)
		`|\]` + // closing bracket
		`)`,
)

// GroupLines merges raw lines into logical multi-line log entries.
// A new entry starts when a line begins with a recognized timestamp pattern
// or looks like a structured log line (JSON/logfmt). Lines that appear to be
// continuations (indented, stack traces, etc.) are appended to the previous entry.
//
// Returns grouped entries where each element contains the full text (joined by \n)
// and the original starting line numbers.
func GroupLines(lines []string) []GroupedEntry {
	if len(lines) == 0 {
		return nil
	}

	// Detect the entry start strategy by sampling the file.
	mode := detectMode(lines)

	if mode == modeSingleLine {
		entries := make([]GroupedEntry, 0, len(lines))
		for i, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			entries = append(entries, GroupedEntry{Text: line, LineNumber: i + 1})
		}
		return entries
	}

	var entries []GroupedEntry
	var current strings.Builder
	currentLine := -1

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			if current.Len() > 0 {
				current.WriteByte('\n')
				current.WriteString(line)
			}
			continue
		}

		start := false
		switch mode {
		case modeTimestampAnchored:
			// Only lines starting with a timestamp begin a new entry
			start = timestampStartPattern.MatchString(line)
		case modeStructured:
			// JSON/logfmt lines begin a new entry
			start = structuredLinePattern.MatchString(line)
		}

		if start {
			if current.Len() > 0 {
				entries = append(entries, GroupedEntry{
					Text:       current.String(),
					LineNumber: currentLine,
				})
				current.Reset()
			}
			current.WriteString(line)
			currentLine = i + 1
		} else {
			if current.Len() == 0 {
				current.WriteString(line)
				currentLine = i + 1
			} else {
				current.WriteByte('\n')
				current.WriteString(line)
			}
		}
	}

	if current.Len() > 0 {
		entries = append(entries, GroupedEntry{
			Text:       current.String(),
			LineNumber: currentLine,
		})
	}

	return entries
}

// GroupedEntry represents a logical log entry that may span multiple raw lines.
type GroupedEntry struct {
	Text       string // full text, possibly multi-line (joined with \n)
	LineNumber int    // original line number of the first line
}

type groupMode int

const (
	modeSingleLine        groupMode = iota // every line is its own entry (JSON, logfmt, etc.)
	modeTimestampAnchored                  // entries start with timestamps, rest is continuation
	modeStructured                         // entries start with { or key=, rest is continuation
)

// detectMode samples the first N lines to decide how to group entries.
func detectMode(lines []string) groupMode {
	sampleSize := 100
	if len(lines) < sampleSize {
		sampleSize = len(lines)
	}

	tsStarts := 0
	structStarts := 0
	nonEmpty := 0

	for i := 0; i < sampleSize; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		nonEmpty++
		if timestampStartPattern.MatchString(line) {
			tsStarts++
		}
		if structuredLinePattern.MatchString(line) {
			structStarts++
		}
	}

	if nonEmpty == 0 {
		return modeSingleLine
	}

	tsRatio := float64(tsStarts) / float64(nonEmpty)
	structRatio := float64(structStarts) / float64(nonEmpty)

	// If most lines are structured (JSON/logfmt), treat as single-line
	if structRatio >= 0.8 {
		return modeSingleLine
	}
	// If most lines are structured but with some continuations, group by structure
	if structRatio > 0.3 {
		return modeStructured
	}
	// If some lines have timestamps, anchor on those
	if tsRatio > 0.02 && tsStarts >= 2 {
		return modeTimestampAnchored
	}
	// Fallback: single-line
	return modeSingleLine
}
