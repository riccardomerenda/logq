package pattern

import (
	"testing"

	"github.com/riccardomerenda/logq/internal/parser"
)

func TestTemplatize_IP(t *testing.T) {
	got := Templatize("Connection to 10.0.1.5 failed")
	want := "Connection to <ip> failed"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplatize_UUID(t *testing.T) {
	got := Templatize("Processing request 550e8400-e29b-41d4-a716-446655440000")
	want := "Processing request <uuid>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplatize_Numbers(t *testing.T) {
	got := Templatize("Processed 42 records in batch 7")
	want := "Processed <num> records in batch <num>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplatize_Duration(t *testing.T) {
	got := Templatize("Request took 3000ms")
	want := "Request took <duration>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplatize_Path(t *testing.T) {
	got := Templatize("Reading /var/log/app.log")
	want := "Reading <path>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplatize_HexID(t *testing.T) {
	got := Templatize("Span 463ac35c9f6413ad started")
	want := "Span <hex> started"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplatize_Timestamp(t *testing.T) {
	got := Templatize("Event at 2026-03-17T10:00:01Z")
	want := "Event at <timestamp>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplatize_Mixed(t *testing.T) {
	got := Templatize("Connection timeout to 10.0.1.5:5432 after 3000ms")
	want := "Connection timeout to <ip>:<num> after <duration>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplatize_NoVariables(t *testing.T) {
	msg := "Health check ok"
	got := Templatize(msg)
	if got != msg {
		t.Errorf("got %q, want %q", got, msg)
	}
}

func TestTemplatize_Empty(t *testing.T) {
	got := Templatize("")
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestClusterize_Grouping(t *testing.T) {
	records := []parser.Record{
		{Message: "Connection timeout to 10.0.1.5:5432 after 3000ms"},
		{Message: "Connection timeout to 10.0.2.8:5432 after 5200ms"},
		{Message: "Connection timeout to 10.0.1.12:5432 after 4100ms"},
		{Message: "Health check ok"},
		{Message: "Health check ok"},
	}
	ids := []int{0, 1, 2, 3, 4}
	clusters := Clusterize(records, ids)

	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}
	// First cluster should be the timeout (3 occurrences)
	if clusters[0].Count != 3 {
		t.Errorf("first cluster count = %d, want 3", clusters[0].Count)
	}
	if clusters[1].Count != 2 {
		t.Errorf("second cluster count = %d, want 2", clusters[1].Count)
	}
}

func TestClusterize_SortOrder(t *testing.T) {
	records := []parser.Record{
		{Message: "A"},
		{Message: "B"},
		{Message: "B"},
		{Message: "C"},
		{Message: "C"},
		{Message: "C"},
	}
	ids := []int{0, 1, 2, 3, 4, 5}
	clusters := Clusterize(records, ids)

	if clusters[0].Template != "C" || clusters[0].Count != 3 {
		t.Errorf("first = %q (%d), want C (3)", clusters[0].Template, clusters[0].Count)
	}
	if clusters[1].Template != "B" || clusters[1].Count != 2 {
		t.Errorf("second = %q (%d), want B (2)", clusters[1].Template, clusters[1].Count)
	}
}

func TestClusterize_RecordIDs(t *testing.T) {
	records := []parser.Record{
		{Message: "Error on host 10.0.0.1"},
		{Message: "Error on host 10.0.0.2"},
		{Message: "Ok"},
	}
	ids := []int{0, 1, 2}
	clusters := Clusterize(records, ids)

	// Find the error cluster
	for _, c := range clusters {
		if c.Template == "Error on host <ip>" {
			if len(c.RecordIDs) != 2 {
				t.Errorf("expected 2 record IDs, got %d", len(c.RecordIDs))
			}
			return
		}
	}
	t.Error("did not find expected cluster")
}

func TestClusterize_FallbackToRaw(t *testing.T) {
	records := []parser.Record{
		{Message: "", Raw: "raw line with 42 items"},
		{Message: "", Raw: "raw line with 99 items"},
	}
	ids := []int{0, 1}
	clusters := Clusterize(records, ids)

	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].Template != "raw line with <num> items" {
		t.Errorf("template = %q", clusters[0].Template)
	}
}

func TestClusterize_FilteredIDs(t *testing.T) {
	records := []parser.Record{
		{Message: "A"},
		{Message: "B"},
		{Message: "C"},
	}
	ids := []int{0, 2} // skip record 1
	clusters := Clusterize(records, ids)

	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}
}
