package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/riccardomerenda/logq/internal/parser"
)

func TestGroupBy(t *testing.T) {
	records := []parser.Record{
		{Fields: map[string]string{"service": "auth"}},
		{Fields: map[string]string{"service": "api"}},
		{Fields: map[string]string{"service": "auth"}},
		{Fields: map[string]string{"service": "api"}},
		{Fields: map[string]string{"service": "api"}},
		{Fields: map[string]string{}}, // missing field
	}
	ids := []int{0, 1, 2, 3, 4, 5}

	groups := GroupBy(records, ids, "service")
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	// Sorted by count desc
	if groups[0].Value != "api" || groups[0].Count != 3 {
		t.Errorf("expected api:3, got %s:%d", groups[0].Value, groups[0].Count)
	}
	if groups[1].Value != "auth" || groups[1].Count != 2 {
		t.Errorf("expected auth:2, got %s:%d", groups[1].Value, groups[1].Count)
	}
	if groups[2].Value != "(empty)" || groups[2].Count != 1 {
		t.Errorf("expected (empty):1, got %s:%d", groups[2].Value, groups[2].Count)
	}
}

func TestTopN(t *testing.T) {
	groups := []GroupResult{
		{Value: "a", Count: 10},
		{Value: "b", Count: 5},
		{Value: "c", Count: 3},
		{Value: "d", Count: 1},
	}

	top2 := TopN(groups, 2)
	if len(top2) != 2 {
		t.Fatalf("expected 2, got %d", len(top2))
	}
	if top2[0].Value != "a" || top2[1].Value != "b" {
		t.Errorf("unexpected top2: %v", top2)
	}

	// n=0 returns all
	all := TopN(groups, 0)
	if len(all) != 4 {
		t.Fatalf("expected 4, got %d", len(all))
	}

	// n > len returns all
	big := TopN(groups, 100)
	if len(big) != 4 {
		t.Fatalf("expected 4, got %d", len(big))
	}
}

func TestWriteGroupsTable(t *testing.T) {
	groups := []GroupResult{
		{Value: "api", Count: 10},
		{Value: "auth", Count: 5},
	}
	var buf bytes.Buffer
	if err := WriteGroups(&buf, groups, FormatRaw); err != nil {
		t.Fatalf("WriteGroups: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "VALUE") || !strings.Contains(out, "COUNT") {
		t.Error("expected header")
	}
	if !strings.Contains(out, "api") || !strings.Contains(out, "10") {
		t.Error("expected api:10")
	}
}

func TestWriteGroupsJSON(t *testing.T) {
	groups := []GroupResult{
		{Value: "api", Count: 10},
	}
	var buf bytes.Buffer
	if err := WriteGroups(&buf, groups, FormatJSON); err != nil {
		t.Fatalf("WriteGroups: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"value"`) || !strings.Contains(out, `"api"`) {
		t.Errorf("unexpected JSON: %s", out)
	}
}

func TestWriteGroupsCSV(t *testing.T) {
	groups := []GroupResult{
		{Value: "api", Count: 10},
		{Value: "auth", Count: 5},
	}
	var buf bytes.Buffer
	if err := WriteGroups(&buf, groups, FormatCSV); err != nil {
		t.Fatalf("WriteGroups: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "value,count") {
		t.Error("expected CSV header")
	}
	if !strings.Contains(out, "api,10") {
		t.Error("expected api,10")
	}
}
