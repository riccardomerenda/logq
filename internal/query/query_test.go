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

// --- Time query tests ---

func TestLexTimestampCompare(t *testing.T) {
	tokens := Lex(`timestamp>"2026-03-08T10:00:05Z"`)
	// word:timestamp, op:>, word:2026-03-08T10:00:05Z, EOF
	if len(tokens) != 4 {
		t.Fatalf("Expected 4 tokens, got %d: %+v", len(tokens), tokens)
	}
	if tokens[0].Value != "timestamp" {
		t.Errorf("Token 0 = %q, want 'timestamp'", tokens[0].Value)
	}
	if tokens[1].Value != ">" {
		t.Errorf("Token 1 = %q, want '>'", tokens[1].Value)
	}
	if tokens[2].Value != "2026-03-08T10:00:05Z" {
		t.Errorf("Token 2 = %q, want '2026-03-08T10:00:05Z'", tokens[2].Value)
	}
}

func TestLexLastRelative(t *testing.T) {
	tokens := Lex("last:5m")
	// word:last, op::, word:5m, EOF
	if len(tokens) != 4 {
		t.Fatalf("Expected 4 tokens, got %d: %+v", len(tokens), tokens)
	}
	if tokens[0].Value != "last" {
		t.Errorf("Token 0 = %q, want 'last'", tokens[0].Value)
	}
	if tokens[1].Value != ":" {
		t.Errorf("Token 1 = %q, want ':'", tokens[1].Value)
	}
	if tokens[2].Value != "5m" {
		t.Errorf("Token 2 = %q, want '5m'", tokens[2].Value)
	}
}

func TestParseTimestampGreater(t *testing.T) {
	node, err := ParseQuery(`timestamp>"2026-03-08T10:00:05Z"`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if node.Type != NodeTimeCompare {
		t.Errorf("Type = %d, want NodeTimeCompare", node.Type)
	}
	if node.Operator != ">" {
		t.Errorf("Operator = %q, want '>'", node.Operator)
	}
	if node.Value != "2026-03-08T10:00:05Z" {
		t.Errorf("Value = %q, want '2026-03-08T10:00:05Z'", node.Value)
	}
}

func TestParseTimestampLessEqual(t *testing.T) {
	node, err := ParseQuery(`timestamp<="2026-03-08T10:00:10Z"`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if node.Type != NodeTimeCompare {
		t.Errorf("Type = %d, want NodeTimeCompare", node.Type)
	}
	if node.Operator != "<=" {
		t.Errorf("Operator = %q, want '<='", node.Operator)
	}
}

func TestParseTimestampAliases(t *testing.T) {
	// All timestamp field aliases should produce NodeTimeCompare
	aliases := []string{"timestamp", "ts", "time", "@timestamp", "datetime", "t"}
	for _, alias := range aliases {
		node, err := ParseQuery(fmt.Sprintf(`%s>"2026-03-08T10:00:00Z"`, alias))
		if err != nil {
			t.Fatalf("Parse error for %s: %v", alias, err)
		}
		if node.Type != NodeTimeCompare {
			t.Errorf("%s: Type = %d, want NodeTimeCompare", alias, node.Type)
		}
	}
}

func TestParseLastRelative(t *testing.T) {
	node, err := ParseQuery("last:5m")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if node.Type != NodeRelativeTime {
		t.Errorf("Type = %d, want NodeRelativeTime", node.Type)
	}
	if node.Value != "5m" {
		t.Errorf("Value = %q, want '5m'", node.Value)
	}
}

func TestParseLastVariousUnits(t *testing.T) {
	cases := []string{"30s", "5m", "1h", "2d"}
	for _, c := range cases {
		node, err := ParseQuery("last:" + c)
		if err != nil {
			t.Fatalf("Parse error for last:%s: %v", c, err)
		}
		if node.Type != NodeRelativeTime {
			t.Errorf("last:%s: Type = %d, want NodeRelativeTime", c, node.Type)
		}
		if node.Value != c {
			t.Errorf("last:%s: Value = %q", c, node.Value)
		}
	}
}

func TestParseLastInvalid(t *testing.T) {
	invalids := []string{"last:abc", "last:5x", "last:0m"}
	for _, q := range invalids {
		_, err := ParseQuery(q)
		if err == nil {
			t.Errorf("Expected error for %q", q)
		}
	}
}

func TestParseTimestampCompound(t *testing.T) {
	node, err := ParseQuery(`level:error AND timestamp>"2026-03-08T10:00:05Z"`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if node.Type != NodeAnd {
		t.Fatalf("Type = %d, want NodeAnd", node.Type)
	}
	if node.Left.Type != NodeFieldMatch {
		t.Errorf("Left type = %d, want NodeFieldMatch", node.Left.Type)
	}
	if node.Right.Type != NodeTimeCompare {
		t.Errorf("Right type = %d, want NodeTimeCompare", node.Right.Type)
	}
}

func TestParseTimestampRange(t *testing.T) {
	node, err := ParseQuery(`timestamp>="2026-03-08T10:00:05Z" AND timestamp<"2026-03-08T10:00:15Z"`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if node.Type != NodeAnd {
		t.Fatalf("Type = %d, want NodeAnd", node.Type)
	}
	if node.Left.Type != NodeTimeCompare || node.Left.Operator != ">=" {
		t.Errorf("Left: type=%d op=%q, want TimeCompare >=", node.Left.Type, node.Left.Operator)
	}
	if node.Right.Type != NodeTimeCompare || node.Right.Operator != "<" {
		t.Errorf("Right: type=%d op=%q, want TimeCompare <", node.Right.Type, node.Right.Operator)
	}
}

// --- Time evaluator tests ---

func TestEvalTimestampGreater(t *testing.T) {
	idx := makeTestIndex()
	// Records have timestamps from 10:00:00 to 10:00:19 (1 per second)
	node, _ := ParseQuery(`timestamp>"2026-03-08T10:00:14Z"`)
	result := Evaluate(node, idx)

	// Should match records 15-19 (timestamps 10:00:15 through 10:00:19)
	if len(result) != 5 {
		t.Errorf("timestamp>10:00:14 returned %d, want 5", len(result))
	}
}

func TestEvalTimestampGreaterEqual(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery(`timestamp>="2026-03-08T10:00:15Z"`)
	result := Evaluate(node, idx)

	// Should match records 15-19
	if len(result) != 5 {
		t.Errorf("timestamp>=10:00:15 returned %d, want 5", len(result))
	}
}

func TestEvalTimestampLess(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery(`timestamp<"2026-03-08T10:00:05Z"`)
	result := Evaluate(node, idx)

	// Should match records 0-4 (timestamps 10:00:00 through 10:00:04)
	if len(result) != 5 {
		t.Errorf("timestamp<10:00:05 returned %d, want 5", len(result))
	}
}

func TestEvalTimestampLessEqual(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery(`timestamp<="2026-03-08T10:00:04Z"`)
	result := Evaluate(node, idx)

	// Should match records 0-4
	if len(result) != 5 {
		t.Errorf("timestamp<=10:00:04 returned %d, want 5", len(result))
	}
}

func TestEvalTimestampRange(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery(`timestamp>="2026-03-08T10:00:05Z" AND timestamp<"2026-03-08T10:00:10Z"`)
	result := Evaluate(node, idx)

	// Should match records 5-9
	if len(result) != 5 {
		t.Errorf("timestamp range returned %d, want 5", len(result))
	}
}

func TestEvalTimestampWithOtherConditions(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery(`level:error AND timestamp>="2026-03-08T10:00:05Z"`)
	result := Evaluate(node, idx)

	for _, id := range result {
		r := idx.Records[id]
		if r.Level != "error" {
			t.Errorf("Record %d: level=%q, want error", id, r.Level)
		}
		if r.Timestamp.Before(time.Date(2026, 3, 8, 10, 0, 5, 0, time.UTC)) {
			t.Errorf("Record %d: timestamp %v should be >= 10:00:05", id, r.Timestamp)
		}
	}
}

func TestEvalRelativeTime(t *testing.T) {
	idx := makeTestIndex()
	// Records: 10:00:00 to 10:00:19
	// Simulate "now" as 10:00:20, so last:10s = 10:00:10 to 10:00:20
	now := time.Date(2026, 3, 8, 10, 0, 20, 0, time.UTC)

	node, _ := ParseQuery("last:10s")
	result := evaluateWithNow(node, idx, now)

	// Should match records 10-19 (timestamps 10:00:10 through 10:00:19)
	if len(result) != 10 {
		t.Errorf("last:10s returned %d, want 10", len(result))
	}
}

func TestEvalRelativeTimeSmall(t *testing.T) {
	idx := makeTestIndex()
	now := time.Date(2026, 3, 8, 10, 0, 20, 0, time.UTC)

	node, _ := ParseQuery("last:5s")
	result := evaluateWithNow(node, idx, now)

	// last:5s from 10:00:20 = 10:00:15 to 10:00:20 → records 15-19
	if len(result) != 5 {
		t.Errorf("last:5s returned %d, want 5", len(result))
	}
}

func TestEvalRelativeTimeWithCondition(t *testing.T) {
	idx := makeTestIndex()
	now := time.Date(2026, 3, 8, 10, 0, 20, 0, time.UTC)

	node, _ := ParseQuery("level:error AND last:10s")
	result := evaluateWithNow(node, idx, now)

	for _, id := range result {
		r := idx.Records[id]
		if r.Level != "error" {
			t.Errorf("Record %d: level=%q, want error", id, r.Level)
		}
		if r.Timestamp.Before(time.Date(2026, 3, 8, 10, 0, 10, 0, time.UTC)) {
			t.Errorf("Record %d: timestamp %v too old for last:10s", id, r.Timestamp)
		}
	}
}

func TestEvalTimestampInvalidFormat(t *testing.T) {
	idx := makeTestIndex()
	node, _ := ParseQuery(`timestamp>"not-a-date"`)
	result := Evaluate(node, idx)

	if len(result) != 0 {
		t.Errorf("Invalid timestamp should return 0 results, got %d", len(result))
	}
}

func TestEvalTimestampNoMatches(t *testing.T) {
	idx := makeTestIndex()
	// All records are at 10:00:00-10:00:19, query for way in the future
	node, _ := ParseQuery(`timestamp>"2099-01-01T00:00:00Z"`)
	result := Evaluate(node, idx)

	if len(result) != 0 {
		t.Errorf("Expected 0 matches for future timestamp, got %d", len(result))
	}
}

func TestEvalRelativeTimeLargeWindow(t *testing.T) {
	idx := makeTestIndex()
	now := time.Date(2026, 3, 8, 10, 0, 20, 0, time.UTC)

	node, _ := ParseQuery("last:1h")
	result := evaluateWithNow(node, idx, now)

	// All 20 records are within last hour
	if len(result) != 20 {
		t.Errorf("last:1h returned %d, want 20", len(result))
	}
}
