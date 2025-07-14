package app

import "github.com/charmbracelet/bubbles/key"

// EditCellKeyMap defines keybindings for the edit cell view
type EditCellKeyMap struct {
	Save          key.Binding
	Cancel        key.Binding
	CursorLeft    key.Binding
	CursorRight   key.Binding
	WordLeft      key.Binding
	WordRight     key.Binding
	LineStart     key.Binding
	LineEnd       key.Binding
	DeleteWord    key.Binding
	DeleteChar    key.Binding
	ToggleHelp    key.Binding
}

// DefaultEditCellKeyMap returns the default keybindings for edit cell
func DefaultEditCellKeyMap() EditCellKeyMap {
	return EditCellKeyMap{
		Save: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "save"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
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
		DeleteChar: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("backspace", "delete char"),
		),
		ToggleHelp: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "toggle help"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k EditCellKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Save, k.Cancel, k.ToggleHelp}
}

// FullHelp returns keybindings for the expanded help view
func (k EditCellKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Save, k.Cancel},
		{k.CursorLeft, k.CursorRight, k.WordLeft, k.WordRight},
		{k.LineStart, k.LineEnd, k.DeleteWord, k.DeleteChar, k.ToggleHelp},
	}
}