package index

import (
	"sort"
	"strconv"
	"time"

	"github.com/riccardomerenda/logq/internal/parser"
)

// NumericEntry holds a float64 value and the index of the record it belongs to.
type NumericEntry struct {
	Value    float64
	RecordID int
}

// TimeEntry holds a timestamp and the index of the record it belongs to.
type TimeEntry struct {
	Time     time.Time
	RecordID int
}

// HistogramBucket represents one bar in the time histogram.
type HistogramBucket struct {
	Start  time.Time
	End    time.Time
	Count  int
	Errors int // count of error/fatal in this bucket
}

// Index provides fast lookups over a set of parsed log records.
type Index struct {
	Records      []parser.Record
	fieldIndex   map[string]map[string][]int // field → value → record IDs
	numericIndex map[string][]NumericEntry   // field → sorted entries
	timeIndex    []TimeEntry                 // sorted by timestamp
	TotalCount   int
}

// Build creates an index from a slice of records.
func Build(records []parser.Record) *Index {
	idx := &Index{
		Records:      records,
		fieldIndex:   make(map[string]map[string][]int),
		numericIndex: make(map[string][]NumericEntry),
		TotalCount:   len(records),
	}

	for i, r := range records {
		idx.addToIndex(r, i)
	}

	// Sort numeric indexes
	for _, entries := range idx.numericIndex {
		sort.Slice(entries, func(a, b int) bool {
			return entries[a].Value < entries[b].Value
		})
	}

	// Sort time index
	sort.Slice(idx.timeIndex, func(a, b int) bool {
		return idx.timeIndex[a].Time.Before(idx.timeIndex[b].Time)
	})

	return idx
}

func (idx *Index) addToIndex(r parser.Record, id int) {
	// Index all fields
	for k, v := range r.Fields {
		if _, ok := idx.fieldIndex[k]; !ok {
			idx.fieldIndex[k] = make(map[string][]int)
		}
		idx.fieldIndex[k][v] = append(idx.fieldIndex[k][v], id)

		// Try numeric indexing
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			idx.numericIndex[k] = append(idx.numericIndex[k], NumericEntry{Value: f, RecordID: id})
		}
	}

	// Also index the normalized level if it differs from the raw value in Fields
	if r.Level != "" {
		raw := r.Fields["level"]
		if raw != r.Level {
			if _, ok := idx.fieldIndex["level"]; !ok {
				idx.fieldIndex["level"] = make(map[string][]int)
			}
			idx.fieldIndex["level"][r.Level] = append(idx.fieldIndex["level"][r.Level], id)
		}
	}

	// Time index
	if !r.Timestamp.IsZero() {
		idx.timeIndex = append(idx.timeIndex, TimeEntry{Time: r.Timestamp, RecordID: id})
	}
}

// AddRecords appends new records to the index incrementally.
func (idx *Index) AddRecords(records []parser.Record) {
	startID := len(idx.Records)
	idx.Records = append(idx.Records, records...)

	for i, r := range records {
		idx.addToIndex(r, startID+i)
	}

	// Re-sort numeric indices
	for _, entries := range idx.numericIndex {
		sort.Slice(entries, func(a, b int) bool {
			return entries[a].Value < entries[b].Value
		})
	}

	// Re-sort time index
	sort.Slice(idx.timeIndex, func(a, b int) bool {
		return idx.timeIndex[a].Time.Before(idx.timeIndex[b].Time)
	})

	idx.TotalCount = len(idx.Records)
}

// FieldLookup returns record IDs where field has the given value.
func (idx *Index) FieldLookup(field, value string) []int {
	if vals, ok := idx.fieldIndex[field]; ok {
		if ids, ok := vals[value]; ok {
			return ids
		}
	}
	return nil
}

// NumericGreater returns record IDs where field > value.
func (idx *Index) NumericGreater(field string, value float64) []int {
	return idx.numericRange(field, value, false, 0, false)
}

// NumericGreaterEqual returns record IDs where field >= value.
func (idx *Index) NumericGreaterEqual(field string, value float64) []int {
	return idx.numericRange(field, value, true, 0, false)
}

// NumericLess returns record IDs where field < value.
func (idx *Index) NumericLess(field string, value float64) []int {
	return idx.numericRangeLess(field, value, false)
}

// NumericLessEqual returns record IDs where field <= value.
func (idx *Index) NumericLessEqual(field string, value float64) []int {
	return idx.numericRangeLess(field, value, true)
}

func (idx *Index) numericRange(field string, min float64, includeMin bool, max float64, hasMax bool) []int {
	entries, ok := idx.numericIndex[field]
	if !ok {
		return nil
	}

	// Binary search for start position
	start := sort.Search(len(entries), func(i int) bool {
		if includeMin {
			return entries[i].Value >= min
		}
		return entries[i].Value > min
	})

	var result []int
	for i := start; i < len(entries); i++ {
		if hasMax && entries[i].Value > max {
			break
		}
		result = append(result, entries[i].RecordID)
	}
	return result
}

func (idx *Index) numericRangeLess(field string, max float64, includeMax bool) []int {
	entries, ok := idx.numericIndex[field]
	if !ok {
		return nil
	}

	var result []int
	for _, e := range entries {
		if includeMax && e.Value <= max {
			result = append(result, e.RecordID)
		} else if !includeMax && e.Value < max {
			result = append(result, e.RecordID)
		} else {
			break
		}
	}
	return result
}

// TimeRange returns record IDs within [start, end].
func (idx *Index) TimeRange(start, end time.Time) []int {
	if len(idx.timeIndex) == 0 {
		return nil
	}

	lo := sort.Search(len(idx.timeIndex), func(i int) bool {
		return !idx.timeIndex[i].Time.Before(start)
	})

	var result []int
	for i := lo; i < len(idx.timeIndex); i++ {
		if idx.timeIndex[i].Time.After(end) {
			break
		}
		result = append(result, idx.timeIndex[i].RecordID)
	}
	return result
}

// Histogram returns bucketed counts for the time range of the given record IDs.
// If ids is nil, uses all records with timestamps.
func (idx *Index) Histogram(buckets int, ids []int) []HistogramBucket {
	if buckets <= 0 || len(idx.timeIndex) == 0 {
		return nil
	}

	// Determine which records to include
	var entries []TimeEntry
	if ids == nil {
		entries = idx.timeIndex
	} else {
		idSet := make(map[int]bool, len(ids))
		for _, id := range ids {
			idSet[id] = true
		}
		for _, te := range idx.timeIndex {
			if idSet[te.RecordID] {
				entries = append(entries, te)
			}
		}
	}

	if len(entries) == 0 {
		return nil
	}

	minTime := entries[0].Time
	maxTime := entries[len(entries)-1].Time

	// Add a small buffer so the last entry falls within the range
	duration := maxTime.Sub(minTime)
	if duration == 0 {
		duration = time.Second
	}
	bucketDuration := duration / time.Duration(buckets)
	if bucketDuration == 0 {
		bucketDuration = time.Millisecond
	}

	result := make([]HistogramBucket, buckets)
	for i := range result {
		result[i].Start = minTime.Add(bucketDuration * time.Duration(i))
		result[i].End = minTime.Add(bucketDuration * time.Duration(i+1))
	}

	for _, te := range entries {
		bi := int(te.Time.Sub(minTime) / bucketDuration)
		if bi >= buckets {
			bi = buckets - 1
		}
		result[bi].Count++

		// Count errors
		r := idx.Records[te.RecordID]
		if r.Level == "error" || r.Level == "fatal" {
			result[bi].Errors++
		}
	}

	return result
}

// FullTextSearch scans all records for a substring match in any field value.
// Returns matching record IDs.
func (idx *Index) FullTextSearch(text string) []int {
	var result []int
	for i, r := range idx.Records {
		for _, v := range r.Fields {
			if containsCI(v, text) {
				result = append(result, i)
				break
			}
		}
	}
	return result
}

// AllIDs returns all record indices.
func (idx *Index) AllIDs() []int {
	ids := make([]int, idx.TotalCount)
	for i := range ids {
		ids[i] = i
	}
	return ids
}

// containsCI does a case-insensitive substring check.
func containsCI(s, substr string) bool {
	ls := len(s)
	lsub := len(substr)
	if lsub > ls {
		return false
	}
	for i := 0; i <= ls-lsub; i++ {
		match := true
		for j := 0; j < lsub; j++ {
			sc := s[i+j]
			tc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if tc >= 'A' && tc <= 'Z' {
				tc += 32
			}
			if sc != tc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// Intersect returns the sorted intersection of two sorted slices.
func Intersect(a, b []int) []int {
	var result []int
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			result = append(result, a[i])
			i++
			j++
		} else if a[i] < b[j] {
			i++
		} else {
			j++
		}
	}
	return result
}

// Union returns the sorted union of two sorted slices.
func Union(a, b []int) []int {
	var result []int
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			result = append(result, a[i])
			i++
			j++
		} else if a[i] < b[j] {
			result = append(result, a[i])
			i++
		} else {
			result = append(result, b[j])
			j++
		}
	}
	result = append(result, a[i:]...)
	result = append(result, b[j:]...)
	return result
}

// Complement returns all IDs not in the given sorted slice.
func (idx *Index) Complement(ids []int) []int {
	set := make(map[int]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	var result []int
	for i := 0; i < idx.TotalCount; i++ {
		if !set[i] {
			result = append(result, i)
		}
	}
	return result
}
