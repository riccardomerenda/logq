package input

import (
	"os"
	"testing"
)

func TestFollowReader_ReadNew(t *testing.T) {
	// Create a temp file
	f, err := os.CreateTemp("", "logq-follow-test-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	// Write initial content
	f.WriteString("{\"msg\":\"line1\"}\n{\"msg\":\"line2\"}\n")
	initialSize, _ := f.Seek(0, 2)
	f.Close()

	// Create follow reader starting after initial content
	fr := NewFollowReader(f.Name(), initialSize)

	// No new content yet
	lines, err := fr.ReadNew()
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(lines))
	}

	// Append new content
	f2, _ := os.OpenFile(f.Name(), os.O_APPEND|os.O_WRONLY, 0644)
	f2.WriteString("{\"msg\":\"line3\"}\n{\"msg\":\"line4\"}\n")
	f2.Close()

	// Read new lines
	lines, err = fr.ReadNew()
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "{\"msg\":\"line3\"}" {
		t.Errorf("expected line3, got %q", lines[0])
	}

	// No more new content
	lines, err = fr.ReadNew()
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(lines))
	}
}

func TestFollowReader_PartialLine(t *testing.T) {
	f, err := os.CreateTemp("", "logq-follow-partial-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()

	fr := NewFollowReader(f.Name(), 0)

	// Write a partial line (no newline)
	f2, _ := os.OpenFile(f.Name(), os.O_APPEND|os.O_WRONLY, 0644)
	f2.WriteString("{\"msg\":\"incomplete")
	f2.Close()

	lines, _ := fr.ReadNew()
	if len(lines) != 0 {
		t.Fatalf("expected 0 lines for partial line, got %d", len(lines))
	}

	// Complete the line
	f3, _ := os.OpenFile(f.Name(), os.O_APPEND|os.O_WRONLY, 0644)
	f3.WriteString("\"}\n")
	f3.Close()

	lines, _ = fr.ReadNew()
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
}
