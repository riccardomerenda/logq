package history

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const maxEntries = 500

// DataDir returns the directory for logq persistent data.
// Uses $XDG_DATA_HOME/logq on Unix, %APPDATA%/logq on Windows,
// falling back to ~/.local/share/logq.
func DataDir() string {
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "logq")
		}
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "logq")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".logq")
	}
	return filepath.Join(home, ".local", "share", "logq")
}

// HistoryPath returns the default path for the history file.
func HistoryPath() string {
	return filepath.Join(DataDir(), "history")
}

// Load reads history entries from the given file, one per line.
// Returns an empty slice if the file does not exist.
func Load(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			entries = append(entries, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return entries, err
	}
	// Cap
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}
	return entries, nil
}

// Save writes all entries to the file, capped at maxEntries.
// Creates parent directories as needed.
func Save(path string, entries []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// Deduplicate consecutive entries and cap
	var deduped []string
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if len(deduped) > 0 && deduped[len(deduped)-1] == e {
			continue
		}
		deduped = append(deduped, e)
	}
	if len(deduped) > maxEntries {
		deduped = deduped[len(deduped)-maxEntries:]
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, e := range deduped {
		if _, err := w.WriteString(e + "\n"); err != nil {
			return err
		}
	}
	return w.Flush()
}

// Append adds a single entry to the history file.
// Creates parent directories as needed.
func Append(path string, entry string) error {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(entry + "\n")
	return err
}
