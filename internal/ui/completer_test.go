package ui

import (
	"testing"

	"github.com/riccardomerenda/logq/internal/index"
	"github.com/riccardomerenda/logq/internal/parser"
)

func TestExtractCompletionContext(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		cursor int
		mode   completeMode
		prefix string
		field  string
	}{
		{"empty", "", 0, completeNone, "", ""},
		{"single char", "l", 1, completeFieldName, "l", ""},
		{"partial field", "lev", 3, completeFieldName, "lev", ""},
		{"full field with colon", "level:", 6, completeFieldValue, "", "level"},
		{"field with partial value", "level:err", 9, completeFieldValue, "err", "level"},
		{"after space", "level:error ", 12, completeNone, "", ""},
		{"second term", "level:error se", 14, completeFieldName, "se", ""},
		{"after AND", "level:error AND la", 18, completeFieldName, "la", ""},
		{"numeric operator", "latency>", 8, completeNone, "", ""},
		{"regex operator", "message~", 8, completeNone, "", ""},
		{"gte operator", "latency>=", 9, completeNone, "", ""},
		{"after paren", "(lev", 4, completeFieldName, "lev", ""},
		{"cursor in middle", "level:error", 5, completeFieldName, "level", ""},
		{"quoted value no complete", `message:"hello `, 15, completeNone, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := extractCompletionContext(tt.text, tt.cursor)
			if ctx.mode != tt.mode {
				t.Errorf("mode = %d, want %d", ctx.mode, tt.mode)
			}
			if ctx.prefix != tt.prefix {
				t.Errorf("prefix = %q, want %q", ctx.prefix, tt.prefix)
			}
			if ctx.field != tt.field {
				t.Errorf("field = %q, want %q", ctx.field, tt.field)
			}
		})
	}
}

func makeTestIndex() *index.Index {
	records := []parser.Record{
		{
			LineNumber: 1,
			Level:      "error",
			Message:    "token expired",
			Fields: map[string]string{
				"level": "error", "service": "auth", "message": "token expired",
				"latency": "12", "user_id": "u_882",
			},
		},
		{
			LineNumber: 2,
			Level:      "info",
			Message:    "request ok",
			Fields: map[string]string{
				"level": "info", "service": "api", "message": "request ok",
				"method": "GET", "latency": "45",
			},
		},
		{
			LineNumber: 3,
			Level:      "warn",
			Message:    "slow query",
			Fields: map[string]string{
				"level": "warn", "service": "db", "message": "slow query",
				"latency": "1523",
			},
		},
	}
	return index.Build(records)
}

func TestComputeCandidatesFieldName(t *testing.T) {
	idx := makeTestIndex()

	ctx := completionContext{mode: completeFieldName, prefix: "lev"}
	candidates := computeCandidates(ctx, idx)

	if len(candidates) != 1 || candidates[0] != "level" {
		t.Errorf("expected [level], got %v", candidates)
	}
}

func TestComputeCandidatesFieldNameMultiple(t *testing.T) {
	idx := makeTestIndex()

	ctx := completionContext{mode: completeFieldName, prefix: "l"}
	candidates := computeCandidates(ctx, idx)

	// Should include "latency", "level", "last" (keyword)
	found := map[string]bool{}
	for _, c := range candidates {
		found[c] = true
	}
	if !found["latency"] || !found["level"] || !found["last"] {
		t.Errorf("expected latency, level, last in %v", candidates)
	}
}

func TestComputeCandidatesFieldValue(t *testing.T) {
	idx := makeTestIndex()

	ctx := completionContext{mode: completeFieldValue, prefix: "err", field: "level"}
	candidates := computeCandidates(ctx, idx)

	if len(candidates) != 1 || candidates[0] != "error" {
		t.Errorf("expected [error], got %v", candidates)
	}
}

func TestComputeCandidatesFieldValueAll(t *testing.T) {
	idx := makeTestIndex()

	ctx := completionContext{mode: completeFieldValue, prefix: "", field: "level"}
	candidates := computeCandidates(ctx, idx)

	// Should include all level values
	if len(candidates) < 3 {
		t.Errorf("expected at least 3 level values, got %v", candidates)
	}
}

func TestComputeCandidatesKeywords(t *testing.T) {
	idx := makeTestIndex()

	ctx := completionContext{mode: completeFieldName, prefix: "AN"}
	candidates := computeCandidates(ctx, idx)

	if len(candidates) != 1 || candidates[0] != "AND" {
		t.Errorf("expected [AND], got %v", candidates)
	}
}

func TestComputeCandidatesNone(t *testing.T) {
	idx := makeTestIndex()

	ctx := completionContext{mode: completeNone}
	candidates := computeCandidates(ctx, idx)

	if candidates != nil {
		t.Errorf("expected nil, got %v", candidates)
	}
}

func TestCompleterGhostSuffix(t *testing.T) {
	c := Completer{
		candidates: []string{"level", "latency"},
		prefix:     "lev",
		mode:       completeFieldName,
	}

	ghost := c.GhostSuffix()
	if ghost != "el:" {
		t.Errorf("ghost = %q, want \"el:\"", ghost)
	}
}

func TestCompleterGhostSuffixValue(t *testing.T) {
	c := Completer{
		candidates: []string{"error"},
		prefix:     "err",
		mode:       completeFieldValue,
	}

	ghost := c.GhostSuffix()
	if ghost != "or" {
		t.Errorf("ghost = %q, want \"or\"", ghost)
	}
}

func TestCompleterCycle(t *testing.T) {
	c := Completer{
		candidates: []string{"alpha", "beta", "gamma"},
		prefix:     "",
	}

	if c.Current() != "alpha" {
		t.Errorf("initial = %q, want \"alpha\"", c.Current())
	}
	c.Next()
	if c.Current() != "beta" {
		t.Errorf("after Next = %q, want \"beta\"", c.Current())
	}
	c.Next()
	c.Next()
	if c.Current() != "alpha" {
		t.Errorf("after wrap = %q, want \"alpha\"", c.Current())
	}
}

func TestCompleterReset(t *testing.T) {
	c := Completer{
		candidates: []string{"level"},
		prefix:     "lev",
		mode:       completeFieldName,
	}
	c.Reset()

	if c.HasCandidates() {
		t.Error("expected no candidates after reset")
	}
	if c.GhostSuffix() != "" {
		t.Error("expected empty ghost after reset")
	}
}

func TestFieldNames(t *testing.T) {
	idx := makeTestIndex()
	names := idx.FieldNames()

	if len(names) == 0 {
		t.Fatal("expected field names")
	}

	// Should be sorted
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("not sorted: %q before %q", names[i-1], names[i])
		}
	}

	// Should contain known fields
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	for _, want := range []string{"level", "service", "message", "latency"} {
		if !found[want] {
			t.Errorf("missing field %q in %v", want, names)
		}
	}
}

func TestFieldValues(t *testing.T) {
	idx := makeTestIndex()
	vals := idx.FieldValues("service")

	if vals == nil {
		t.Fatal("expected service values")
	}

	found := map[string]bool{}
	for _, v := range vals {
		found[v] = true
	}
	for _, want := range []string{"auth", "api", "db"} {
		if !found[want] {
			t.Errorf("missing value %q in %v", want, vals)
		}
	}
}

func TestFieldValuesHighCardinality(t *testing.T) {
	// Create index with many unique values
	records := make([]parser.Record, 100)
	for i := range records {
		records[i] = parser.Record{
			LineNumber: i,
			Fields:     map[string]string{"id": string(rune('a' + i%26)) + string(rune('0'+i/26))},
		}
	}
	idx := index.Build(records)

	vals := idx.FieldValues("id")
	if vals != nil {
		t.Errorf("expected nil for high-cardinality field, got %d values", len(vals))
	}
}
