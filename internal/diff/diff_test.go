package diff

import (
	"bytes"
	"strings"
	"testing"

	"github.com/riccardomerenda/logq/internal/parser"
)

func makeRecord(lineNum int, level, message string) parser.Record {
	return parser.Record{
		LineNumber: lineNum,
		Level:      level,
		Message:    message,
		Fields:     map[string]string{"level": level, "message": message},
		Raw:        `{"level":"` + level + `","message":"` + message + `"}`,
	}
}

func TestCompare(t *testing.T) {
	left := []parser.Record{
		makeRecord(1, "error", "Connection timeout to 10.0.1.5:8080"),
		makeRecord(2, "error", "Connection timeout to 10.0.2.3:9090"),
		makeRecord(3, "info", "Processed 100 items"),
		makeRecord(4, "info", "Processed 200 items"),
	}
	right := []parser.Record{
		makeRecord(1, "error", "Connection timeout to 10.0.3.1:8080"),
		makeRecord(2, "error", "Connection timeout to 10.0.3.2:9090"),
		makeRecord(3, "error", "Connection timeout to 10.0.3.3:8080"),
		makeRecord(4, "error", "Connection timeout to 10.0.3.4:9090"),
		makeRecord(5, "info", "Processed 300 items"),
		makeRecord(6, "error", "Disk full on /var/log/app"),
	}

	leftIDs := []int{0, 1, 2, 3}
	rightIDs := []int{0, 1, 2, 3, 4, 5}

	result := Compare(left, right, leftIDs, rightIDs)

	if result.LeftCount != 4 {
		t.Errorf("LeftCount = %d, want 4", result.LeftCount)
	}
	if result.RightCount != 6 {
		t.Errorf("RightCount = %d, want 6", result.RightCount)
	}

	// Error level: left=2, right=5
	foundError := false
	for _, l := range result.Levels {
		if l.Level == "error" {
			foundError = true
			if l.LeftCount != 2 {
				t.Errorf("error left = %d, want 2", l.LeftCount)
			}
			if l.RightCount != 5 {
				t.Errorf("error right = %d, want 5", l.RightCount)
			}
		}
	}
	if !foundError {
		t.Error("error level not found in diff")
	}

	// Info level: left=2, right=1
	for _, l := range result.Levels {
		if l.Level == "info" {
			if l.LeftCount != 2 {
				t.Errorf("info left = %d, want 2", l.LeftCount)
			}
			if l.RightCount != 1 {
				t.Errorf("info right = %d, want 1", l.RightCount)
			}
		}
	}

	// Should have at least 1 new pattern ("Disk full on <path>")
	if len(result.NewPatterns) == 0 {
		t.Error("expected at least 1 new pattern")
	}

	// Connection timeout and Processed exist in both — should be in Changed
	if len(result.Changed) == 0 {
		t.Error("expected changed patterns")
	}
}

func TestCompareIdentical(t *testing.T) {
	records := []parser.Record{
		makeRecord(1, "info", "Request completed in 100ms"),
		makeRecord(2, "info", "Request completed in 200ms"),
	}
	ids := []int{0, 1}

	result := Compare(records, records, ids, ids)

	if result.LeftCount != 2 || result.RightCount != 2 {
		t.Errorf("counts: left=%d right=%d", result.LeftCount, result.RightCount)
	}
	if len(result.NewPatterns) != 0 {
		t.Errorf("expected no new patterns, got %d", len(result.NewPatterns))
	}
	if len(result.GonePatterns) != 0 {
		t.Errorf("expected no gone patterns, got %d", len(result.GonePatterns))
	}
}

func TestCompareEmpty(t *testing.T) {
	left := []parser.Record{
		makeRecord(1, "error", "Something failed"),
	}
	right := []parser.Record{}

	result := Compare(left, right, []int{0}, []int{})

	if result.LeftCount != 1 || result.RightCount != 0 {
		t.Errorf("counts: left=%d right=%d", result.LeftCount, result.RightCount)
	}
	if len(result.GonePatterns) != 1 {
		t.Errorf("expected 1 gone pattern, got %d", len(result.GonePatterns))
	}
}

func TestChangePercent(t *testing.T) {
	tests := []struct {
		before, after int
		want          float64
	}{
		{10, 20, 100},
		{20, 10, -50},
		{10, 10, 0},
		{0, 10, 100},
		{0, 0, 0},
	}
	for _, tt := range tests {
		got := ChangePercent(tt.before, tt.after)
		if got != tt.want {
			t.Errorf("ChangePercent(%d, %d) = %f, want %f", tt.before, tt.after, got, tt.want)
		}
	}
}

func TestFormatChange(t *testing.T) {
	tests := []struct {
		before, after int
		want          string
	}{
		{10, 20, "+100%"},
		{20, 10, "-50%"},
		{10, 10, "0%"},
		{0, 10, "new"},
		{0, 0, "-"},
	}
	for _, tt := range tests {
		got := FormatChange(tt.before, tt.after)
		if got != tt.want {
			t.Errorf("FormatChange(%d, %d) = %q, want %q", tt.before, tt.after, got, tt.want)
		}
	}
}

func TestWriteDiffTable(t *testing.T) {
	result := Result{
		LeftName:      "before.log",
		RightName:     "after.log",
		LeftCount:     100,
		RightCount:    200,
		LeftPatterns:  5,
		RightPatterns: 8,
		Levels: []LevelDiff{
			{Level: "error", LeftCount: 10, RightCount: 50},
		},
		NewPatterns: []PatternDiff{
			{Template: "New error <num>", RightCount: 20},
		},
		GonePatterns: []PatternDiff{},
		Changed: []PatternDiff{
			{Template: "Timeout after <duration>", LeftCount: 5, RightCount: 30},
		},
	}

	var buf bytes.Buffer
	if err := WriteDiff(&buf, result, "", 0, 50); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "before.log") {
		t.Error("output should contain left filename")
	}
	if !strings.Contains(out, "after.log") {
		t.Error("output should contain right filename")
	}
	if !strings.Contains(out, "New error") {
		t.Error("output should contain new pattern")
	}
	if !strings.Contains(out, "Timeout after") {
		t.Error("output should contain changed pattern")
	}
}

func TestWriteDiffJSON(t *testing.T) {
	result := Result{
		LeftName:      "a.log",
		RightName:     "b.log",
		LeftCount:     10,
		RightCount:    20,
		LeftPatterns:  2,
		RightPatterns: 3,
		Levels:        []LevelDiff{{Level: "error", LeftCount: 5, RightCount: 15}},
		NewPatterns:   []PatternDiff{{Template: "new thing", RightCount: 5}},
	}

	var buf bytes.Buffer
	if err := WriteDiff(&buf, result, "json", 0, 50); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, `"a.log"`) {
		t.Error("JSON should contain left name")
	}
	if !strings.Contains(out, `"new thing"`) {
		t.Error("JSON should contain new pattern template")
	}
}
