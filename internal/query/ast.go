package query

// NodeType represents the type of a query AST node.
type NodeType int

const (
	NodeAnd          NodeType = iota // AND of two children
	NodeOr                          // OR of two children
	NodeNot                         // NOT of a child
	NodeFieldMatch                  // field:value exact match
	NodeFieldCompare                // field>value, field>=value, etc.
	NodeFieldRegex                  // field~"pattern"
	NodeFullText                    // bare word full-text search
	NodeMatchAll                    // empty query, matches everything
	NodeTimeCompare                 // timestamp>"2026-03-08T10:00:00"
	NodeRelativeTime                // last:5m, last:1h
)

// Node is a single node in the query AST.
type Node struct {
	Type     NodeType
	Field    string // for field operations
	Operator string // ":", ">", ">=", "<", "<=", "~"
	Value    string // the value to match/compare
	Left     *Node  // for AND/OR
	Right    *Node  // for AND/OR
	Child    *Node  // for NOT
}
