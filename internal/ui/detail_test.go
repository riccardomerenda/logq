package ui

import (
	"encoding/json"
	"testing"
)

func TestHasNestedObjects(t *testing.T) {
	tests := []struct {
		name string
		json string
		want bool
	}{
		{"flat", `{"a":"1","b":"2"}`, false},
		{"nested object", `{"a":"1","b":{"c":"2"}}`, true},
		{"nested array", `{"a":"1","b":[1,2]}`, true},
		{"empty", `{}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(tt.json), &obj); err != nil {
				t.Fatal(err)
			}
			if got := hasNestedObjects(obj); got != tt.want {
				t.Errorf("hasNestedObjects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildTreeLines(t *testing.T) {
	jsonStr := `{"level":"error","request":{"id":"req_123","method":"POST"},"tags":["a","b"]}`
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		t.Fatal(err)
	}

	d := &DetailView{
		jsonObj:  obj,
		expanded: make(map[string]bool),
	}
	initExpanded("", obj, d.expanded)

	lines := d.buildTreeLines()
	if len(lines) == 0 {
		t.Fatal("expected tree lines, got none")
	}

	// Check that "level" is a leaf
	found := false
	for _, l := range lines {
		if l.key == "level" {
			found = true
			if l.isNode {
				t.Error("level should be a leaf")
			}
			if l.value != "error" {
				t.Errorf("level value = %q, want %q", l.value, "error")
			}
		}
	}
	if !found {
		t.Error("level key not found in tree lines")
	}

	// Check that "request" is a node
	for _, l := range lines {
		if l.key == "request" {
			if !l.isNode {
				t.Error("request should be a node")
			}
			if l.childCount != 2 {
				t.Errorf("request childCount = %d, want 2", l.childCount)
			}
		}
	}

	// Check that "tags" is a node (array)
	for _, l := range lines {
		if l.key == "tags" {
			if !l.isNode {
				t.Error("tags should be a node")
			}
			if l.childCount != 2 {
				t.Errorf("tags childCount = %d, want 2", l.childCount)
			}
		}
	}
}

func TestBuildTreeLinesCollapsed(t *testing.T) {
	jsonStr := `{"a":"1","nested":{"b":"2","c":"3"}}`
	var obj map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &obj)

	d := &DetailView{
		jsonObj:  obj,
		expanded: make(map[string]bool),
	}
	initExpanded("", obj, d.expanded)

	// All expanded: a, nested, nested.b, nested.c = 4 lines
	lines := d.buildTreeLines()
	if len(lines) != 4 {
		t.Errorf("expanded: got %d lines, want 4", len(lines))
	}

	// Collapse nested
	d.expanded["nested"] = false
	lines = d.buildTreeLines()
	if len(lines) != 2 {
		t.Errorf("collapsed: got %d lines, want 2 (a + nested)", len(lines))
	}
}

func TestTreeToggle(t *testing.T) {
	jsonStr := `{"nested":{"a":"1","b":"2"}}`
	var obj map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &obj)

	d := &DetailView{
		jsonObj:    obj,
		expanded:   make(map[string]bool),
		treeCursor: 0,
		height:     40,
	}
	initExpanded("", obj, d.expanded)

	// Initially expanded: nested, nested.a, nested.b = 3 lines
	lines := d.buildTreeLines()
	if len(lines) != 3 {
		t.Fatalf("initial: got %d lines, want 3", len(lines))
	}

	// Toggle collapse
	d.TreeToggle()
	lines = d.buildTreeLines()
	if len(lines) != 1 {
		t.Errorf("after collapse: got %d lines, want 1", len(lines))
	}

	// Toggle expand
	d.TreeToggle()
	lines = d.buildTreeLines()
	if len(lines) != 3 {
		t.Errorf("after expand: got %d lines, want 3", len(lines))
	}
}

func TestTreeCursorClamp(t *testing.T) {
	jsonStr := `{"nested":{"a":"1","b":"2","c":"3"}}`
	var obj map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &obj)

	d := &DetailView{
		jsonObj:    obj,
		expanded:   make(map[string]bool),
		treeCursor: 0,
		height:     40,
	}
	initExpanded("", obj, d.expanded)

	// Collapse "nested" - cursor should stay at 0
	d.TreeToggle()
	if d.treeCursor != 0 {
		t.Errorf("after collapse, cursor = %d, want 0", d.treeCursor)
	}

	// Expand, move cursor to last child, then collapse
	d.TreeToggle() // expand
	d.treeCursor = 3 // on "nested.c"
	d.treeCursor = 0 // move back to node
	d.TreeToggle()   // collapse
	lines := d.buildTreeLines()
	if d.treeCursor >= len(lines) {
		t.Errorf("cursor %d out of bounds (%d lines)", d.treeCursor, len(lines))
	}
}

func TestFormatJSONValue(t *testing.T) {
	tests := []struct {
		input interface{}
		want  string
	}{
		{"hello", "hello"},
		{float64(42), "42"},
		{float64(3.14), "3.14"},
		{true, "true"},
		{false, "false"},
		{nil, "null"},
	}
	for _, tt := range tests {
		got := formatJSONValue(tt.input)
		if got != tt.want {
			t.Errorf("formatJSONValue(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTreeDotPath(t *testing.T) {
	jsonStr := `{"request":{"headers":{"content-type":"application/json"}}}`
	var obj map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &obj)

	d := &DetailView{
		jsonObj:    obj,
		expanded:   make(map[string]bool),
		treeCursor: 0,
		height:     40,
	}
	initExpanded("", obj, d.expanded)

	// Cursor at 0 = "request"
	path := d.SelectedDotPath()
	if path != "request" {
		t.Errorf("dot path at 0 = %q, want %q", path, "request")
	}

	// Move to content-type (depth 2): request, headers, content-type
	d.treeCursor = 2
	path = d.SelectedDotPath()
	if path != "request.headers.content-type" {
		t.Errorf("dot path at 2 = %q, want %q", path, "request.headers.content-type")
	}
}

func TestTreeCollapseMoveToParent(t *testing.T) {
	jsonStr := `{"parent":{"child":"value"}}`
	var obj map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &obj)

	d := &DetailView{
		jsonObj:    obj,
		expanded:   make(map[string]bool),
		treeCursor: 1, // on "child" leaf
		height:     40,
	}
	initExpanded("", obj, d.expanded)

	// Collapse on a leaf should move to parent
	d.TreeCollapse()
	if d.treeCursor != 0 {
		t.Errorf("after collapse on leaf, cursor = %d, want 0 (parent)", d.treeCursor)
	}
}

func TestIsTreeModeFlat(t *testing.T) {
	d := &DetailView{}
	if d.IsTreeMode() {
		t.Error("empty detail view should not be tree mode")
	}
}
