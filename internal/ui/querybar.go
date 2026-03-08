package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// QueryBar wraps a text input for filter queries.
type QueryBar struct {
	input    textinput.Model
	errMsg   string
	width    int
}

// NewQueryBar creates a new query bar.
func NewQueryBar() QueryBar {
	ti := textinput.New()
	ti.Placeholder = "Type a filter... (level:error AND latency>500)"
	ti.CharLimit = 500
	ti.Prompt = "Filter: "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(colorPurple)

	return QueryBar{
		input: ti,
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
}

// Blur removes focus from the query bar.
func (qb *QueryBar) Blur() {
	qb.input.Blur()
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

// View renders the query bar.
func (qb *QueryBar) View() string {
	bar := qb.input.View()
	if qb.errMsg != "" {
		bar += "\n" + StyleError.Render("  "+qb.errMsg)
	}
	return StyleQueryBar.Width(qb.width - 2).Render(bar)
}
