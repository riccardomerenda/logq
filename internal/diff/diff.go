package diff

import (
	"fmt"
	"math"
	"sort"

	"github.com/riccardomerenda/logq/internal/parser"
	"github.com/riccardomerenda/logq/internal/pattern"
)

// Result holds the comparison between two sets of log records.
type Result struct {
	LeftName      string
	RightName     string
	LeftCount     int
	RightCount    int
	LeftPatterns  int
	RightPatterns int
	Levels        []LevelDiff
	NewPatterns   []PatternDiff
	GonePatterns  []PatternDiff
	Changed       []PatternDiff
}

// LevelDiff shows the count of a log level in each set.
type LevelDiff struct {
	Level      string
	LeftCount  int
	RightCount int
}

// PatternDiff shows a pattern template and its count in each set.
type PatternDiff struct {
	Template   string
	LeftCount  int
	RightCount int
}

// Compare analyzes two sets of log records and returns their differences.
func Compare(leftRecords, rightRecords []parser.Record, leftIDs, rightIDs []int) Result {
	leftClusters := pattern.Clusterize(leftRecords, leftIDs)
	rightClusters := pattern.Clusterize(rightRecords, rightIDs)

	leftMap := make(map[string]int)
	for _, c := range leftClusters {
		leftMap[c.Template] = c.Count
	}
	rightMap := make(map[string]int)
	for _, c := range rightClusters {
		rightMap[c.Template] = c.Count
	}

	var newPatterns, gonePatterns, changed []PatternDiff

	for _, c := range rightClusters {
		leftCount, exists := leftMap[c.Template]
		if !exists {
			newPatterns = append(newPatterns, PatternDiff{Template: c.Template, RightCount: c.Count})
		} else {
			changed = append(changed, PatternDiff{Template: c.Template, LeftCount: leftCount, RightCount: c.Count})
		}
	}

	for _, c := range leftClusters {
		if _, exists := rightMap[c.Template]; !exists {
			gonePatterns = append(gonePatterns, PatternDiff{Template: c.Template, LeftCount: c.Count})
		}
	}

	// Sort: new/gone by count desc, changed by absolute % change desc
	sort.Slice(newPatterns, func(i, j int) bool { return newPatterns[i].RightCount > newPatterns[j].RightCount })
	sort.Slice(gonePatterns, func(i, j int) bool { return gonePatterns[i].LeftCount > gonePatterns[j].LeftCount })
	sort.Slice(changed, func(i, j int) bool {
		return math.Abs(ChangePercent(changed[i].LeftCount, changed[i].RightCount)) >
			math.Abs(ChangePercent(changed[j].LeftCount, changed[j].RightCount))
	})

	// Level distribution
	leftLevels := countByLevel(leftRecords, leftIDs)
	rightLevels := countByLevel(rightRecords, rightIDs)
	allLevels := []string{"fatal", "error", "warn", "info", "debug"}
	var levels []LevelDiff
	for _, level := range allLevels {
		l, r := leftLevels[level], rightLevels[level]
		if l > 0 || r > 0 {
			levels = append(levels, LevelDiff{Level: level, LeftCount: l, RightCount: r})
		}
	}

	return Result{
		LeftCount:     len(leftIDs),
		RightCount:    len(rightIDs),
		LeftPatterns:  len(leftClusters),
		RightPatterns: len(rightClusters),
		Levels:        levels,
		NewPatterns:   newPatterns,
		GonePatterns:  gonePatterns,
		Changed:       changed,
	}
}

func countByLevel(records []parser.Record, ids []int) map[string]int {
	counts := make(map[string]int)
	for _, id := range ids {
		if id >= 0 && id < len(records) {
			if level := records[id].Level; level != "" {
				counts[level]++
			}
		}
	}
	return counts
}

// ChangePercent returns the percentage change from before to after.
func ChangePercent(before, after int) float64 {
	if before == 0 {
		if after == 0 {
			return 0
		}
		return 100
	}
	return float64(after-before) / float64(before) * 100
}

// FormatChange returns a human-readable change string like "+150%" or "-30%".
func FormatChange(before, after int) string {
	if before == 0 && after == 0 {
		return "-"
	}
	if before == 0 {
		return "new"
	}
	pct := ChangePercent(before, after)
	if pct == 0 {
		return "0%"
	}
	if pct > 0 {
		return fmt.Sprintf("+%.0f%%", pct)
	}
	return fmt.Sprintf("%.0f%%", pct)
}
