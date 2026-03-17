package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// AliasEntry represents a query alias, optionally with column overrides.
type AliasEntry struct {
	Query   string
	Columns []string
}

// Config holds settings loaded from a .logq.toml file.
type Config struct {
	Theme   string
	Columns []string
	Aliases map[string]AliasEntry
}

// rawConfig mirrors the TOML structure for decoding.
type rawConfig struct {
	Theme   string                 `toml:"theme"`
	Columns []string               `toml:"columns"`
	Aliases map[string]interface{} `toml:"aliases"`
}

// FindConfig searches for .logq.toml starting from the current directory
// and walking up to the filesystem root. Returns the path if found, or "".
func FindConfig() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return findConfigFrom(dir)
}

// findConfigFrom searches from a specific directory (testable).
func findConfigFrom(dir string) (string, error) {
	for {
		path := filepath.Join(dir, ".logq.toml")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", nil
}

// Load reads and parses a .logq.toml file. If path is empty, returns a zero-value Config.
func Load(path string) (*Config, error) {
	if path == "" {
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	return Parse(string(data))
}

// Parse decodes TOML content into a Config.
func Parse(content string) (*Config, error) {
	var raw rawConfig
	if _, err := toml.Decode(content, &raw); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg := &Config{
		Theme:   raw.Theme,
		Columns: raw.Columns,
		Aliases: make(map[string]AliasEntry),
	}

	for name, val := range raw.Aliases {
		switch v := val.(type) {
		case string:
			cfg.Aliases[name] = AliasEntry{Query: v}
		case map[string]interface{}:
			entry := AliasEntry{}
			if q, ok := v["query"].(string); ok {
				entry.Query = q
			}
			if cols, ok := v["columns"].([]interface{}); ok {
				for _, c := range cols {
					if s, ok := c.(string); ok {
						entry.Columns = append(entry.Columns, s)
					}
				}
			}
			cfg.Aliases[name] = entry
		default:
			return nil, fmt.Errorf("invalid alias %q: expected string or table", name)
		}
	}

	return cfg, nil
}

// ScaffoldTemplate returns a starter .logq.toml file with comments.
func ScaffoldTemplate() string {
	return `# logq configuration — https://github.com/riccardomerenda/logq
# Place this file in your project root. logq auto-discovers it.

# Color theme: "auto", "dark", or "light"
# theme = "auto"

# Default columns for TUI and batch mode
# columns = ["timestamp", "level", "service", "message"]

# Query aliases — use as @name in queries
# Built-in aliases (@err, @warn, @slow) are always available.
[aliases]
# err = "level:error OR level:fatal"
# slow = "latency>1000"
# noisy = "NOT service:healthcheck AND NOT service:ping"

# Rich alias with column override
# [aliases.oncall]
# query = "level:error AND last:15m"
# columns = ["timestamp", "service", "message"]
`
}
