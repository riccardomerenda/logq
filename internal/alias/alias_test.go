package alias

import (
	"strings"
	"testing"

	"github.com/riccardomerenda/logq/internal/config"
)

func TestExpandBuiltinErr(t *testing.T) {
	r := NewRegistry(nil)
	got, err := r.Expand("@err")
	if err != nil {
		t.Fatal(err)
	}
	if got != "(level:error OR level:fatal)" {
		t.Errorf("got %q", got)
	}
}

func TestExpandBuiltinWarn(t *testing.T) {
	r := NewRegistry(nil)
	got, err := r.Expand("@warn")
	if err != nil {
		t.Fatal(err)
	}
	if got != "(level:warn OR level:warning)" {
		t.Errorf("got %q", got)
	}
}

func TestExpandCompoundQuery(t *testing.T) {
	r := NewRegistry(nil)
	got, err := r.Expand("@err AND service:auth")
	if err != nil {
		t.Fatal(err)
	}
	if got != "(level:error OR level:fatal) AND service:auth" {
		t.Errorf("got %q", got)
	}
}

func TestExpandMultipleAliases(t *testing.T) {
	r := NewRegistry(nil)
	got, err := r.Expand("@err OR @slow")
	if err != nil {
		t.Fatal(err)
	}
	if got != "(level:error OR level:fatal) OR (latency>1000)" {
		t.Errorf("got %q", got)
	}
}

func TestExpandUserAlias(t *testing.T) {
	r := NewRegistry(map[string]config.AliasEntry{
		"noisy": {Query: "NOT service:healthcheck"},
	})
	got, err := r.Expand("@noisy AND @err")
	if err != nil {
		t.Fatal(err)
	}
	if got != "(NOT service:healthcheck) AND (level:error OR level:fatal)" {
		t.Errorf("got %q", got)
	}
}

func TestExpandUserOverridesBuiltin(t *testing.T) {
	r := NewRegistry(map[string]config.AliasEntry{
		"err": {Query: "level:error"},
	})
	got, err := r.Expand("@err")
	if err != nil {
		t.Fatal(err)
	}
	if got != "(level:error)" {
		t.Errorf("got %q", got)
	}
}

func TestExpandNestedAlias(t *testing.T) {
	r := NewRegistry(map[string]config.AliasEntry{
		"oncall": {Query: "@err AND last:15m"},
	})
	got, err := r.Expand("@oncall")
	if err != nil {
		t.Fatal(err)
	}
	if got != "((level:error OR level:fatal) AND last:15m)" {
		t.Errorf("got %q", got)
	}
}

func TestExpandCircularAlias(t *testing.T) {
	r := NewRegistry(map[string]config.AliasEntry{
		"a": {Query: "@b"},
		"b": {Query: "@a"},
	})
	_, err := r.Expand("@a")
	if err == nil {
		t.Error("expected circular alias error")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected circular error, got: %v", err)
	}
}

func TestExpandUnknownAlias(t *testing.T) {
	r := NewRegistry(nil)
	_, err := r.Expand("@unknown")
	if err == nil {
		t.Error("expected unknown alias error")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected unknown error, got: %v", err)
	}
}

func TestExpandInsideQuotesNotExpanded(t *testing.T) {
	r := NewRegistry(nil)
	got, err := r.Expand(`message:"@err in email"`)
	if err != nil {
		t.Fatal(err)
	}
	if got != `message:"@err in email"` {
		t.Errorf("got %q, want no expansion inside quotes", got)
	}
}

func TestExpandNoAliases(t *testing.T) {
	r := NewRegistry(nil)
	got, err := r.Expand("level:error AND service:auth")
	if err != nil {
		t.Fatal(err)
	}
	if got != "level:error AND service:auth" {
		t.Errorf("got %q", got)
	}
}

func TestExpandEmptyQuery(t *testing.T) {
	r := NewRegistry(nil)
	got, err := r.Expand("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("got %q", got)
	}
}

func TestExpandNilRegistry(t *testing.T) {
	var r *Registry
	got, err := r.Expand("@err")
	if err != nil {
		t.Fatal(err)
	}
	if got != "@err" {
		t.Errorf("got %q", got)
	}
}

func TestExpandWithParentheses(t *testing.T) {
	r := NewRegistry(nil)
	got, err := r.Expand("(@err) AND service:api")
	if err != nil {
		t.Fatal(err)
	}
	if got != "((level:error OR level:fatal)) AND service:api" {
		t.Errorf("got %q", got)
	}
}

func TestNames(t *testing.T) {
	r := NewRegistry(map[string]config.AliasEntry{
		"noisy": {Query: "NOT service:healthcheck"},
	})
	names := r.Names()
	// Should include builtins + user-defined, sorted
	if len(names) != 4 {
		t.Errorf("expected 4 names, got %v", names)
	}
	// Check sorted
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("not sorted: %v", names)
			break
		}
	}
}

func TestLookup(t *testing.T) {
	r := NewRegistry(nil)
	entry, ok := r.Lookup("err")
	if !ok {
		t.Fatal("expected to find err")
	}
	if entry.Query != "level:error OR level:fatal" {
		t.Errorf("got %q", entry.Query)
	}

	_, ok = r.Lookup("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestNilRegistryNames(t *testing.T) {
	var r *Registry
	if names := r.Names(); names != nil {
		t.Errorf("expected nil, got %v", names)
	}
}
