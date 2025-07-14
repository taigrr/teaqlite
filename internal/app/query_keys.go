package app

import "github.com/charmbracelet/bubbles/key"

// QueryKeyMap defines keybindings for the query view.
// Navigation follows vim-like patterns:
// - gg: go to start (requires two 'g' presses)
// - G: go to end (single 'G' press)
type QueryKeyMap struct {
	// Input mode keys
	Execute       key.Binding
	Escape        key.Binding
	CursorLeft    key.Binding
	CursorRight   key.Binding
	WordLeft      key.Binding
	WordRight     key.Binding
	LineStart     key.Binding
	LineEnd       key.Binding
	DeleteWord    key.Binding
	
	// Results mode keys
	Up            key.Binding
	Down          key.Binding
	Enter         key.Binding
	EditQuery     key.Binding
	GoToStart     key.Binding
	GoToEnd       key.Binding
	Back          key.Binding
}

// DefaultQueryKeyMap returns the default keybindings for query view
func DefaultQueryKeyMap() QueryKeyMap {
	return QueryKeyMap{
		// Input mode
		Execute: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "execute query"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		CursorLeft: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "cursor left"),
		),
		CursorRight: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "cursor right"),
		),
		WordLeft: key.NewBinding(
			key.WithKeys("ctrl+left"),
			key.WithHelp("ctrl+←", "word left"),
		),
		WordRight: key.NewBinding(
			key.WithKeys("ctrl+right"),
			key.WithHelp("ctrl+→", "word right"),
		),
		LineStart: key.NewBinding(
			key.WithKeys("home", "ctrl+a"),
			key.WithHelp("home/ctrl+a", "line start"),
		),
		LineEnd: key.NewBinding(
			key.WithKeys("end", "ctrl+e"),
			key.WithHelp("end/ctrl+e", "line end"),
		),
		DeleteWord: key.NewBinding(
			key.WithKeys("ctrl+w"),
			key.WithHelp("ctrl+w", "delete word"),
		),
		
		// Results mode
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
			key.WithHelp("enter", "view details"),
		),
		EditQuery: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "edit query"),
		),
		GoToStart: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "go to start"),
		),
		GoToEnd: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to end"),
		),
		Back: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "back"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k QueryKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Execute, k.Up, k.Down, k.GoToStart, k.GoToEnd, k.EditQuery}
}

// FullHelp returns keybindings for the expanded help view
func (k QueryKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Execute, k.Escape, k.EditQuery, k.Back},
		{k.Up, k.Down, k.Enter, k.GoToStart, k.GoToEnd},
		{k.CursorLeft, k.CursorRight, k.WordLeft, k.WordRight},
		{k.LineStart, k.LineEnd, k.DeleteWord},
	}
}