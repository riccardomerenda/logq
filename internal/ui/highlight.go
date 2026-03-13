package ui

import (
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/riccardomerenda/logq/internal/query"
)

// HighlightTerm represents a search term to highlight in log output.
type HighlightTerm struct {
	Text    string         // plain text (case-insensitive match)
	Pattern *regexp.Regexp // compiled regex (for field~"pattern" queries)
	Field   string         // if non-empty, only highlight within this field
}

// messageFields are aliases for the message field.
var messageFields = map[string]bool{
	"message": true, "msg": true, "body": true, "text": true,
}

// ExtractHighlightTerms walks the query AST and collects terms for highlighting.
func ExtractHighlightTerms(node *query.Node) []HighlightTerm {
	if node == nil {
		return nil
	}
	var terms []HighlightTerm
	collectTerms(node, &terms)
	return terms
}

func collectTerms(node *query.Node, terms *[]HighlightTerm) {
	switch node.Type {
	case query.NodeFullText:
		*terms = append(*terms, HighlightTerm{Text: node.Value})
	case query.NodeFieldMatch:
		*terms = append(*terms, HighlightTerm{Text: node.Value, Field: node.Field})
	case query.NodeFieldRegex:
		if re, err := regexp.Compile("(?i)" + node.Value); err == nil {
			*terms = append(*terms, HighlightTerm{Pattern: re, Field: node.Field})
		}
	case query.NodeAnd, query.NodeOr:
		collectTerms(node.Left, terms)
		collectTerms(node.Right, terms)
	case query.NodeNot:
		// Don't highlight negated terms — they represent exclusions.
	}
}

type matchRange struct {
	start, end int
}

// fieldMatches reports whether a term targeting termField should highlight
// text displayed under displayField.
func fieldMatches(termField, displayField string) bool {
	if termField == displayField {
		return true
	}
	// Handle message field aliases: msg:foo should highlight in the message display.
	if messageFields[termField] && messageFields[displayField] {
		return true
	}
	return false
}

func findMatches(text string, terms []HighlightTerm, field string) []matchRange {
	var matches []matchRange

	for _, term := range terms {
		// Full-text terms (Field=="") match everywhere.
		// Field-specific terms only match their field.
		if term.Field != "" && !fieldMatches(term.Field, field) {
			continue
		}

		if term.Pattern != nil {
			for _, loc := range term.Pattern.FindAllStringIndex(text, -1) {
				matches = append(matches, matchRange{loc[0], loc[1]})
			}
		} else if term.Text != "" {
			lower := strings.ToLower(text)
			search := strings.ToLower(term.Text)
			pos := 0
			for {
				idx := strings.Index(lower[pos:], search)
				if idx < 0 {
					break
				}
				abs := pos + idx
				matches = append(matches, matchRange{abs, abs + len(search)})
				pos = abs + len(search)
			}
		}
	}

	if len(matches) == 0 {
		return nil
	}

	// Sort by start position.
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].start < matches[j].start
	})

	// Merge overlapping ranges.
	merged := []matchRange{matches[0]}
	for _, m := range matches[1:] {
		last := &merged[len(merged)-1]
		if m.start <= last.end {
			if m.end > last.end {
				last.end = m.end
			}
		} else {
			merged = append(merged, m)
		}
	}

	return merged
}

// highlightText renders text with matching portions highlighted using StyleMatch.
// baseStyle is used for non-matching portions. field identifies which field the
// text belongs to (for field-specific term matching).
func highlightText(text string, terms []HighlightTerm, baseStyle lipgloss.Style, field string) string {
	if len(terms) == 0 || text == "" {
		return baseStyle.Render(text)
	}

	matches := findMatches(text, terms, field)
	if len(matches) == 0 {
		return baseStyle.Render(text)
	}

	var b strings.Builder
	pos := 0
	for _, m := range matches {
		if pos < m.start {
			b.WriteString(baseStyle.Render(text[pos:m.start]))
		}
		b.WriteString(StyleMatch.Render(text[m.start:m.end]))
		pos = m.end
	}
	if pos < len(text) {
		b.WriteString(baseStyle.Render(text[pos:]))
	}
	return b.String()
}
