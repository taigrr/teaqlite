package app

import "github.com/charmbracelet/bubbles/key"

// RowDetailKeyMap defines keybindings for the row detail view.
// Navigation follows vim-like patterns:
// - gg: go to start (requires two 'g' presses)
// - G: go to end (single 'G' press)
type RowDetailKeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Escape     key.Binding
	Back       key.Binding
	GoToStart  key.Binding
	GoToEnd    key.Binding
	ToggleHelp key.Binding
}

// DefaultRowDetailKeyMap returns the default keybindings for row detail
func DefaultRowDetailKeyMap() RowDetailKeyMap {
	return RowDetailKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "edit cell"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Back: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "back"),
		),
		GoToStart: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "go to start"),
		),
		GoToEnd: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to end"),
		),
		ToggleHelp: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "toggle help"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k RowDetailKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.GoToStart, k.GoToEnd, k.Back, k.ToggleHelp}
}

// FullHelp returns keybindings for the expanded help view
func (k RowDetailKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Escape, k.Back, k.GoToStart, k.GoToEnd, k.ToggleHelp},
	}
}