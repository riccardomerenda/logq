package trace

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/riccardomerenda/logq/internal/parser"
)

// DefaultIDFields are the field names checked by default when detecting trace IDs.
var DefaultIDFields = []string{
	"trace_id",
	"request_id",
	"correlation_id",
	"span_id",
	"x_request_id",
}

// IDField represents a detected trace/correlation ID field and its value.
type IDField struct {
	Name  string
	Value string
}

var (
	uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	hexRe  = regexp.MustCompile(`^[0-9a-fA-F]{16,}$`)
)

// DetectIDFields finds fields in a record that look like trace/correlation IDs.
// Fields matching knownNames (by normalized name) are returned first, followed
// by fields whose values look like UUIDs or long hex strings.
func DetectIDFields(r parser.Record, knownNames []string) []IDField {
	normalizedKnown := make(map[string]bool, len(knownNames))
	for _, name := range knownNames {
		normalizedKnown[NormalizeFieldName(name)] = true
	}

	var byName, byValue []IDField

	for field, val := range r.Fields {
		if val == "" {
			continue
		}
		norm := NormalizeFieldName(field)
		if normalizedKnown[norm] {
			byName = append(byName, IDField{Name: field, Value: val})
		} else if IsUUIDLike(val) || IsHexID(val) {
			byValue = append(byValue, IDField{Name: field, Value: val})
		}
	}

	return append(byName, byValue...)
}

// NormalizeFieldName converts a field name to a canonical form for comparison.
// Lowercases, converts camelCase boundaries, hyphens, and dots to underscores.
func NormalizeFieldName(s string) string {
	// Insert underscore before uppercase letters preceded by a lowercase letter
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(runes[i-1]) {
			b.WriteRune('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	result := b.String()
	result = strings.ReplaceAll(result, "-", "_")
	result = strings.ReplaceAll(result, ".", "_")
	return result
}

// IsUUIDLike returns true if s looks like a UUID (8-4-4-4-12 hex pattern).
func IsUUIDLike(s string) bool {
	return uuidRe.MatchString(s)
}

// IsHexID returns true if s is a hex string of 16+ characters (common trace ID format).
func IsHexID(s string) bool {
	return hexRe.MatchString(s)
}

// BuildQuery returns a query string for filtering by a specific field value.
// Values containing special characters are quoted.
func BuildQuery(field, value string) string {
	if needsQuoting(value) {
		return field + `:` + `"` + value + `"`
	}
	return field + ":" + value
}

// needsQuoting returns true if a value contains characters that need quoting
// in the query language.
func needsQuoting(s string) bool {
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.') {
			return true
		}
	}
	return false
}
