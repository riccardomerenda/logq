package parser

// parseLogfmt attempts to parse a logfmt-formatted line.
// Format: key=value key2="value with spaces" key3=123
// Returns (record, true) on success, (zero, false) if it doesn't look like logfmt.
func parseLogfmt(line string, lineNumber int) (Record, bool) {
	fields := make(map[string]string)
	i := 0
	n := len(line)
	pairCount := 0

	for i < n {
		// Skip whitespace
		for i < n && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i >= n {
			break
		}

		// Read key (up to '=' or whitespace)
		keyStart := i
		for i < n && line[i] != '=' && line[i] != ' ' && line[i] != '\t' {
			i++
		}
		if i >= n || line[i] != '=' {
			// No '=' found for this token — not logfmt
			if pairCount == 0 {
				return Record{}, false
			}
			break
		}
		key := line[keyStart:i]
		i++ // skip '='

		// Read value
		var value string
		if i < n && line[i] == '"' {
			// Quoted value
			i++ // skip opening quote
			valStart := i
			for i < n && line[i] != '"' {
				if line[i] == '\\' && i+1 < n {
					i++ // skip escaped char
				}
				i++
			}
			value = line[valStart:i]
			if i < n {
				i++ // skip closing quote
			}
		} else {
			// Unquoted value (up to next whitespace)
			valStart := i
			for i < n && line[i] != ' ' && line[i] != '\t' {
				i++
			}
			value = line[valStart:i]
		}

		if key != "" {
			fields[key] = value
			pairCount++
		}
	}

	// Need at least 2 key=value pairs to consider it logfmt
	if pairCount < 2 {
		return Record{}, false
	}

	// Map well-known logfmt aliases (msg → message, ts → timestamp)
	if _, ok := fields["msg"]; ok {
		if _, exists := fields["message"]; !exists {
			fields["message"] = fields["msg"]
		}
	}
	if _, ok := fields["ts"]; ok {
		if _, exists := fields["timestamp"]; !exists {
			fields["timestamp"] = fields["ts"]
		}
	}
	if _, ok := fields["lvl"]; ok {
		if _, exists := fields["level"]; !exists {
			fields["level"] = fields["lvl"]
		}
	}

	r := Record{
		LineNumber: lineNumber,
		Fields:     fields,
		Raw:        line,
	}
	extractWellKnown(&r)
	return r, true
}
