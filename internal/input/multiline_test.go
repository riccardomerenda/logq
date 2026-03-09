package input

import (
	"testing"
)

func TestGroupLines_JSONSingleLine(t *testing.T) {
	lines := []string{
		`{"level":"info","message":"hello"}`,
		`{"level":"error","message":"oops"}`,
		`{"level":"warn","message":"hmm"}`,
	}
	entries := GroupLines(lines)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	for i, e := range entries {
		if e.LineNumber != i+1 {
			t.Errorf("entry %d: expected line %d, got %d", i, i+1, e.LineNumber)
		}
	}
}

func TestGroupLines_TimestampAnchored(t *testing.T) {
	lines := []string{
		"12:43:10 Some timestamp line",
		"System.Exception: something broke",
		"   at Foo.Bar() in Foo.cs:line 42",
		"   at Baz.Qux() in Baz.cs:line 10",
		"12:44:00 Another entry",
		"continuation line",
		"   more stack trace",
	}
	entries := GroupLines(lines)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].LineNumber != 1 {
		t.Errorf("entry 0: expected line 1, got %d", entries[0].LineNumber)
	}
	if entries[1].LineNumber != 5 {
		t.Errorf("entry 1: expected line 5, got %d", entries[1].LineNumber)
	}
}

func TestGroupLines_ISOTimestampAnchored(t *testing.T) {
	lines := []string{
		"2026-03-08 10:00:01 INFO Starting server",
		"  listening on port 8080",
		"  ready",
		"2026-03-08 10:00:02 ERROR Failed to connect",
		"  connection refused",
		"  retrying in 5s",
	}
	entries := GroupLines(lines)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestGroupLines_EmptyInput(t *testing.T) {
	entries := GroupLines(nil)
	if entries != nil {
		t.Fatalf("expected nil, got %v", entries)
	}
	entries = GroupLines([]string{})
	if entries != nil {
		t.Fatalf("expected nil, got %v", entries)
	}
}

func TestGroupLines_AllEmpty(t *testing.T) {
	entries := GroupLines([]string{"", "", ""})
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestGroupLines_MixedFormats(t *testing.T) {
	// File where most lines are JSON → single-line mode
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = `{"msg":"line"}`
	}
	entries := GroupLines(lines)
	if len(entries) != 10 {
		t.Fatalf("expected 10 entries, got %d", len(entries))
	}
}

func TestGroupLines_JavaStackTrace(t *testing.T) {
	lines := []string{
		"2026-03-08 10:00:01 ERROR NullPointerException",
		"	at com.example.Foo.bar(Foo.java:42)",
		"	at com.example.Main.run(Main.java:10)",
		"Caused by: java.io.IOException: broken pipe",
		"	at com.example.IO.read(IO.java:55)",
		"	... 3 more",
		"2026-03-08 10:00:02 INFO Recovery successful",
	}
	entries := GroupLines(lines)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestGroupLines_PreservesLineNumbers(t *testing.T) {
	lines := []string{
		"",
		"12:00:00 First entry",
		"  continuation",
		"",
		"12:01:00 Second entry",
	}
	entries := GroupLines(lines)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].LineNumber != 2 {
		t.Errorf("first entry: expected line 2, got %d", entries[0].LineNumber)
	}
	if entries[1].LineNumber != 5 {
		t.Errorf("second entry: expected line 5, got %d", entries[1].LineNumber)
	}
}

func TestDetectMode_JSON(t *testing.T) {
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = `{"key":"value"}`
	}
	mode := detectMode(lines)
	if mode != modeSingleLine {
		t.Errorf("expected modeSingleLine, got %d", mode)
	}
}

func TestDetectMode_Timestamps(t *testing.T) {
	lines := []string{
		"12:00:00 Entry 1",
		"  detail A",
		"  detail B",
		"12:01:00 Entry 2",
		"  detail C",
	}
	mode := detectMode(lines)
	if mode != modeTimestampAnchored {
		t.Errorf("expected modeTimestampAnchored, got %d", mode)
	}
}
