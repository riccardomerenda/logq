package benchmarks

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/riccardomerenda/logq/internal/index"
	"github.com/riccardomerenda/logq/internal/parser"
	"github.com/riccardomerenda/logq/internal/query"
)

// generateJSONLines creates n realistic JSON log lines.
func generateJSONLines(n int) []string {
	levels := []string{"debug", "info", "info", "info", "warn", "error", "fatal"}
	services := []string{"api", "auth", "db", "cache", "gateway"}
	messages := []string{
		"request started",
		"request completed",
		"token expired",
		"slow query",
		"cache hit",
		"connection refused",
		"health check ok",
		"rate limit approaching",
		"deadlock detected",
		"upstream timeout",
	}

	rng := rand.New(rand.NewSource(42))
	base := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	lines := make([]string, n)

	for i := 0; i < n; i++ {
		ts := base.Add(time.Duration(i) * time.Millisecond * 100)
		level := levels[rng.Intn(len(levels))]
		svc := services[rng.Intn(len(services))]
		msg := messages[rng.Intn(len(messages))]
		latency := rng.Intn(5000)
		reqID := fmt.Sprintf("req_%06d", i)

		lines[i] = fmt.Sprintf(
			`{"timestamp":"%s","level":"%s","service":"%s","message":"%s","latency":%d,"request_id":"%s"}`,
			ts.Format(time.RFC3339Nano), level, svc, msg, latency, reqID,
		)
	}
	return lines
}

// parseLines parses raw lines into records.
func parseLines(lines []string) []parser.Record {
	records := make([]parser.Record, len(lines))
	for i, line := range lines {
		records[i] = parser.Parse(line, i+1)
	}
	return records
}

// --- Parse benchmarks ---

func BenchmarkParse_10k(b *testing.B) {
	lines := generateJSONLines(10_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j, line := range lines {
			parser.Parse(line, j+1)
		}
	}
}

func BenchmarkParse_100k(b *testing.B) {
	lines := generateJSONLines(100_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j, line := range lines {
			parser.Parse(line, j+1)
		}
	}
}

// --- Index build benchmarks ---

func BenchmarkIndexBuild_10k(b *testing.B) {
	records := parseLines(generateJSONLines(10_000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index.Build(records)
	}
}

func BenchmarkIndexBuild_100k(b *testing.B) {
	records := parseLines(generateJSONLines(100_000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index.Build(records)
	}
}

// --- Query benchmarks (on pre-built 100k index) ---

func buildIndex100k() *index.Index {
	return index.Build(parseLines(generateJSONLines(100_000)))
}

func BenchmarkQuery_ExactMatch(b *testing.B) {
	idx := buildIndex100k()
	node, _ := query.ParseQuery("level:error")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.Evaluate(node, idx)
	}
}

func BenchmarkQuery_NumericCompare(b *testing.B) {
	idx := buildIndex100k()
	node, _ := query.ParseQuery("latency>2500")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.Evaluate(node, idx)
	}
}

func BenchmarkQuery_Compound(b *testing.B) {
	idx := buildIndex100k()
	node, _ := query.ParseQuery("level:error AND latency>1000")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.Evaluate(node, idx)
	}
}

func BenchmarkQuery_Complex(b *testing.B) {
	idx := buildIndex100k()
	node, _ := query.ParseQuery("level:error AND latency>1000 AND NOT service:cache")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.Evaluate(node, idx)
	}
}

func BenchmarkQuery_FullText(b *testing.B) {
	idx := buildIndex100k()
	node, _ := query.ParseQuery("timeout")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.Evaluate(node, idx)
	}
}

func BenchmarkQuery_Regex(b *testing.B) {
	idx := buildIndex100k()
	node, _ := query.ParseQuery(`message~"timeout.*"`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.Evaluate(node, idx)
	}
}

// --- Histogram benchmark ---

func BenchmarkHistogram_100k(b *testing.B) {
	idx := buildIndex100k()
	allIDs := idx.AllIDs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Histogram(50, allIDs)
	}
}

// --- Memory benchmark ---

func BenchmarkMemory_100k(b *testing.B) {
	lines := generateJSONLines(100_000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		records := parseLines(lines)
		index.Build(records)
	}
}
