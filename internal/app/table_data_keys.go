package app

import "github.com/charmbracelet/bubbles/key"

// TableDataKeyMap defines keybindings for the table data view.
// Navigation follows vim-like patterns:
// - gg: go to start (requires two 'g' presses)
// - G: go to end (single 'G' press)
type TableDataKeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Enter      key.Binding
	Search     key.Binding
	Escape     key.Binding
	Back       key.Binding
	GoToStart  key.Binding
	GoToEnd    key.Binding
	Refresh    key.Binding
	SQLMode    key.Binding
	ToggleHelp key.Binding
}

// DefaultTableDataKeyMap returns the default keybindings for table data
func DefaultTableDataKeyMap() TableDataKeyMap {
	return TableDataKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "prev page"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next page"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "view details"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/clear"),
		),
		Back: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "back to tables"),
		),
		GoToStart: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "go to start"),
		),
		GoToEnd: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to end"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		SQLMode: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "SQL mode"),
		),
		ToggleHelp: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "toggle help"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k TableDataKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.GoToStart, k.GoToEnd, k.Search, k.ToggleHelp}
}

// FullHelp returns keybindings for the expanded help view
func (k TableDataKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Search, k.Escape, k.Back},
		{k.GoToStart, k.GoToEnd, k.Refresh, k.SQLMode, k.ToggleHelp},
	}
}