package pattern

import (
	"regexp"
	"sort"

	"github.com/riccardomerenda/logq/internal/parser"
)

// Cluster represents a group of log records sharing the same message template.
type Cluster struct {
	Template  string
	Count     int
	RecordIDs []int
}

// replacement pairs applied in order (most specific first).
var replacements = []struct {
	re   *regexp.Regexp
	repl string
}{
	// UUIDs: 8-4-4-4-12 hex
	{regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`), "<uuid>"},
	// IPv4 addresses
	{regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), "<ip>"},
	// Hex IDs (16+ hex chars, standalone)
	{regexp.MustCompile(`\b[0-9a-fA-F]{16,}\b`), "<hex>"},
	// ISO/RFC3339 timestamps
	{regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?`), "<timestamp>"},
	// File paths
	{regexp.MustCompile(`(/[a-zA-Z0-9._-]+){2,}`), "<path>"},
	// Durations with units (e.g. 3000ms, 4.5s, 2h)
	{regexp.MustCompile(`\b\d+(\.\d+)?(ms|ns|us|µs|s|m|h)\b`), "<duration>"},
	// Numbers (integers and decimals)
	{regexp.MustCompile(`\b\d+(\.\d+)?\b`), "<num>"},
}

// Templatize replaces variable parts of a message with typed placeholders.
func Templatize(message string) string {
	result := message
	for _, r := range replacements {
		result = r.re.ReplaceAllString(result, r.repl)
	}
	return result
}

// Clusterize groups records by their templatized message and returns
// clusters sorted by count descending.
func Clusterize(records []parser.Record, ids []int) []Cluster {
	type entry struct {
		template  string
		recordIDs []int
	}
	groups := make(map[string]*entry)

	for _, id := range ids {
		msg := records[id].Message
		if msg == "" {
			msg = records[id].Raw
		}
		tmpl := Templatize(msg)
		if e, ok := groups[tmpl]; ok {
			e.recordIDs = append(e.recordIDs, id)
		} else {
			groups[tmpl] = &entry{template: tmpl, recordIDs: []int{id}}
		}
	}

	clusters := make([]Cluster, 0, len(groups))
	for _, e := range groups {
		clusters = append(clusters, Cluster{
			Template:  e.template,
			Count:     len(e.recordIDs),
			RecordIDs: e.recordIDs,
		})
	}

	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].Count != clusters[j].Count {
			return clusters[i].Count > clusters[j].Count
		}
		return clusters[i].Template < clusters[j].Template
	})

	return clusters
}
