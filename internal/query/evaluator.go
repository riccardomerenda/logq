package query

import (
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/riccardomerenda/logq/internal/index"
	"github.com/riccardomerenda/logq/internal/parser"
)

// Evaluate walks the AST and returns matching record indices from the index.
func Evaluate(node *Node, idx *index.Index) []int {
	return evaluateWithNow(node, idx, time.Now())
}

// evaluateWithNow is the internal evaluator that accepts a "now" time for testing.
func evaluateWithNow(node *Node, idx *index.Index, now time.Time) []int {
	if node == nil {
		return idx.AllIDs()
	}

	switch node.Type {
	case NodeMatchAll:
		return idx.AllIDs()

	case NodeFieldMatch:
		result := idx.FieldLookup(node.Field, node.Value)
		sort.Ints(result)
		return result

	case NodeFieldCompare:
		val, err := strconv.ParseFloat(node.Value, 64)
		if err != nil {
			return nil
		}
		var result []int
		switch node.Operator {
		case ">":
			result = idx.NumericGreater(node.Field, val)
		case ">=":
			result = idx.NumericGreaterEqual(node.Field, val)
		case "<":
			result = idx.NumericLess(node.Field, val)
		case "<=":
			result = idx.NumericLessEqual(node.Field, val)
		}
		sort.Ints(result)
		return result

	case NodeTimeCompare:
		t, err := parser.ParseTimestamp(node.Value)
		if err != nil {
			return nil
		}
		var result []int
		switch node.Operator {
		case ">":
			result = idx.TimeAfter(t)
		case ">=":
			result = idx.TimeAfterEqual(t)
		case "<":
			result = idx.TimeBefore(t)
		case "<=":
			result = idx.TimeBeforeEqual(t)
		}
		sort.Ints(result)
		return result

	case NodeRelativeTime:
		dur := parseDuration(node.Value)
		if dur == 0 {
			return nil
		}
		start := now.Add(-dur)
		result := idx.TimeAfterEqual(start)
		sort.Ints(result)
		return result

	case NodeFieldRegex:
		re, err := regexp.Compile(node.Value)
		if err != nil {
			return nil
		}
		return regexSearch(idx, node.Field, re)

	case NodeFullText:
		result := idx.FullTextSearch(node.Value)
		sort.Ints(result)
		return result

	case NodeAnd:
		left := evaluateWithNow(node.Left, idx, now)
		right := evaluateWithNow(node.Right, idx, now)
		return index.Intersect(left, right)

	case NodeOr:
		left := evaluateWithNow(node.Left, idx, now)
		right := evaluateWithNow(node.Right, idx, now)
		return index.Union(left, right)

	case NodeNot:
		child := evaluateWithNow(node.Child, idx, now)
		return idx.Complement(child)

	default:
		return nil
	}
}

// parseDuration converts "5m", "1h", "30s", "2d" to time.Duration.
func parseDuration(s string) time.Duration {
	m := durationRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	switch m[2] {
	case "s":
		return time.Duration(n) * time.Second
	case "m":
		return time.Duration(n) * time.Minute
	case "h":
		return time.Duration(n) * time.Hour
	case "d":
		return time.Duration(n) * 24 * time.Hour
	}
	return 0
}

// regexSearch scans the given field across all records for regex matches.
func regexSearch(idx *index.Index, field string, re *regexp.Regexp) []int {
	var result []int
	for i, r := range idx.Records {
		if v, ok := r.Fields[field]; ok {
			if re.MatchString(v) {
				result = append(result, i)
			}
		}
	}
	return result
}
