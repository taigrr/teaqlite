package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type RowDetailModel struct {
	Shared      *SharedData
	rowIndex    int
	selectedCol int
	FromQuery   bool
	gPressed    bool
}

func NewRowDetailModel(shared *SharedData, rowIndex int) *RowDetailModel {
	return &RowDetailModel{
		Shared:      shared,
		rowIndex:    rowIndex,
		selectedCol: 0,
	}
}

func (m *RowDetailModel) Init() tea.Cmd {
	return nil
}

func (m *RowDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.gPressed = false
			if m.FromQuery {
				return m, func() tea.Msg { return ReturnToQueryMsg{} }
			}
			return m, func() tea.Msg { return SwitchToTableDataMsg{TableIndex: m.Shared.SelectedTable} }

		case "g":
			if m.gPressed {
				// Second g - go to beginning
				m.selectedCol = 0
				m.gPressed = false
			} else {
				// First g - wait for second g
				m.gPressed = true
			}
			return m, nil

		case "G":
			// Go to end
			if len(m.Shared.Columns) > 0 {
				m.selectedCol = len(m.Shared.Columns) - 1
			}
			m.gPressed = false
			return m, nil

		case "e":
			m.gPressed = false
			if len(m.Shared.FilteredData) > m.rowIndex && len(m.Shared.Columns) > m.selectedCol {
				return m, func() tea.Msg {
					return SwitchToEditCellMsg{RowIndex: m.rowIndex, ColIndex: m.selectedCol}
				}
			}

		case "up", "k":
			m.gPressed = false
			if m.selectedCol > 0 {
				m.selectedCol--
			}

		case "down", "j":
			m.gPressed = false
			if m.selectedCol < len(m.Shared.Columns)-1 {
				m.selectedCol++
			}

		default:
			// Any other key resets the g state
			m.gPressed = false
		}
	}
	return m, nil
}

func (m *RowDetailModel) View() string {
	var content strings.Builder

	content.WriteString(TitleStyle.Render("Row Details"))
	content.WriteString("\n\n")

	if m.rowIndex >= len(m.Shared.FilteredData) {
		content.WriteString("Invalid row index")
		return content.String()
	}

	// Show current row position
	content.WriteString(fmt.Sprintf("Row %d of %d\n\n", m.rowIndex+1, len(m.Shared.FilteredData)))

	row := m.Shared.FilteredData[m.rowIndex]
	for i, col := range m.Shared.Columns {
		if i < len(row) {
			if i == m.selectedCol {
				content.WriteString(SelectedStyle.Render(fmt.Sprintf("> %s: %s", col, row[i])))
			} else {
				content.WriteString(NormalStyle.Render(fmt.Sprintf("  %s: %s", col, row[i])))
			}
			content.WriteString("\n")
		}
	}

	content.WriteString("\n")
	content.WriteString(HelpStyle.Render("↑/↓: navigate columns • e: edit • gg/G: first/last • q: back"))

	return content.String()
}
