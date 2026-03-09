package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/riccardomerenda/logq/internal/parser"
)

func testRecords() []parser.Record {
	return []parser.Record{
		{
			LineNumber: 1,
			Timestamp:  time.Date(2026, 3, 8, 10, 0, 1, 0, time.UTC),
			Level:      "error",
			Message:    "token expired",
			Fields:     map[string]string{"level": "error", "message": "token expired", "service": "auth", "user_id": "u_882"},
			Raw:        `{"level":"error","message":"token expired","service":"auth","user_id":"u_882"}`,
		},
		{
			LineNumber: 2,
			Timestamp:  time.Date(2026, 3, 8, 10, 0, 2, 0, time.UTC),
			Level:      "info",
			Message:    "request ok",
			Fields:     map[string]string{"level": "info", "message": "request ok", "service": "api", "latency": "45"},
			Raw:        `{"level":"info","message":"request ok","service":"api","latency":45}`,
		},
		{
			LineNumber: 3,
			Timestamp:  time.Date(2026, 3, 8, 10, 0, 3, 0, time.UTC),
			Level:      "warn",
			Message:    "slow query",
			Fields:     map[string]string{"level": "warn", "message": "slow query", "service": "db"},
			Raw:        `{"level":"warn","message":"slow query","service":"db"}`,
		},
	}
}

func TestWriteRaw(t *testing.T) {
	records := testRecords()
	var buf bytes.Buffer

	err := Write(&buf, records, []int{0, 2}, FormatRaw)
	if err != nil {
		t.Fatalf("Write raw: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}
	if lines[0] != records[0].Raw {
		t.Errorf("Line 0 = %q, want %q", lines[0], records[0].Raw)
	}
	if lines[1] != records[2].Raw {
		t.Errorf("Line 1 = %q, want %q", lines[1], records[2].Raw)
	}
}

func TestWriteJSON(t *testing.T) {
	records := testRecords()
	var buf bytes.Buffer

	err := Write(&buf, records, []int{1}, FormatJSON)
	if err != nil {
		t.Fatalf("Write JSON: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	if !strings.Contains(out, `"service":"api"`) {
		t.Errorf("JSON output missing expected field: %s", out)
	}
	if !strings.Contains(out, `"latency":"45"`) {
		t.Errorf("JSON output missing latency: %s", out)
	}
}

func TestWriteCSV(t *testing.T) {
	records := testRecords()
	var buf bytes.Buffer

	err := Write(&buf, records, []int{0, 1}, FormatCSV)
	if err != nil {
		t.Fatalf("Write CSV: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// Header + 2 data rows
	if len(lines) != 3 {
		t.Fatalf("Expected 3 CSV lines (header + 2 rows), got %d: %v", len(lines), lines)
	}
	// Header should contain field names
	header := lines[0]
	if !strings.Contains(header, "level") || !strings.Contains(header, "message") {
		t.Errorf("CSV header missing expected fields: %s", header)
	}
}

func TestWriteEmptyResults(t *testing.T) {
	records := testRecords()
	var buf bytes.Buffer

	// Raw with empty IDs
	err := Write(&buf, records, []int{}, FormatRaw)
	if err != nil {
		t.Fatalf("Write raw empty: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("Expected empty output, got %q", buf.String())
	}

	// CSV with empty IDs
	buf.Reset()
	err = Write(&buf, records, []int{}, FormatCSV)
	if err != nil {
		t.Fatalf("Write CSV empty: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("Expected empty CSV output, got %q", buf.String())
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    Format
		wantErr bool
	}{
		{"raw", FormatRaw, false},
		{"json", FormatJSON, false},
		{"csv", FormatCSV, false},
		{"", FormatRaw, false},
		{"xml", "", true},
		{"JSON", "", true},
	}

	for _, tt := range tests {
		f, err := ParseFormat(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseFormat(%q): expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseFormat(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if f != tt.want {
			t.Errorf("ParseFormat(%q) = %q, want %q", tt.input, f, tt.want)
		}
	}
}
