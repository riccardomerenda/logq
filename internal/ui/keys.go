package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
	Left     key.Binding
	Right    key.Binding
	Search   key.Binding
	Enter    key.Binding
	Escape   key.Binding
	Tab      key.Binding
	Quit     key.Binding
	Copy       key.Binding
	CopyPath   key.Binding
	Save       key.Binding
	Trace          key.Binding
	TraceClear     key.Binding
	Pattern        key.Binding
	BookmarkToggle key.Binding
	BookmarkNext   key.Binding
	BookmarkFilter key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("up/k", "scroll up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("down/j", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("home", "go to start"),
		),
		End: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("end", "go to end"),
		),
		Left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("left", "collapse"),
		),
		Right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("right", "expand"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select/execute"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/clear"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle focus"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Copy: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy raw"),
		),
		CopyPath: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "copy path"),
		),
		Save: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "save results"),
		),
		Trace: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "follow trace"),
		),
		TraceClear: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "clear trace"),
		),
		Pattern: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "toggle patterns"),
		),
		BookmarkToggle: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "toggle bookmark"),
		),
		BookmarkNext: key.NewBinding(
			key.WithKeys("'"),
			key.WithHelp("'", "next bookmark"),
		),
		BookmarkFilter: key.NewBinding(
			key.WithKeys("B"),
			key.WithHelp("B", "filter bookmarks"),
		),
	}
}
