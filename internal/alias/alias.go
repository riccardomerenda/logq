package alias

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/riccardomerenda/logq/internal/config"
)

// maxExpansionDepth limits recursive alias expansion to prevent infinite loops.
const maxExpansionDepth = 10

// aliasPattern matches @name tokens in a query string.
var aliasPattern = regexp.MustCompile(`@([a-zA-Z_][a-zA-Z0-9_]*)`)

// builtins are always-available aliases.
var builtins = map[string]string{
	"err":  "level:error OR level:fatal",
	"warn": "level:warn OR level:warning",
	"slow": "latency>1000",
}

// Entry represents a resolved alias with its query and optional columns.
type Entry struct {
	Query   string
	Columns []string
}

// Registry holds built-in and user-defined aliases.
type Registry struct {
	aliases map[string]Entry
}

// NewRegistry creates a registry by merging builtins with user-defined aliases.
// User aliases override builtins with the same name.
func NewRegistry(userAliases map[string]config.AliasEntry) *Registry {
	r := &Registry{
		aliases: make(map[string]Entry, len(builtins)+len(userAliases)),
	}
	for name, q := range builtins {
		r.aliases[name] = Entry{Query: q}
	}
	for name, entry := range userAliases {
		r.aliases[name] = Entry{Query: entry.Query, Columns: entry.Columns}
	}
	return r
}

// Expand replaces all @name references in a query with their alias bodies,
// wrapped in parentheses. Returns an error for unknown or circular aliases.
func (r *Registry) Expand(query string) (string, error) {
	if r == nil || !strings.Contains(query, "@") {
		return query, nil
	}

	result := query
	for i := 0; i < maxExpansionDepth; i++ {
		expanded, err := r.expandOnce(result)
		if err != nil {
			return "", err
		}
		if expanded == result {
			return result, nil
		}
		result = expanded
	}

	// After max iterations, check if there are still unresolved aliases
	if aliasPattern.MatchString(result) {
		return "", fmt.Errorf("circular alias detected (expansion exceeded %d levels)", maxExpansionDepth)
	}
	return result, nil
}

// expandOnce performs a single pass of alias expansion, skipping quoted strings.
func (r *Registry) expandOnce(query string) (string, error) {
	// Split into quoted and unquoted segments to avoid expanding inside strings
	segments := splitQuoted(query)
	var b strings.Builder
	for _, seg := range segments {
		if seg.quoted {
			b.WriteString(seg.text)
			continue
		}
		expanded, err := r.expandSegment(seg.text)
		if err != nil {
			return "", err
		}
		b.WriteString(expanded)
	}
	return b.String(), nil
}

// expandSegment replaces @name tokens in an unquoted segment.
func (r *Registry) expandSegment(text string) (string, error) {
	var lastErr error
	result := aliasPattern.ReplaceAllStringFunc(text, func(match string) string {
		name := match[1:] // strip "@"
		entry, ok := r.aliases[name]
		if !ok {
			lastErr = fmt.Errorf("unknown alias: %s", match)
			return match
		}
		return "(" + entry.Query + ")"
	})
	return result, lastErr
}

// segment represents a portion of query text.
type segment struct {
	text   string
	quoted bool
}

// splitQuoted splits text into alternating unquoted and quoted segments.
func splitQuoted(s string) []segment {
	var segments []segment
	start := 0
	inQuote := false
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			if inQuote {
				// End of quoted segment (include the closing quote)
				segments = append(segments, segment{text: s[start : i+1], quoted: true})
				start = i + 1
				inQuote = false
			} else {
				// End of unquoted segment, start of quoted
				if i > start {
					segments = append(segments, segment{text: s[start:i], quoted: false})
				}
				start = i
				inQuote = true
			}
		}
	}
	// Remaining text
	if start < len(s) {
		segments = append(segments, segment{text: s[start:], quoted: inQuote})
	}
	return segments
}

// Names returns all registered alias names (without @), sorted.
func (r *Registry) Names() []string {
	if r == nil {
		return nil
	}
	names := make([]string, 0, len(r.aliases))
	for name := range r.aliases {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Lookup returns the entry for a given alias name (without @).
func (r *Registry) Lookup(name string) (Entry, bool) {
	if r == nil {
		return Entry{}, false
	}
	e, ok := r.aliases[name]
	return e, ok
}
