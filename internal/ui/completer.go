package ui

import (
	"sort"
	"strings"

	"github.com/riccardomerenda/logq/internal/alias"
	"github.com/riccardomerenda/logq/internal/index"
)

type completeMode int

const (
	completeNone       completeMode = iota
	completeFieldName               // completing a field name (e.g. "lev" -> "level")
	completeFieldValue              // completing a field value (e.g. "level:err" -> "level:error")
	completeAlias                   // completing an alias (e.g. "@er" -> "@err")
)

// Completer manages inline auto-completion state for the query bar.
type Completer struct {
	candidates []string
	index      int
	prefix     string
	mode       completeMode
	field      string // the field name when completing values
}

// Reset clears all completion state.
func (c *Completer) Reset() {
	c.candidates = nil
	c.index = 0
	c.prefix = ""
	c.mode = completeNone
	c.field = ""
}

// HasCandidates returns true if there are active completions.
func (c *Completer) HasCandidates() bool {
	return len(c.candidates) > 0
}

// GhostSuffix returns the dimmed text to append after the cursor.
func (c *Completer) GhostSuffix() string {
	if len(c.candidates) == 0 {
		return ""
	}
	candidate := c.candidates[c.index]
	if len(candidate) <= len(c.prefix) {
		return ""
	}
	suffix := candidate[len(c.prefix):]
	// For field name completion, append ":" to hint that a value follows
	if c.mode == completeFieldName {
		suffix += ":"
	}
	return suffix
}

// Current returns the full text of the current candidate, or "".
func (c *Completer) Current() string {
	if len(c.candidates) == 0 {
		return ""
	}
	return c.candidates[c.index]
}

// Next cycles to the next candidate.
func (c *Completer) Next() {
	if len(c.candidates) == 0 {
		return
	}
	c.index = (c.index + 1) % len(c.candidates)
}

// completionContext holds the parsed context for completion.
type completionContext struct {
	mode       completeMode
	prefix     string // the partial text to match against
	field      string // for value completion: the field name
	tokenStart int    // rune offset where the replaceable prefix starts
}

// extractCompletionContext analyzes the query text at the cursor position
// and determines what kind of completion to offer.
// cursorPos is a rune offset (matching textinput.Cursor()).
func extractCompletionContext(text string, cursorPos int) completionContext {
	runes := []rune(text)
	if cursorPos > len(runes) {
		cursorPos = len(runes)
	}
	if cursorPos == 0 {
		return completionContext{mode: completeNone}
	}

	// Don't complete if cursor is in the middle of a word
	if cursorPos < len(runes) {
		ch := runes[cursorPos]
		if ch != ' ' && ch != '\t' && ch != ')' && ch != ':' && ch != '>' && ch != '<' && ch != '~' {
			return completionContext{mode: completeNone}
		}
	}

	// Walk backwards to find token start
	left := runes[:cursorPos]
	tokenStart := cursorPos
	for tokenStart > 0 {
		ch := left[tokenStart-1]
		if ch == ' ' || ch == '\t' || ch == '(' || ch == ')' {
			break
		}
		tokenStart--
	}

	token := string(left[tokenStart:])
	if token == "" {
		return completionContext{mode: completeNone}
	}

	// Check if inside a quoted string (odd number of quotes before cursor)
	quotes := strings.Count(string(left), "\"")
	if quotes%2 != 0 {
		return completionContext{mode: completeNone}
	}

	// Check for alias prefix (@)
	if strings.HasPrefix(token, "@") {
		return completionContext{
			mode:       completeAlias,
			prefix:     token, // include the "@" in prefix for matching
			tokenStart: tokenStart,
		}
	}

	// Check for operators that don't support value completion
	if strings.ContainsAny(token, "><=~") {
		return completionContext{mode: completeNone}
	}

	// Check for field:value pattern
	if colonIdx := strings.IndexByte(token, ':'); colonIdx >= 0 {
		field := token[:colonIdx]
		valuePrefix := token[colonIdx+1:]
		// Strip quotes from value prefix
		valuePrefix = strings.TrimPrefix(valuePrefix, "\"")
		// tokenStart offset needs to account for field name + colon in runes
		runeColonOffset := len([]rune(field)) + 1
		return completionContext{
			mode:       completeFieldValue,
			prefix:     valuePrefix,
			field:      field,
			tokenStart: tokenStart + runeColonOffset,
		}
	}

	// It's a partial field name (or keyword)
	return completionContext{
		mode:       completeFieldName,
		prefix:     token,
		tokenStart: tokenStart,
	}
}

// keywords that can be completed alongside field names.
var completableKeywords = []string{"AND", "NOT", "OR", "last"}

// computeCandidates returns matching completions for the given context.
func computeCandidates(ctx completionContext, idx *index.Index, aliases *alias.Registry) []string {
	switch ctx.mode {
	case completeFieldName:
		names := idx.FieldNames()
		// Add keywords
		names = append(names, completableKeywords...)
		return filterByPrefix(names, ctx.prefix)

	case completeFieldValue:
		values := idx.FieldValues(ctx.field)
		if values == nil {
			return nil
		}
		return filterByPrefix(values, ctx.prefix)

	case completeAlias:
		if aliases == nil {
			return nil
		}
		aliasNames := aliases.Names()
		// Prefix each name with "@" for display
		prefixed := make([]string, len(aliasNames))
		for i, n := range aliasNames {
			prefixed[i] = "@" + n
		}
		return filterByPrefix(prefixed, ctx.prefix)

	default:
		return nil
	}
}

// filterByPrefix returns entries that start with prefix (case-insensitive), sorted.
func filterByPrefix(items []string, prefix string) []string {
	if prefix == "" {
		sorted := make([]string, len(items))
		copy(sorted, items)
		sort.Strings(sorted)
		return sorted
	}

	lowerPrefix := strings.ToLower(prefix)
	var result []string
	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item), lowerPrefix) {
			result = append(result, item)
		}
	}
	sort.Strings(result)
	return result
}
