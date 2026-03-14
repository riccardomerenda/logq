package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	entries, err := Load(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty, got %d entries", len(entries))
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	entries := []string{"level:error", "service:auth", "latency>500"}

	if err := Save(path, entries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != len(entries) {
		t.Fatalf("expected %d entries, got %d", len(entries), len(loaded))
	}
	for i, e := range entries {
		if loaded[i] != e {
			t.Errorf("entry %d: expected %q, got %q", i, e, loaded[i])
		}
	}
}

func TestSaveDeduplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	entries := []string{"a", "a", "b", "b", "b", "c"}

	if err := Save(path, entries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	expected := []string{"a", "b", "c"}
	if len(loaded) != len(expected) {
		t.Fatalf("expected %d entries, got %d", len(expected), len(loaded))
	}
}

func TestSaveCaps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	entries := make([]string, 600)
	for i := range entries {
		entries[i] = "query" + string(rune('A'+i%26))
	}

	if err := Save(path, entries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) > maxEntries {
		t.Fatalf("expected at most %d entries, got %d", maxEntries, len(loaded))
	}
}

func TestAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")

	if err := Append(path, "first"); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := Append(path, "second"); err != nil {
		t.Fatalf("Append: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded))
	}
	if loaded[0] != "first" || loaded[1] != "second" {
		t.Errorf("unexpected entries: %v", loaded)
	}
}

func TestAppendEmptyIsNoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")

	if err := Append(path, ""); err != nil {
		t.Fatalf("Append: %v", err)
	}

	_, err := os.Stat(path)
	if err == nil {
		t.Fatal("expected file not to be created for empty append")
	}
}

func TestDataDir(t *testing.T) {
	dir := DataDir()
	if dir == "" {
		t.Fatal("DataDir returned empty string")
	}
}
