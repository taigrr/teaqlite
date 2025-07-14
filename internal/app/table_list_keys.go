package app

import "github.com/charmbracelet/bubbles/key"

// TableListKeyMap defines keybindings for the table list view
type TableListKeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Enter      key.Binding
	Search     key.Binding
	Escape     key.Binding
	GoToStart  key.Binding
	GoToEnd    key.Binding
	Refresh    key.Binding
	SQLMode    key.Binding
}

// DefaultTableListKeyMap returns the default keybindings for table list
func DefaultTableListKeyMap() TableListKeyMap {
	return TableListKeyMap{
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
			key.WithHelp("enter", "view table"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filter"),
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
	}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k TableListKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Search}
}

// FullHelp returns keybindings for the expanded help view
func (k TableListKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Search, k.Escape, k.Refresh},
		{k.GoToStart, k.GoToEnd, k.SQLMode},
	}
}