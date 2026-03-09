package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// QueryBar wraps a text input for filter queries with history support.
type QueryBar struct {
	input      textinput.Model
	errMsg     string
	width      int
	history    []string
	historyIdx int  // -1 means "not browsing history"
	draft      string // preserves what the user was typing before browsing history
}

// NewQueryBar creates a new query bar.
func NewQueryBar() QueryBar {
	ti := textinput.New()
	ti.Placeholder = "Type a filter... (level:error AND latency>500)"
	ti.CharLimit = 500
	ti.Prompt = "Filter: "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(colorPurple)

	return QueryBar{
		input:      ti,
		historyIdx: -1,
	}
}

// SetWidth updates the query bar width.
func (qb *QueryBar) SetWidth(w int) {
	qb.width = w
	qb.input.Width = w - 12 // account for prompt and padding
}

// Focus gives focus to the query bar.
func (qb *QueryBar) Focus() {
	qb.input.Focus()
	qb.historyIdx = -1
}

// Blur removes focus from the query bar.
func (qb *QueryBar) Blur() {
	qb.input.Blur()
	qb.historyIdx = -1
}

// Focused returns whether the query bar has focus.
func (qb *QueryBar) Focused() bool {
	return qb.input.Focused()
}

// Value returns the current query text.
func (qb *QueryBar) Value() string {
	return qb.input.Value()
}

// SetValue sets the query text.
func (qb *QueryBar) SetValue(v string) {
	qb.input.SetValue(v)
}

// SetError sets the error message.
func (qb *QueryBar) SetError(msg string) {
	qb.errMsg = msg
}

// TextInput returns the underlying textinput model for update handling.
func (qb *QueryBar) TextInput() *textinput.Model {
	return &qb.input
}

// PushHistory adds a query to the history (deduplicates consecutive entries).
func (qb *QueryBar) PushHistory(q string) {
	if q == "" {
		return
	}
	if len(qb.history) > 0 && qb.history[len(qb.history)-1] == q {
		return
	}
	qb.history = append(qb.history, q)
	// Cap at 100 entries
	if len(qb.history) > 100 {
		qb.history = qb.history[len(qb.history)-100:]
	}
	qb.historyIdx = -1
}

// HistoryUp loads the previous history entry. Returns true if handled.
func (qb *QueryBar) HistoryUp() bool {
	if len(qb.history) == 0 {
		return false
	}
	if qb.historyIdx == -1 {
		// Entering history mode: save current draft
		qb.draft = qb.input.Value()
		qb.historyIdx = len(qb.history) - 1
	} else if qb.historyIdx > 0 {
		qb.historyIdx--
	} else {
		return true // already at oldest entry
	}
	qb.input.SetValue(qb.history[qb.historyIdx])
	qb.input.CursorEnd()
	return true
}

// HistoryDown loads the next history entry. Returns true if handled.
func (qb *QueryBar) HistoryDown() bool {
	if qb.historyIdx == -1 {
		return false
	}
	if qb.historyIdx < len(qb.history)-1 {
		qb.historyIdx++
		qb.input.SetValue(qb.history[qb.historyIdx])
		qb.input.CursorEnd()
	} else {
		// Past the end: restore draft
		qb.historyIdx = -1
		qb.input.SetValue(qb.draft)
		qb.input.CursorEnd()
	}
	return true
}

// View renders the query bar.
func (qb *QueryBar) View() string {
	bar := qb.input.View()
	if qb.errMsg != "" {
		bar += "\n" + StyleError.Render("  "+qb.errMsg)
	}
	return StyleQueryBar.Width(qb.width - 2).Render(bar)
}
