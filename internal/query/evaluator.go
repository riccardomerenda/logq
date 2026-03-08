package query

import (
	"regexp"
	"sort"
	"strconv"

	"github.com/riccardomerenda/logq/internal/index"
)

// Evaluate walks the AST and returns matching record indices from the index.
func Evaluate(node *Node, idx *index.Index) []int {
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
		left := Evaluate(node.Left, idx)
		right := Evaluate(node.Right, idx)
		return index.Intersect(left, right)

	case NodeOr:
		left := Evaluate(node.Left, idx)
		right := Evaluate(node.Right, idx)
		return index.Union(left, right)

	case NodeNot:
		child := Evaluate(node.Child, idx)
		return idx.Complement(child)

	default:
		return nil
	}
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
