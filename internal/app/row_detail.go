package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Row Detail Model
type RowDetailModel struct {
	Shared    *SharedData
	rowIndex  int
	FromQuery bool
}

func NewRowDetailModel(shared *SharedData, rowIndex int) *RowDetailModel {
	return &RowDetailModel{
		Shared:   shared,
		rowIndex: rowIndex,
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
			if m.FromQuery {
				return m, func() tea.Msg { return ReturnToQueryMsg{} }
			}
			return m, func() tea.Msg { return SwitchToTableDataMsg{TableIndex: m.Shared.SelectedTable} }

		case "e":
			if len(m.Shared.FilteredData) > m.rowIndex && len(m.Shared.Columns) > 0 {
				return m, func() tea.Msg {
					return SwitchToEditCellMsg{RowIndex: m.rowIndex, ColIndex: 0}
				}
			}
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

	row := m.Shared.FilteredData[m.rowIndex]
	for i, col := range m.Shared.Columns {
		if i < len(row) {
			content.WriteString(fmt.Sprintf("%s: %s\n", col, row[i]))
		}
	}

	content.WriteString("\n")
	content.WriteString(HelpStyle.Render("e: edit • q: back"))

	return content.String()
}