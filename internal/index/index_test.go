package index

import (
	"fmt"
	"testing"
	"time"

	"github.com/riccardomerenda/logq/internal/parser"
)

func makeRecords(n int) []parser.Record {
	records := make([]parser.Record, n)
	base := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)

	levels := []string{"info", "error", "warn", "debug"}
	services := []string{"api", "auth", "db"}

	for i := 0; i < n; i++ {
		level := levels[i%len(levels)]
		service := services[i%len(services)]
		latency := (i + 1) * 100

		records[i] = parser.Record{
			LineNumber: i + 1,
			Timestamp:  base.Add(time.Duration(i) * time.Second),
			Level:      level,
			Message:    fmt.Sprintf("message %d", i),
			Fields: map[string]string{
				"level":   level,
				"service": service,
				"latency": fmt.Sprintf("%d", latency),
				"message": fmt.Sprintf("message %d", i),
			},
			Raw: fmt.Sprintf(`{"level":"%s","service":"%s","latency":%d,"message":"message %d"}`, level, service, latency, i),
		}
	}
	return records
}

func TestFieldLookup(t *testing.T) {
	records := makeRecords(20)
	idx := Build(records)

	// level:error should match records 1, 5, 9, 13, 17
	errors := idx.FieldLookup("level", "error")
	if len(errors) == 0 {
		t.Fatal("Expected some error records")
	}
	for _, id := range errors {
		if idx.Records[id].Level != "error" {
			t.Errorf("Record %d has level %q, want error", id, idx.Records[id].Level)
		}
	}

	// service:api
	apis := idx.FieldLookup("service", "api")
	if len(apis) == 0 {
		t.Fatal("Expected some api records")
	}
	for _, id := range apis {
		if idx.Records[id].Fields["service"] != "api" {
			t.Errorf("Record %d has service %q, want api", id, idx.Records[id].Fields["service"])
		}
	}
}

func TestFieldLookupNoMatch(t *testing.T) {
	records := makeRecords(10)
	idx := Build(records)

	result := idx.FieldLookup("level", "nonexistent")
	if len(result) != 0 {
		t.Errorf("Expected no matches, got %d", len(result))
	}
}

func TestNumericGreater(t *testing.T) {
	records := makeRecords(10)
	idx := Build(records)

	// latency>500 means records with latency 600, 700, 800, 900, 1000
	result := idx.NumericGreater("latency", 500)
	if len(result) != 5 {
		t.Errorf("Expected 5 records with latency>500, got %d", len(result))
	}
	for _, id := range result {
		if idx.Records[id].Fields["latency"] <= "500" {
			// Check numerically
		}
	}
}

func TestNumericGreaterEqual(t *testing.T) {
	records := makeRecords(10)
	idx := Build(records)

	// latency>=500 means records with latency 500, 600, 700, 800, 900, 1000
	result := idx.NumericGreaterEqual("latency", 500)
	if len(result) != 6 {
		t.Errorf("Expected 6 records with latency>=500, got %d", len(result))
	}
}

func TestNumericLess(t *testing.T) {
	records := makeRecords(10)
	idx := Build(records)

	// latency<300 means records with latency 100, 200
	result := idx.NumericLess("latency", 300)
	if len(result) != 2 {
		t.Errorf("Expected 2 records with latency<300, got %d", len(result))
	}
}

func TestNumericLessEqual(t *testing.T) {
	records := makeRecords(10)
	idx := Build(records)

	// latency<=300 means records with latency 100, 200, 300
	result := idx.NumericLessEqual("latency", 300)
	if len(result) != 3 {
		t.Errorf("Expected 3 records with latency<=300, got %d", len(result))
	}
}

func TestHistogram(t *testing.T) {
	records := makeRecords(100)
	idx := Build(records)

	buckets := idx.Histogram(10, nil)
	if len(buckets) != 10 {
		t.Fatalf("Expected 10 buckets, got %d", len(buckets))
	}

	// Total count across buckets should equal number of records
	total := 0
	for _, b := range buckets {
		total += b.Count
		if b.Count < 0 {
			t.Errorf("Bucket count should not be negative")
		}
	}
	if total != 100 {
		t.Errorf("Total bucket count = %d, want 100", total)
	}

	// Check that errors are counted
	totalErrors := 0
	for _, b := range buckets {
		totalErrors += b.Errors
	}
	if totalErrors == 0 {
		t.Error("Expected some errors in histogram buckets")
	}
}

func TestHistogramWithFilter(t *testing.T) {
	records := makeRecords(100)
	idx := Build(records)

	// Only errors
	errorIDs := idx.FieldLookup("level", "error")
	buckets := idx.Histogram(5, errorIDs)

	total := 0
	for _, b := range buckets {
		total += b.Count
	}
	if total != len(errorIDs) {
		t.Errorf("Filtered histogram total = %d, want %d", total, len(errorIDs))
	}
}

func TestTimeRange(t *testing.T) {
	records := makeRecords(100)
	idx := Build(records)

	base := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	start := base.Add(10 * time.Second)
	end := base.Add(20 * time.Second)

	result := idx.TimeRange(start, end)
	if len(result) == 0 {
		t.Fatal("Expected some records in time range")
	}
	for _, id := range result {
		ts := idx.Records[id].Timestamp
		if ts.Before(start) || ts.After(end) {
			t.Errorf("Record %d timestamp %v outside range [%v, %v]", id, ts, start, end)
		}
	}
}

func TestTimeRangeNoMatch(t *testing.T) {
	records := makeRecords(10)
	idx := Build(records)

	future := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	result := idx.TimeRange(future, future.Add(time.Hour))
	if len(result) != 0 {
		t.Errorf("Expected no matches, got %d", len(result))
	}
}

func TestEmptyIndex(t *testing.T) {
	idx := Build(nil)

	if idx.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", idx.TotalCount)
	}
	if len(idx.FieldLookup("level", "error")) != 0 {
		t.Error("FieldLookup on empty index should return empty")
	}
	if len(idx.NumericGreater("latency", 0)) != 0 {
		t.Error("NumericGreater on empty index should return empty")
	}
	if idx.Histogram(10, nil) != nil {
		t.Error("Histogram on empty index should return nil")
	}
}

func TestFullTextSearch(t *testing.T) {
	records := makeRecords(10)
	idx := Build(records)

	result := idx.FullTextSearch("message 5")
	if len(result) != 1 {
		t.Errorf("Expected 1 match for 'message 5', got %d", len(result))
	}

	result = idx.FullTextSearch("api")
	if len(result) == 0 {
		t.Error("Expected matches for 'api'")
	}
}

func TestIntersect(t *testing.T) {
	a := []int{1, 3, 5, 7, 9}
	b := []int{2, 3, 5, 8, 9}
	result := Intersect(a, b)
	expected := []int{3, 5, 9}

	if len(result) != len(expected) {
		t.Fatalf("Intersect length = %d, want %d", len(result), len(expected))
	}
	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("Intersect[%d] = %d, want %d", i, result[i], expected[i])
		}
	}
}

func TestUnion(t *testing.T) {
	a := []int{1, 3, 5}
	b := []int{2, 3, 6}
	result := Union(a, b)
	expected := []int{1, 2, 3, 5, 6}

	if len(result) != len(expected) {
		t.Fatalf("Union length = %d, want %d", len(result), len(expected))
	}
	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("Union[%d] = %d, want %d", i, result[i], expected[i])
		}
	}
}

func TestComplement(t *testing.T) {
	records := makeRecords(10)
	idx := Build(records)

	ids := []int{0, 2, 4, 6, 8}
	result := idx.Complement(ids)
	expected := []int{1, 3, 5, 7, 9}

	if len(result) != len(expected) {
		t.Fatalf("Complement length = %d, want %d", len(result), len(expected))
	}
	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("Complement[%d] = %d, want %d", i, result[i], expected[i])
		}
	}
}
