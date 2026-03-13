package ui

import (
	"regexp"
	"testing"

	"github.com/riccardomerenda/logq/internal/query"
)

func TestExtractHighlightTerms(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantLen int
	}{
		{"empty query", "", 0},
		{"full-text", "error", 1},
		{"field match", "level:error", 1},
		{"field regex", `message~"timeout.*"`, 1},
		{"AND combines", "error AND timeout", 2},
		{"OR combines", "error OR timeout", 2},
		{"NOT excludes", "NOT error", 0},
		{"mixed", "error AND level:warn", 2},
		{"numeric comparison skipped", "latency>500", 0},
		{"time comparison skipped", `timestamp>"2026-01-01"`, 0},
		{"relative time skipped", "last:5m", 0},
		{"complex", "error AND (service:auth OR message~\"fail.*\")", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := query.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			terms := ExtractHighlightTerms(node)
			if len(terms) != tt.wantLen {
				t.Errorf("got %d terms, want %d", len(terms), tt.wantLen)
			}
		})
	}
}

func TestExtractHighlightTermsContent(t *testing.T) {
	node, _ := query.ParseQuery("error AND service:auth")
	terms := ExtractHighlightTerms(node)

	if len(terms) != 2 {
		t.Fatalf("expected 2 terms, got %d", len(terms))
	}

	// First: full-text "error"
	if terms[0].Text != "error" || terms[0].Field != "" {
		t.Errorf("term[0] = {%q, %q}, want {\"error\", \"\"}", terms[0].Text, terms[0].Field)
	}

	// Second: field match "auth" in "service"
	if terms[1].Text != "auth" || terms[1].Field != "service" {
		t.Errorf("term[1] = {%q, %q}, want {\"auth\", \"service\"}", terms[1].Text, terms[1].Field)
	}
}

func TestExtractHighlightTermsRegex(t *testing.T) {
	node, _ := query.ParseQuery(`message~"timeout.*"`)
	terms := ExtractHighlightTerms(node)

	if len(terms) != 1 {
		t.Fatalf("expected 1 term, got %d", len(terms))
	}
	if terms[0].Pattern == nil {
		t.Fatal("expected regex pattern, got nil")
	}
	if terms[0].Field != "message" {
		t.Errorf("field = %q, want \"message\"", terms[0].Field)
	}
}

func TestFindMatches(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		terms []HighlightTerm
		field string
		want  []matchRange
	}{
		{
			name:  "no terms",
			text:  "hello world",
			terms: nil,
			want:  nil,
		},
		{
			name:  "simple match",
			text:  "connection timeout error",
			terms: []HighlightTerm{{Text: "timeout"}},
			want:  []matchRange{{11, 18}},
		},
		{
			name:  "case insensitive",
			text:  "Connection TIMEOUT Error",
			terms: []HighlightTerm{{Text: "timeout"}},
			want:  []matchRange{{11, 18}},
		},
		{
			name:  "multiple matches",
			text:  "error: another error occurred",
			terms: []HighlightTerm{{Text: "error"}},
			want:  []matchRange{{0, 5}, {15, 20}},
		},
		{
			name:  "overlapping merged",
			text:  "abcdef",
			terms: []HighlightTerm{{Text: "bcd"}, {Text: "cde"}},
			want:  []matchRange{{1, 5}},
		},
		{
			name:  "field-specific skipped",
			text:  "some auth message",
			terms: []HighlightTerm{{Text: "auth", Field: "service"}},
			field: "message",
			want:  nil,
		},
		{
			name:  "field-specific matched",
			text:  "auth-service",
			terms: []HighlightTerm{{Text: "auth", Field: "service"}},
			field: "service",
			want:  []matchRange{{0, 4}},
		},
		{
			name:  "full-text matches any field",
			text:  "auth-service",
			terms: []HighlightTerm{{Text: "auth"}},
			field: "service",
			want:  []matchRange{{0, 4}},
		},
		{
			name:  "regex match",
			text:  "request timeout after 5000ms",
			terms: []HighlightTerm{{Pattern: regexp.MustCompile(`(?i)timeout.*\d+ms`)}},
			want:  []matchRange{{8, 28}},
		},
		{
			name:  "message alias msg matches message",
			text:  "connection failed",
			terms: []HighlightTerm{{Text: "failed", Field: "msg"}},
			field: "message",
			want:  []matchRange{{11, 17}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findMatches(tt.text, tt.terms, tt.field)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d matches, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("match[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestHighlightTextNoTerms(t *testing.T) {
	style := StyleDim
	result := highlightText("hello", nil, style, "")
	expected := style.Render("hello")
	if result != expected {
		t.Errorf("with no terms, expected plain render")
	}
}

func TestHighlightTextNoMatch(t *testing.T) {
	terms := []HighlightTerm{{Text: "xyz"}}
	style := StyleDim
	result := highlightText("hello world", terms, style, "")
	expected := style.Render("hello world")
	if result != expected {
		t.Errorf("with no match, expected plain render")
	}
}

func TestHighlightTextWithMatch(t *testing.T) {
	terms := []HighlightTerm{{Text: "world"}}
	style := StyleDim
	result := highlightText("hello world end", terms, style, "")

	// The result should contain the match styled differently from the rest
	if result == style.Render("hello world end") {
		t.Error("expected highlighted output to differ from plain render")
	}

	// Should contain the match text styled with StyleMatch
	matchPart := StyleMatch.Render("world")
	if !containsSubstring(result, matchPart) {
		t.Error("expected result to contain match-styled 'world'")
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
