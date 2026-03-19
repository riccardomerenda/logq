package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/riccardomerenda/logq/internal/parser"
	"github.com/riccardomerenda/logq/internal/trace"
)

// treeLine represents a single line in the JSON tree view.
type treeLine struct {
	dotPath    string
	key        string
	value      string
	depth      int
	isNode     bool
	childCount int
}

// DetailView renders a full-screen overlay showing all fields of a record.
type DetailView struct {
	record     *parser.Record
	width      int
	height     int
	copyMsg    string
	highlights []HighlightTerm

	// Pick mode for trace ID selection
	pickMode   bool
	pickItems  []trace.IDField
	pickCursor int

	// JSON tree mode
	jsonObj    map[string]interface{}
	expanded   map[string]bool
	treeCursor int
	treeScroll int
}

// NewDetailView creates a new detail view.
func NewDetailView() DetailView {
	return DetailView{}
}

// SetRecord sets the record to display.
func (d *DetailView) SetRecord(r *parser.Record) {
	d.record = r
	d.treeCursor = 0
	d.treeScroll = 0
	d.jsonObj = nil
	d.expanded = nil

	// Try to parse as JSON for tree mode
	if raw := strings.TrimSpace(r.Raw); len(raw) > 0 && raw[0] == '{' {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &obj); err == nil {
			if hasNestedObjects(obj) {
				d.jsonObj = obj
				d.expanded = make(map[string]bool)
				initExpanded("", obj, d.expanded)
			}
		}
	}
}

// SetHighlights sets the terms to highlight in the detail view.
func (d *DetailView) SetHighlights(h []HighlightTerm) {
	d.highlights = h
}

// SetSize updates the overlay dimensions.
func (d *DetailView) SetSize(w, h int) {
	d.width = w
	d.height = h
	if d.jsonObj != nil {
		d.ensureTreeVisible()
	}
}

// EnterPickMode shows a selection list of trace ID fields.
func (d *DetailView) EnterPickMode(items []trace.IDField) {
	d.pickMode = true
	d.pickItems = items
	d.pickCursor = 0
}

// ExitPickMode returns to normal detail view.
func (d *DetailView) ExitPickMode() {
	d.pickMode = false
	d.pickItems = nil
	d.pickCursor = 0
}

// PickUp moves the pick cursor up.
func (d *DetailView) PickUp() {
	if d.pickCursor > 0 {
		d.pickCursor--
	}
}

// PickDown moves the pick cursor down.
func (d *DetailView) PickDown() {
	if d.pickCursor < len(d.pickItems)-1 {
		d.pickCursor++
	}
}

// PickSelected returns the currently selected trace ID field.
func (d *DetailView) PickSelected() trace.IDField {
	return d.pickItems[d.pickCursor]
}

// IsTreeMode returns true if the detail view is showing a JSON tree.
func (d *DetailView) IsTreeMode() bool {
	return d.jsonObj != nil
}

// TreeUp moves the tree cursor up.
func (d *DetailView) TreeUp() {
	if d.treeCursor > 0 {
		d.treeCursor--
	}
	d.ensureTreeVisible()
}

// TreeDown moves the tree cursor down.
func (d *DetailView) TreeDown() {
	lines := d.buildTreeLines()
	if d.treeCursor < len(lines)-1 {
		d.treeCursor++
	}
	d.ensureTreeVisible()
}

// TreeToggle toggles expand/collapse on the selected node.
func (d *DetailView) TreeToggle() {
	lines := d.buildTreeLines()
	if d.treeCursor >= len(lines) {
		return
	}
	line := lines[d.treeCursor]
	if !line.isNode {
		return
	}
	d.expanded[line.dotPath] = !d.expanded[line.dotPath]
	// Clamp cursor if collapse reduced visible lines
	newLines := d.buildTreeLines()
	if d.treeCursor >= len(newLines) {
		d.treeCursor = len(newLines) - 1
	}
	d.ensureTreeVisible()
}

// TreeExpand expands the selected node.
func (d *DetailView) TreeExpand() {
	lines := d.buildTreeLines()
	if d.treeCursor >= len(lines) {
		return
	}
	line := lines[d.treeCursor]
	if line.isNode && !d.expanded[line.dotPath] {
		d.expanded[line.dotPath] = true
		d.ensureTreeVisible()
	}
}

// TreeCollapse collapses the selected node or moves cursor to parent.
func (d *DetailView) TreeCollapse() {
	lines := d.buildTreeLines()
	if d.treeCursor >= len(lines) {
		return
	}
	line := lines[d.treeCursor]
	if line.isNode && d.expanded[line.dotPath] {
		d.expanded[line.dotPath] = false
		newLines := d.buildTreeLines()
		if d.treeCursor >= len(newLines) {
			d.treeCursor = len(newLines) - 1
		}
		d.ensureTreeVisible()
		return
	}
	// Move to parent node
	dotIdx := strings.LastIndex(line.dotPath, ".")
	bracketIdx := strings.LastIndex(line.dotPath, "[")
	parentEnd := dotIdx
	if bracketIdx > dotIdx {
		parentEnd = bracketIdx
	}
	if parentEnd > 0 {
		parentPath := line.dotPath[:parentEnd]
		for i, l := range lines {
			if l.dotPath == parentPath {
				d.treeCursor = i
				d.ensureTreeVisible()
				return
			}
		}
	}
}

// SelectedDotPath returns the dot path of the currently selected tree line.
func (d *DetailView) SelectedDotPath() string {
	lines := d.buildTreeLines()
	if d.treeCursor >= len(lines) {
		return ""
	}
	return lines[d.treeCursor].dotPath
}

func (d *DetailView) treeVisibleHeight() int {
	h := d.height - 12
	if h < 3 {
		h = 3
	}
	return h
}

func (d *DetailView) ensureTreeVisible() {
	vh := d.treeVisibleHeight()
	if d.treeCursor < d.treeScroll {
		d.treeScroll = d.treeCursor
	}
	if d.treeCursor >= d.treeScroll+vh {
		d.treeScroll = d.treeCursor - vh + 1
	}
}

// View renders the detail overlay.
func (d DetailView) View() string {
	if d.record == nil {
		return ""
	}
	if d.pickMode {
		return d.viewPickMode()
	}
	if d.jsonObj != nil {
		return d.viewTree()
	}
	return d.viewFlat()
}

// viewFlat renders the traditional flat key-value detail view.
func (d DetailView) viewFlat() string {
	r := d.record
	innerWidth := d.width - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	var b strings.Builder

	title := fmt.Sprintf(" Record #%d ", r.LineNumber)
	b.WriteString(StyleTitle.Render(title))
	b.WriteString("\n\n")

	// Collect all fields, sorted
	var keys []string
	for k := range r.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Find max key length for alignment
	maxKeyLen := 0
	for _, k := range keys {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	valStyle := StyleBase.Copy().Foreground(colorWhite)
	for _, k := range keys {
		v := r.Fields[k]
		keyStr := fmt.Sprintf("  %-*s", maxKeyLen+2, k)
		b.WriteString(StyleDim.Render(keyStr))
		b.WriteString(highlightText(v, d.highlights, valStyle, k))
		b.WriteString("\n")
	}

	// Raw line
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  Raw:"))
	b.WriteString("\n")

	raw := r.Raw
	if len(raw) > innerWidth*3 {
		raw = raw[:innerWidth*3] + "..."
	}
	b.WriteString(StyleDim.Render("  " + raw))
	b.WriteString("\n\n")
	hint := "  Esc close  c copy  t trace"
	if d.copyMsg != "" {
		hint = "  " + d.copyMsg
	}
	b.WriteString(StyleDim.Render(hint))

	content := b.String()

	return StyleBase.Copy().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple).
		Padding(1, 2).
		Width(d.width - 4).
		Render(content)
}

// viewTree renders the JSON tree view with collapsible nodes.
func (d DetailView) viewTree() string {
	r := d.record
	innerWidth := d.width - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	var b strings.Builder

	title := fmt.Sprintf(" Record #%d ", r.LineNumber)
	b.WriteString(StyleTitle.Render(title))
	b.WriteString("\n\n")

	lines := d.buildTreeLines()
	visibleHeight := d.treeVisibleHeight()

	startIdx := d.treeScroll
	endIdx := startIdx + visibleHeight
	if startIdx > len(lines) {
		startIdx = len(lines)
	}
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	nodeStyle := StyleBase.Copy().Foreground(colorCyan)
	valStyle := StyleBase.Copy().Foreground(colorWhite)
	cursorPrefix := StyleBase.Copy().Foreground(colorGreen).Bold(true)

	for i := startIdx; i < endIdx; i++ {
		line := lines[i]
		indent := strings.Repeat("  ", line.depth)

		var prefix string
		if line.isNode {
			if d.expanded[line.dotPath] {
				prefix = "\u25BC " // ▼
			} else {
				prefix = "\u25B8 " // ▸
			}
		} else {
			prefix = "  "
		}

		// Cursor indicator
		linePrefix := "  "
		if i == d.treeCursor {
			linePrefix = cursorPrefix.Render("> ")
		}

		keyPart := indent + prefix + line.key

		var rendered string
		if line.isNode && !d.expanded[line.dotPath] {
			summary := fmt.Sprintf(" {%d}", line.childCount)
			rendered = nodeStyle.Render(keyPart) + StyleDim.Render(summary)
		} else if line.value != "" {
			highlighted := highlightText(line.value, d.highlights, valStyle, line.key)
			rendered = StyleDim.Render(keyPart+"  ") + highlighted
		} else {
			rendered = nodeStyle.Render(keyPart)
		}

		b.WriteString(linePrefix)
		b.WriteString(rendered)
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(lines) > visibleHeight {
		scrollInfo := fmt.Sprintf("  [%d/%d]", d.treeCursor+1, len(lines))
		b.WriteString(StyleDim.Render(scrollInfo))
		b.WriteString("\n")
	}

	// Raw line
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  Raw:"))
	b.WriteString("\n")
	raw := r.Raw
	if len(raw) > innerWidth*3 {
		raw = raw[:innerWidth*3] + "..."
	}
	b.WriteString(StyleDim.Render("  " + raw))
	b.WriteString("\n\n")

	hint := "  Esc close  c copy  t trace  Enter fold  d path"
	if d.copyMsg != "" {
		hint = "  " + d.copyMsg
	}
	b.WriteString(StyleDim.Render(hint))

	content := b.String()

	return StyleBase.Copy().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple).
		Padding(1, 2).
		Width(d.width - 4).
		Render(content)
}

// viewPickMode renders the trace ID selection menu.
func (d DetailView) viewPickMode() string {
	var b strings.Builder

	b.WriteString(StyleTitle.Render(" Select trace ID "))
	b.WriteString("\n\n")

	for i, item := range d.pickItems {
		prefix := "  "
		if i == d.pickCursor {
			prefix = StyleBase.Copy().Foreground(colorGreen).Bold(true).Render("> ")
		}
		nameStyle := StyleBase.Copy().Foreground(colorCyan)
		valStyle := StyleBase.Copy().Foreground(colorWhite)

		line := prefix + nameStyle.Render(item.Name) + StyleDim.Render(" = ") + valStyle.Render(item.Value)
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  Up/Down select  Enter confirm  Esc cancel"))

	content := b.String()

	return StyleBase.Copy().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple).
		Padding(1, 2).
		Width(d.width - 4).
		Render(content)
}

// hasNestedObjects returns true if the JSON object contains nested objects or arrays.
func hasNestedObjects(obj map[string]interface{}) bool {
	for _, v := range obj {
		switch v.(type) {
		case map[string]interface{}, []interface{}:
			return true
		}
	}
	return false
}

// initExpanded marks all nodes as expanded initially.
func initExpanded(prefix string, obj map[string]interface{}, expanded map[string]bool) {
	for k, v := range obj {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		switch cv := v.(type) {
		case map[string]interface{}:
			expanded[path] = true
			initExpanded(path, cv, expanded)
		case []interface{}:
			expanded[path] = true
			for i, item := range cv {
				if m, ok := item.(map[string]interface{}); ok {
					arrayPath := fmt.Sprintf("%s[%d]", path, i)
					expanded[arrayPath] = true
					initExpanded(arrayPath, m, expanded)
				}
			}
		}
	}
}

// buildTreeLines returns the visible tree lines based on expanded state.
func (d DetailView) buildTreeLines() []treeLine {
	if d.jsonObj == nil {
		return nil
	}
	var lines []treeLine
	walkJSON("", d.jsonObj, 0, d.expanded, &lines)
	return lines
}

// walkJSON recursively builds tree lines from a JSON value.
func walkJSON(prefix string, obj interface{}, depth int, expanded map[string]bool, lines *[]treeLine) {
	switch v := obj.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			child := v[k]
			switch cv := child.(type) {
			case map[string]interface{}:
				*lines = append(*lines, treeLine{dotPath: path, key: k, depth: depth, isNode: true, childCount: len(cv)})
				if expanded[path] {
					walkJSON(path, cv, depth+1, expanded, lines)
				}
			case []interface{}:
				*lines = append(*lines, treeLine{dotPath: path, key: k, depth: depth, isNode: true, childCount: len(cv)})
				if expanded[path] {
					walkJSON(path, cv, depth+1, expanded, lines)
				}
			default:
				*lines = append(*lines, treeLine{dotPath: path, key: k, value: formatJSONValue(child), depth: depth})
			}
		}
	case []interface{}:
		for i, item := range v {
			path := fmt.Sprintf("%s[%d]", prefix, i)
			switch cv := item.(type) {
			case map[string]interface{}:
				*lines = append(*lines, treeLine{dotPath: path, key: fmt.Sprintf("[%d]", i), depth: depth, isNode: true, childCount: len(cv)})
				if expanded[path] {
					walkJSON(path, cv, depth+1, expanded, lines)
				}
			case []interface{}:
				*lines = append(*lines, treeLine{dotPath: path, key: fmt.Sprintf("[%d]", i), depth: depth, isNode: true, childCount: len(cv)})
				if expanded[path] {
					walkJSON(path, cv, depth+1, expanded, lines)
				}
			default:
				*lines = append(*lines, treeLine{dotPath: path, key: fmt.Sprintf("[%d]", i), value: formatJSONValue(item), depth: depth})
			}
		}
	}
}

// formatJSONValue formats a JSON value for display.
func formatJSONValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", val)
	}
}
