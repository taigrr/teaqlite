package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type EditCellModel struct {
	Shared   *SharedData
	rowIndex int
	colIndex int
	value    string
	cursor   int
}

func NewEditCellModel(shared *SharedData, rowIndex, colIndex int) *EditCellModel {
	value := ""
	if rowIndex < len(shared.FilteredData) && colIndex < len(shared.FilteredData[rowIndex]) {
		value = shared.FilteredData[rowIndex][colIndex]
	}

	return &EditCellModel{
		Shared:   shared,
		rowIndex: rowIndex,
		colIndex: colIndex,
		value:    value,
		cursor:   len(value),
	}
}

func (m *EditCellModel) Init() tea.Cmd {
	return nil
}

func (m *EditCellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return SwitchToRowDetailMsg{RowIndex: m.rowIndex} }

		case "enter":
			return m, func() tea.Msg {
				return UpdateCellMsg{
					RowIndex: m.rowIndex,
					ColIndex: m.colIndex,
					Value:    m.value,
				}
			}

		case "backspace":
			if m.cursor > 0 {
				m.value = m.value[:m.cursor-1] + m.value[m.cursor:]
				m.cursor--
			}

		case "left":
			if m.cursor > 0 {
				m.cursor--
			}

		case "right":
			if m.cursor < len(m.value) {
				m.cursor++
			}

		default:
			if len(msg.String()) == 1 {
				m.value = m.value[:m.cursor] + msg.String() + m.value[m.cursor:]
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m *EditCellModel) View() string {
	columnName := ""
	if m.colIndex < len(m.Shared.Columns) {
		columnName = m.Shared.Columns[m.colIndex]
	}

	content := TitleStyle.Render(fmt.Sprintf("Edit Cell: %s", columnName)) + "\n\n"
	
	// Display value with visible cursor
	displayValue := m.value
	if m.cursor <= len(displayValue) {
		// Insert cursor character at cursor position
		displayValue = displayValue[:m.cursor] + "_" + displayValue[m.cursor:]
	}
	
	content += fmt.Sprintf("Value: %s\n\n", displayValue)
	content += HelpStyle.Render("enter: save • esc: cancel")

	return content
}
