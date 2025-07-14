package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

type RowDetailModel struct {
	Shared      *SharedData
	rowIndex    int
	selectedCol int
	FromQuery   bool
	gPressed    bool
	keyMap      RowDetailKeyMap
	help        help.Model
	focused     bool
	id          int
}

// RowDetailOption is a functional option for configuring RowDetailModel
type RowDetailOption func(*RowDetailModel)

// WithRowDetailKeyMap sets the key map
func WithRowDetailKeyMap(km RowDetailKeyMap) RowDetailOption {
	return func(m *RowDetailModel) {
		m.keyMap = km
	}
}

func NewRowDetailModel(shared *SharedData, rowIndex int, opts ...RowDetailOption) *RowDetailModel {
	m := &RowDetailModel{
		Shared:      shared,
		rowIndex:    rowIndex,
		selectedCol: 0,
		FromQuery:   false,
		keyMap:      DefaultRowDetailKeyMap(),
		help:        help.New(),
		focused:     true,
		id:          nextID(),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// ID returns the unique ID of the model
func (m RowDetailModel) ID() int {
	return m.id
}

// Focus sets the focus state
func (m *RowDetailModel) Focus() {
	m.focused = true
}

// Blur removes focus
func (m *RowDetailModel) Blur() {
	m.focused = false
}

// Focused returns the focus state
func (m RowDetailModel) Focused() bool {
	return m.focused
}

func (m *RowDetailModel) Init() tea.Cmd {
	return nil
}

func (m *RowDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleNavigation(msg)
	}
	return m, nil
}

func (m *RowDetailModel) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Escape), key.Matches(msg, m.keyMap.Back):
		m.gPressed = false
		if m.FromQuery {
			return m, func() tea.Msg { return ReturnToQueryMsg{} }
		}
		return m, func() tea.Msg { return SwitchToTableDataMsg{TableIndex: m.Shared.SelectedTable} }

	case key.Matches(msg, m.keyMap.GoToStart):
		if m.gPressed {
			// Second g - go to beginning
			m.selectedCol = 0
			m.gPressed = false
		} else {
			// First g - wait for second g
			m.gPressed = true
		}
		return m, nil

	case key.Matches(msg, m.keyMap.GoToEnd):
		// Go to end
		if len(m.Shared.Columns) > 0 {
			m.selectedCol = len(m.Shared.Columns) - 1
		}
		m.gPressed = false
		return m, nil

	case key.Matches(msg, m.keyMap.Enter):
		m.gPressed = false
		return m, func() tea.Msg {
			return SwitchToEditCellMsg{RowIndex: m.rowIndex, ColIndex: m.selectedCol}
		}

	case key.Matches(msg, m.keyMap.Up):
		m.gPressed = false
		if m.selectedCol > 0 {
			m.selectedCol--
		}

	case key.Matches(msg, m.keyMap.Down):
		m.gPressed = false
		if m.selectedCol < len(m.Shared.Columns)-1 {
			m.selectedCol++
		}

	default:
		// Any other key resets the g state
		m.gPressed = false
	}
	return m, nil
}

func (m *RowDetailModel) View() string {
	var content strings.Builder

	content.WriteString(TitleStyle.Render("Row Details"))
	content.WriteString("\n\n")

	if m.rowIndex >= len(m.Shared.FilteredData) {
		content.WriteString("Row not found")
		return content.String()
	}

	row := m.Shared.FilteredData[m.rowIndex]

	// Show each column and its value
	for i, col := range m.Shared.Columns {
		if i >= len(row) {
			break
		}

		value := row[i]
		if len(value) > 50 {
			// Wrap long values
			lines := WrapText(value, 50)
			value = strings.Join(lines, "\n    ")
		}

		line := fmt.Sprintf("%s: %s", col, value)
		if i == m.selectedCol {
			content.WriteString(SelectedStyle.Render("> " + line))
		} else {
			content.WriteString(NormalStyle.Render("  " + line))
		}
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(m.help.View(m.keyMap))

	return content.String()
}