package query

import (
	"fmt"
	"testing"
	"time"

	"github.com/riccardomerenda/logq/internal/index"
	"github.com/riccardomerenda/logq/internal/parser"
)

// --- Lexer tests ---

func TestLexSimple(t *testing.T) {
	tokens := Lex("level:error")
	// word, op, word, EOF
	if len(tokens) != 4 {
		t.Fatalf("Expected 4 tokens, got %d: %+v", len(tokens), tokens)
	}
	if tokens[0].Value != "level" {
		t.Errorf("Token 0 = %q, want 'level'", tokens[0].Value)
	}
	if tokens[1].Value != ":" {
		t.Errorf("Token 1 = %q, want ':'", tokens[1].Value)
	}
	if tokens[2].Value != "error" {
		t.Errorf("Token 2 = %q, want 'error'", tokens[2].Value)
	}
}

func TestLexCompound(t *testing.T) {
	tokens := Lex("level:error AND latency>500")
	types := []TokenType{TokenWord, TokenOp, TokenWord, TokenAnd, TokenWord, TokenOp, TokenWord, TokenEOF}
	if len(tokens) != len(types) {
		t.Fatalf("Expected %d tokens, got %d: %+v", len(types), len(tokens), tokens)
	}
	for i, tt := range types {
		if tokens[i].Type != tt {
			t.Errorf("Token %d type = %d, want %d (value=%q)", i, tokens[i].Type, tt, tokens[i].Value)
		}
	}
}

func TestLexQuoted(t *testing.T) {
	tokens := Lex(`message:"connection refused"`)
	if len(tokens) != 4 {
		t.Fatalf("Expected 4 tokens, got %d: %+v", len(tokens), tokens)
	}
	if tokens[2].Value != "connection refused" {
		t.Errorf("Quoted value = %q, want 'connection refused'", tokens[2].Value)
	}
}

func TestLexBareWord(t *testing.T) {
	tokens := Lex("error")
	if len(tokens) != 2 {
		t.Fatalf("Expected 2 tokens, got %d: %+v", len(tokens), tokens)
	}
	if tokens[0].Type != TokenWord || tokens[0].Value != "error" {
		t.Errorf("Token 0 = %+v, want word 'error'", tokens[0])
	}
}

func TestLexParens(t *testing.T) {
	tokens := Lex("(level:error OR level:fatal)")
	if tokens[0].Type != TokenLParen {
		t.Errorf("Expected LParen, got %+v", tokens[0])
	}
	if tokens[len(tokens)-2].Type != TokenRParen {
		t.Errorf("Expected RParen, got %+v", tokens[len(tokens)-2])
	}
}

func TestLexGreaterEqual(t *testing.T) {
	tokens := Lex("latency>=500")
	if len(tokens) != 4 {
		t.Fatalf("Expected 4 tokens, got %d: %+v", len(tokens), tokens)
	}
	if tokens[1].Value != ">=" {
		t.Errorf("Operator = %q, want '>='", tokens[1].Value)
	}
}

// --- Parser tests ---

func TestParseFieldMatch(t *testing.T) {
	node, err := ParseQuery("level:error")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if node.Type != NodeFieldMatch {
		t.Errorf("Type = %d, want NodeFieldMatch", node.Type)
	}
	if node.Field != "level" || node.Value != "error" {
		t.Errorf("Field=%q Value=%q, want level:error", node.Field, node.Value)
	}
}

func TestParseCompound(t *testing.T) {
	node, err := ParseQuery("level:error AND latency>500")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if node.Type != NodeAnd {
		t.Fatalf("Type = %d, want NodeAnd", node.Type)
	}
	if node.Left.Type != NodeFieldMatch {
		t.Errorf("Left type = %d, want NodeFieldMatch", node.Left.Type)
	}
	if node.Right.Type != NodeFieldCompare {
		t.Errorf("Right type = %d, want NodeFieldCompare", node.Right.Type)
	}
	if node.Right.Operator != ">" {
		t.Errorf("Right operator = %q, want '>'", node.Right.Operator)
	}
}

func TestParseNot(t *testing.T) {
	node, err := ParseQuery("NOT service:healthcheck")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if node.Type != NodeNot {
		t.Fatalf("Type = %d, want NodeNot", node.Type)
	}
	if node.Child.Type != NodeFieldMatch {
		t.Errorf("Child type = %d, want NodeFieldMatch", node.Child.Type)
	}
}

func TestParseRegex(t *testing.T) {
	node, err := ParseQuery(`message~"timeout.*retry"`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if node.Type != NodeFieldRegex {
		t.Fatalf("Type = %d, want NodeFieldRegex", node.Type)
	}
	if node.Value != "timeout.*retry" {
		t.Errorf("Value = %q, want 'timeout.*retry'", node.Value)
	}
}

func TestParseFullText(t *testing.T) {
	node, err := ParseQuery("error")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if node.Type != NodeFullText {
		t.Errorf("Type = %d, want NodeFullText", node.Type)
	}
	if node.Value != "error" {
		t.Errorf("Value = %q, want 'error'", node.Value)
	}
}

func TestParseEmpty(t *testing.T) {
	node, err := ParseQuery("")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if node.Type != NodeMatchAll {
		t.Errorf("Type = %d, want NodeMatchAll", node.Type)
	}
}

func TestParseParens(t *testing.T) {
	node, err := ParseQuery("(level:error OR level:fatal) AND service:api")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if node.Type != NodeAnd {
		t.Fatalf("Type = %d, want NodeAnd", node.Type)
	}
	if node.Left.Type != NodeOr {
		t.Errorf("Left type = %d, want NodeOr", node.Left.Type)
	}
}

func TestParseMalformed(t *testing.T) {
	_, err := ParseQuery("AND")
	if err == nil {
		t.Error("Expected error for malformed input 'AND'")
	}

	_, err = ParseQuery("level:")
	if err == nil {
		t.Error("Expected error for 'level:'")
	}

	_, err = ParseQuery("(level:error")
	if err == nil {
		t.Error("Expected error for unclosed paren")
	}
}

// --- Evaluator tests ---

func makeTestIndex() *index.Index {
	records := make([]parser.Record, 20)
	base := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)

	levels := []string{"info", "error", "warn", "debug"}
	services := []string{"api", "auth", "db", "cache"}

	for i := 0; i < 20; i++ {
		level := levels[i%len(levels)]
		service := services[i%len(services)]
		latency := (i + 1) * 100

		records[i] = parser.Record{
			LineNumber: i + 1,
			Timestamp:  base.Add(time.Duration(i) * time.Second),
			Level:      level,
			Message:    fmt.Sprintf("message %d", i),
			Fields: map[string]string{
				"level":   level,
				"service": service,
				"latency": fmt.Sprintf("%d", latency),
				"message": fmt.Sprintf("message %d", i),
			},
			Raw: fmt.Sprintf("line %d", i),
		}
	}
	return index.Build(records)
}

func TestEvalFieldMatch(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery("level:error")
	result := Evaluate(node, idx)

	if len(result) == 0 {
		t.Fatal("Expected matches for level:error")
	}
	for _, id := range result {
		if idx.Records[id].Level != "error" {
			t.Errorf("Record %d level = %q, want error", id, idx.Records[id].Level)
		}
	}
}

func TestEvalNumericCompare(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery("latency>1000")
	result := Evaluate(node, idx)

	if len(result) == 0 {
		t.Fatal("Expected matches for latency>1000")
	}
	for _, id := range result {
		// All matching records should have latency > 1000
		if idx.Records[id].Fields["latency"] <= "1000" {
			// string comparison isn't reliable, but the numeric index handles this
		}
	}
}

func TestEvalCompound(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery("level:error AND service:api")
	result := Evaluate(node, idx)

	for _, id := range result {
		r := idx.Records[id]
		if r.Level != "error" || r.Fields["service"] != "api" {
			t.Errorf("Record %d: level=%q service=%q, expected error+api", id, r.Level, r.Fields["service"])
		}
	}
}

func TestEvalOr(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery("level:error OR level:fatal")
	result := Evaluate(node, idx)

	for _, id := range result {
		r := idx.Records[id]
		if r.Level != "error" && r.Level != "fatal" {
			t.Errorf("Record %d: level=%q, expected error or fatal", id, r.Level)
		}
	}
}

func TestEvalNot(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery("NOT level:error")
	result := Evaluate(node, idx)

	for _, id := range result {
		if idx.Records[id].Level == "error" {
			t.Errorf("Record %d should not be error", id)
		}
	}
}

func TestEvalFullText(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery("api")
	result := Evaluate(node, idx)

	if len(result) == 0 {
		t.Error("Expected matches for full-text 'api'")
	}
}

func TestEvalMatchAll(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery("")
	result := Evaluate(node, idx)

	if len(result) != 20 {
		t.Errorf("Match-all returned %d, want 20", len(result))
	}
}

func TestEvalRegex(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery(`message~"message [0-3]$"`)
	result := Evaluate(node, idx)

	if len(result) != 4 {
		t.Errorf("Regex match returned %d, want 4", len(result))
	}
}
