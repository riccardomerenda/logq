package trace

import (
	"testing"

	"github.com/riccardomerenda/logq/internal/parser"
)

func TestDetectIDFields_ByName(t *testing.T) {
	r := parser.Record{
		Fields: map[string]string{
			"trace_id": "abc-123",
			"level":    "error",
			"message":  "timeout",
		},
	}
	ids := DetectIDFields(r, DefaultIDFields)
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID field, got %d", len(ids))
	}
	if ids[0].Name != "trace_id" || ids[0].Value != "abc-123" {
		t.Errorf("unexpected ID: %+v", ids[0])
	}
}

func TestDetectIDFields_MultipleByName(t *testing.T) {
	r := parser.Record{
		Fields: map[string]string{
			"trace_id":   "abc-123",
			"request_id": "def-456",
			"level":      "info",
		},
	}
	ids := DetectIDFields(r, DefaultIDFields)
	if len(ids) != 2 {
		t.Fatalf("expected 2 ID fields, got %d", len(ids))
	}
}

func TestDetectIDFields_ByValueUUID(t *testing.T) {
	r := parser.Record{
		Fields: map[string]string{
			"my_custom_field": "550e8400-e29b-41d4-a716-446655440000",
			"level":           "error",
		},
	}
	ids := DetectIDFields(r, DefaultIDFields)
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID field, got %d", len(ids))
	}
	if ids[0].Name != "my_custom_field" {
		t.Errorf("expected my_custom_field, got %s", ids[0].Name)
	}
}

func TestDetectIDFields_ByValueHex(t *testing.T) {
	r := parser.Record{
		Fields: map[string]string{
			"zipkin_id": "463ac35c9f6413ad48485a3953bb6124",
			"level":     "info",
		},
	}
	ids := DetectIDFields(r, DefaultIDFields)
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID field, got %d", len(ids))
	}
}

func TestDetectIDFields_NumericNotDetected(t *testing.T) {
	r := parser.Record{
		Fields: map[string]string{
			"latency": "523",
			"level":   "error",
		},
	}
	ids := DetectIDFields(r, DefaultIDFields)
	if len(ids) != 0 {
		t.Fatalf("expected 0 ID fields, got %d", len(ids))
	}
}

func TestDetectIDFields_EmptyValueSkipped(t *testing.T) {
	r := parser.Record{
		Fields: map[string]string{
			"trace_id": "",
			"level":    "error",
		},
	}
	ids := DetectIDFields(r, DefaultIDFields)
	if len(ids) != 0 {
		t.Fatalf("expected 0 ID fields, got %d", len(ids))
	}
}

func TestDetectIDFields_CamelCase(t *testing.T) {
	r := parser.Record{
		Fields: map[string]string{
			"traceId": "abc-123",
		},
	}
	ids := DetectIDFields(r, DefaultIDFields)
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID field, got %d", len(ids))
	}
	if ids[0].Name != "traceId" {
		t.Errorf("expected original field name 'traceId', got %s", ids[0].Name)
	}
}

func TestDetectIDFields_HyphenatedName(t *testing.T) {
	r := parser.Record{
		Fields: map[string]string{
			"x-request-id": "req-789",
		},
	}
	ids := DetectIDFields(r, DefaultIDFields)
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID field, got %d", len(ids))
	}
}

func TestDetectIDFields_DottedName(t *testing.T) {
	r := parser.Record{
		Fields: map[string]string{
			"trace.id": "abc-123",
		},
	}
	ids := DetectIDFields(r, DefaultIDFields)
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID field, got %d", len(ids))
	}
}

func TestDetectIDFields_NameBeforeValue(t *testing.T) {
	r := parser.Record{
		Fields: map[string]string{
			"trace_id":    "simple-value",
			"custom_data": "550e8400-e29b-41d4-a716-446655440000",
		},
	}
	ids := DetectIDFields(r, DefaultIDFields)
	if len(ids) != 2 {
		t.Fatalf("expected 2 ID fields, got %d", len(ids))
	}
	// Name-matched should come first
	if ids[0].Name != "trace_id" {
		t.Errorf("expected name-matched field first, got %s", ids[0].Name)
	}
}

func TestDetectIDFields_CustomConfig(t *testing.T) {
	r := parser.Record{
		Fields: map[string]string{
			"my_trace": "abc-123",
			"level":    "error",
		},
	}
	ids := DetectIDFields(r, []string{"my_trace"})
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID field, got %d", len(ids))
	}
}

func TestNormalizeFieldName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"trace_id", "trace_id"},
		{"traceId", "trace_id"},
		{"TraceID", "trace_id"},
		{"x-request-id", "x_request_id"},
		{"trace.id", "trace_id"},
		{"TRACE_ID", "trace_id"},
		{"spanId", "span_id"},
	}
	for _, tt := range tests {
		got := NormalizeFieldName(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeFieldName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsUUIDLike(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"abc-123", false},
		{"not-a-uuid", false},
		{"550e8400e29b41d4a716446655440000", false}, // no dashes
	}
	for _, tt := range tests {
		if got := IsUUIDLike(tt.input); got != tt.want {
			t.Errorf("IsUUIDLike(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsHexID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"463ac35c9f6413ad48485a3953bb6124", true},
		{"463ac35c9f6413ad", true},  // 16 chars
		{"463ac35c9f6413a", false},  // 15 chars
		{"not-hex-at-all!", false},
	}
	for _, tt := range tests {
		if got := IsHexID(tt.input); got != tt.want {
			t.Errorf("IsHexID(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestBuildQuery(t *testing.T) {
	tests := []struct {
		field, value, want string
	}{
		{"trace_id", "abc-123", `trace_id:abc-123`},
		{"trace_id", "abc 123", `trace_id:"abc 123"`},
		{"trace_id", "550e8400-e29b-41d4-a716-446655440000", `trace_id:550e8400-e29b-41d4-a716-446655440000`},
		{"field", "val:ue", `field:"val:ue"`},
	}
	for _, tt := range tests {
		got := BuildQuery(tt.field, tt.value)
		if got != tt.want {
			t.Errorf("BuildQuery(%q, %q) = %q, want %q", tt.field, tt.value, got, tt.want)
		}
	}
}
