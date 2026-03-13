package parser

import (
	"strings"
	"testing"
	"time"
)

func TestParseJSON(t *testing.T) {
	line := `{"timestamp":"2026-03-08T10:00:01Z","level":"info","service":"api","message":"request started","method":"GET","path":"/users","latency":45}`
	r := Parse(line, 1)

	if r.LineNumber != 1 {
		t.Errorf("LineNumber = %d, want 1", r.LineNumber)
	}
	if r.Level != "info" {
		t.Errorf("Level = %q, want %q", r.Level, "info")
	}
	if r.Message != "request started" {
		t.Errorf("Message = %q, want %q", r.Message, "request started")
	}
	if r.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if r.Fields["service"] != "api" {
		t.Errorf("Fields[service] = %q, want %q", r.Fields["service"], "api")
	}
	if r.Fields["method"] != "GET" {
		t.Errorf("Fields[method] = %q, want %q", r.Fields["method"], "GET")
	}
	if r.Fields["latency"] != "45" {
		t.Errorf("Fields[latency] = %q, want %q", r.Fields["latency"], "45")
	}
	if r.Raw != line {
		t.Error("Raw should equal original line")
	}
}

func TestParseJSONNested(t *testing.T) {
	line := `{"timestamp":"2026-03-08T10:00:01Z","level":"error","message":"fail","request":{"method":"POST","headers":{"host":"example.com"}}}`
	r := Parse(line, 1)

	if r.Fields["request.method"] != "POST" {
		t.Errorf("Fields[request.method] = %q, want %q", r.Fields["request.method"], "POST")
	}
	if r.Fields["request.headers.host"] != "example.com" {
		t.Errorf("Fields[request.headers.host] = %q, want %q", r.Fields["request.headers.host"], "example.com")
	}
}

func TestParseJSONArrayField(t *testing.T) {
	line := `{"message":"test","tags":["web","api"]}`
	r := Parse(line, 1)

	if r.Fields["tags"] != `["web","api"]` {
		t.Errorf("Fields[tags] = %q, want %q", r.Fields["tags"], `["web","api"]`)
	}
}

func TestParseJSONNullField(t *testing.T) {
	line := `{"message":"test","extra":null}`
	r := Parse(line, 1)

	if _, ok := r.Fields["extra"]; ok {
		t.Error("null fields should be skipped")
	}
}

func TestParseJSONBoolField(t *testing.T) {
	line := `{"message":"test","success":true}`
	r := Parse(line, 1)

	if r.Fields["success"] != "true" {
		t.Errorf("Fields[success] = %q, want %q", r.Fields["success"], "true")
	}
}

func TestParseLogfmt(t *testing.T) {
	line := `ts=2026-03-08T10:00:01Z level=info service=api msg="request started" method=GET path=/users latency=45`
	r := Parse(line, 2)

	if r.Level != "info" {
		t.Errorf("Level = %q, want %q", r.Level, "info")
	}
	if r.Message != "request started" {
		t.Errorf("Message = %q, want %q", r.Message, "request started")
	}
	if r.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if r.Fields["service"] != "api" {
		t.Errorf("Fields[service] = %q, want %q", r.Fields["service"], "api")
	}
	if r.Fields["latency"] != "45" {
		t.Errorf("Fields[latency] = %q, want %q", r.Fields["latency"], "45")
	}
}

func TestParsePlainText(t *testing.T) {
	line := "This is just a plain log message with no structure"
	r := Parse(line, 3)

	if r.Message != line {
		t.Errorf("Message = %q, want %q", r.Message, line)
	}
	if r.Fields["message"] != line {
		t.Errorf("Fields[message] = %q, want %q", r.Fields["message"], line)
	}
	if r.Raw != line {
		t.Error("Raw should equal original line")
	}
}

func TestParseMalformedJSON(t *testing.T) {
	line := `{broken json here`
	r := Parse(line, 1)

	// Should fall through to plain text, not panic
	if r.Message != line {
		t.Errorf("Malformed JSON should fall back to plain text, got Message = %q", r.Message)
	}
}

func TestParseEmptyLine(t *testing.T) {
	r := Parse("", 1)

	if r.LineNumber != 1 {
		t.Errorf("LineNumber = %d, want 1", r.LineNumber)
	}
	if r.Fields == nil {
		t.Error("Fields should not be nil")
	}
}

func TestParseLongLine(t *testing.T) {
	// 1MB line — should not panic
	long := `{"message":"` + strings.Repeat("x", 1_000_000) + `"}`
	r := Parse(long, 1)

	if r.Message == "" {
		t.Error("Should parse long lines without panic")
	}
}

func TestNormalizeLevel(t *testing.T) {
	tests := map[string]string{
		"debug":       "debug",
		"DEBUG":       "debug",
		"DBG":         "debug",
		"trace":       "debug",
		"info":        "info",
		"INFO":        "info",
		"INF":         "info",
		"information": "info",
		"warn":        "warn",
		"WARNING":     "warn",
		"WRN":         "warn",
		"error":       "error",
		"ERROR":       "error",
		"ERR":         "error",
		"fatal":       "fatal",
		"FATAL":       "fatal",
		"critical":    "fatal",
		"CRIT":        "fatal",
		"panic":       "fatal",
		"custom":      "custom",
	}

	for input, want := range tests {
		got := NormalizeLevel(input)
		if got != want {
			t.Errorf("NormalizeLevel(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseTimestampFormats(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
	}{
		{
			"2026-03-08T10:00:01Z",
			time.Date(2026, 3, 8, 10, 0, 1, 0, time.UTC),
		},
		{
			"2026-03-08T10:00:01.123Z",
			time.Date(2026, 3, 8, 10, 0, 1, 123000000, time.UTC),
		},
		{
			"2026-03-08 10:00:01.000",
			time.Date(2026, 3, 8, 10, 0, 1, 0, time.UTC),
		},
		{
			"2026-03-08 10:00:01",
			time.Date(2026, 3, 8, 10, 0, 1, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		got, err := ParseTimestamp(tt.input)
		if err != nil {
			t.Errorf("ParseTimestamp(%q) error: %v", tt.input, err)
			continue
		}
		if !got.Equal(tt.want) {
			t.Errorf("ParseTimestamp(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseTimestampUnixEpoch(t *testing.T) {
	// Unix seconds
	got, err := ParseTimestamp("1773064801")
	if err != nil {
		t.Fatalf("ParseTimestamp(epoch seconds) error: %v", err)
	}
	if got.Year() < 2025 {
		t.Errorf("Expected recent year, got %d", got.Year())
	}

	// Unix milliseconds
	got, err = ParseTimestamp("1773064801000")
	if err != nil {
		t.Fatalf("ParseTimestamp(epoch ms) error: %v", err)
	}
	if got.Year() < 2025 {
		t.Errorf("Expected recent year, got %d", got.Year())
	}
}

func TestParseTimestampInvalid(t *testing.T) {
	_, err := ParseTimestamp("not a timestamp")
	if err == nil {
		t.Error("Expected error for invalid timestamp")
	}
}

func TestAutoDetectionOrder(t *testing.T) {
	// JSON takes priority
	jsonLine := `{"level":"info","message":"hello","foo":"bar"}`
	r := Parse(jsonLine, 1)
	if r.Level != "info" {
		t.Errorf("JSON should be detected first, got Level=%q", r.Level)
	}

	// logfmt detected when not JSON
	logfmtLine := `level=error msg="something failed" service=api`
	r = Parse(logfmtLine, 2)
	if r.Level != "error" {
		t.Errorf("logfmt should be detected, got Level=%q", r.Level)
	}

	// Plain text fallback
	plainLine := `Just a regular log message`
	r = Parse(plainLine, 3)
	if r.Message != plainLine {
		t.Errorf("Plain text fallback failed, got Message=%q", r.Message)
	}
}

