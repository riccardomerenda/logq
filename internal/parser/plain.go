package parser

// parsePlain treats the entire line as a message field.
func parsePlain(line string, lineNumber int) Record {
	return Record{
		LineNumber: lineNumber,
		Message:    line,
		Fields: map[string]string{
			"message": line,
		},
		Raw: line,
	}
}
