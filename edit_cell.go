package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Edit Cell Model
type editCellModel struct {
	shared        *sharedData
	rowIndex      int
	colIndex      int
	editingValue  string
	originalValue string
}

func newEditCellModel(shared *sharedData, rowIndex, colIndex int) *editCellModel {
	originalValue := ""
	if rowIndex < len(shared.filteredData) && colIndex < len(shared.filteredData[rowIndex]) {
		originalValue = shared.filteredData[rowIndex][colIndex]
	}
	
	return &editCellModel{
		shared:        shared,
		rowIndex:      rowIndex,
		colIndex:      colIndex,
		editingValue:  originalValue,
		originalValue: originalValue,
	}
}

func (m *editCellModel) Init() tea.Cmd {
	return nil
}

func (m *editCellModel) Update(msg tea.Msg) (subModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleInput(msg)
	}
	return m, nil
}

func (m *editCellModel) handleInput(msg tea.KeyMsg) (subModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg {
			return switchToRowDetailMsg{rowIndex: m.rowIndex}
		}

	case "enter":
		return m, func() tea.Msg {
			return updateCellMsg{
				rowIndex: m.rowIndex,
				colIndex: m.colIndex,
				value:    m.editingValue,
			}
		}

	case "backspace":
		if len(m.editingValue) > 0 {
			m.editingValue = m.editingValue[:len(m.editingValue)-1]
		}

	default:
		if len(msg.String()) == 1 {
			m.editingValue += msg.String()
		}
	}
	return m, nil
}

func (m *editCellModel) View() string {
	var content strings.Builder

	tableName := m.shared.filteredTables[m.shared.selectedTable]
	columnName := ""
	if m.colIndex < len(m.shared.columns) {
		columnName = m.shared.columns[m.colIndex]
	}
	
	content.WriteString(titleStyle.Render(fmt.Sprintf("Edit: %s.%s", tableName, columnName)))
	content.WriteString("\n\n")
	
	// Calculate available width for text (leave some margin)
	textWidth := max(20, m.shared.width-4)
	
	// Wrap original value
	content.WriteString("Original:")
	content.WriteString("\n")
	originalLines := wrapText(m.originalValue, textWidth)
	for _, line := range originalLines {
		content.WriteString("  " + line)
		content.WriteString("\n")
	}
	
	content.WriteString("\n")
	
	// Wrap new value
	content.WriteString("New:")
	content.WriteString("\n")
	newLines := wrapText(m.editingValue+"_", textWidth) // Add cursor
	for _, line := range newLines {
		content.WriteString("  " + line)
		content.WriteString("\n")
	}
	
	content.WriteString("\n")
	content.WriteString(helpStyle.Render("Type new value • enter: save • esc: cancel"))

	return content.String()
}