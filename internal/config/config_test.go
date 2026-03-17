package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSimpleAlias(t *testing.T) {
	cfg, err := Parse(`
[aliases]
err = "level:error OR level:fatal"
slow = "latency>1000"
`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Aliases["err"].Query != "level:error OR level:fatal" {
		t.Errorf("err alias = %q", cfg.Aliases["err"].Query)
	}
	if cfg.Aliases["slow"].Query != "latency>1000" {
		t.Errorf("slow alias = %q", cfg.Aliases["slow"].Query)
	}
}

func TestParseRichAlias(t *testing.T) {
	cfg, err := Parse(`
[aliases.oncall]
query = "level:error AND last:15m"
columns = ["timestamp", "service", "message"]
`)
	if err != nil {
		t.Fatal(err)
	}
	entry := cfg.Aliases["oncall"]
	if entry.Query != "level:error AND last:15m" {
		t.Errorf("oncall query = %q", entry.Query)
	}
	if len(entry.Columns) != 3 || entry.Columns[0] != "timestamp" {
		t.Errorf("oncall columns = %v", entry.Columns)
	}
}

func TestParseThemeAndColumns(t *testing.T) {
	cfg, err := Parse(`
theme = "dark"
columns = ["timestamp", "level", "message"]
`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "dark" {
		t.Errorf("theme = %q", cfg.Theme)
	}
	if len(cfg.Columns) != 3 {
		t.Errorf("columns = %v", cfg.Columns)
	}
}

func TestParseEmptyConfig(t *testing.T) {
	cfg, err := Parse("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "" {
		t.Errorf("expected empty theme, got %q", cfg.Theme)
	}
	if len(cfg.Aliases) != 0 {
		t.Errorf("expected no aliases, got %d", len(cfg.Aliases))
	}
}

func TestParseInvalidTOML(t *testing.T) {
	_, err := Parse("[invalid toml ===")
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

func TestLoadEmptyPath(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "" || len(cfg.Aliases) != 0 {
		t.Error("expected zero-value config for empty path")
	}
}

func TestFindConfigWalksUp(t *testing.T) {
	// Create a temp directory structure: /tmp/root/.logq.toml and /tmp/root/sub/
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub", "deep")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, ".logq.toml")
	if err := os.WriteFile(configPath, []byte(`theme = "dark"`), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := findConfigFrom(subdir)
	if err != nil {
		t.Fatal(err)
	}
	if found != configPath {
		t.Errorf("found = %q, want %q", found, configPath)
	}
}

func TestFindConfigNotFound(t *testing.T) {
	dir := t.TempDir()
	found, err := findConfigFrom(dir)
	if err != nil {
		t.Fatal(err)
	}
	if found != "" {
		t.Errorf("expected empty path, got %q", found)
	}
}

func TestParseMixedAliases(t *testing.T) {
	cfg, err := Parse(`
[aliases]
err = "level:error"

[aliases.oncall]
query = "level:error AND last:15m"
columns = ["timestamp", "message"]
`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Aliases["err"].Query != "level:error" {
		t.Errorf("err = %q", cfg.Aliases["err"].Query)
	}
	if cfg.Aliases["oncall"].Query != "level:error AND last:15m" {
		t.Errorf("oncall = %q", cfg.Aliases["oncall"].Query)
	}
	if len(cfg.Aliases["oncall"].Columns) != 2 {
		t.Errorf("oncall columns = %v", cfg.Aliases["oncall"].Columns)
	}
}

func TestParseTraceConfig(t *testing.T) {
	cfg, err := Parse(`
[trace]
id_fields = ["trace_id", "request_id", "my_custom_id"]
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Trace.IDFields) != 3 {
		t.Fatalf("expected 3 trace ID fields, got %d", len(cfg.Trace.IDFields))
	}
	if cfg.Trace.IDFields[0] != "trace_id" {
		t.Errorf("first field = %q", cfg.Trace.IDFields[0])
	}
}

func TestParseEmptyTraceConfig(t *testing.T) {
	cfg, err := Parse(`theme = "dark"`)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Trace.IDFields) != 0 {
		t.Errorf("expected nil/empty trace fields, got %v", cfg.Trace.IDFields)
	}
}

func TestScaffoldTemplateNotEmpty(t *testing.T) {
	tpl := ScaffoldTemplate()
	if len(tpl) < 100 {
		t.Error("scaffold template too short")
	}
}
