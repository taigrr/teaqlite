package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Row Detail Model
type rowDetailModel struct {
	shared      *sharedData
	rowIndex    int
	selectedCol int
}

func newRowDetailModel(shared *sharedData, rowIndex int) *rowDetailModel {
	return &rowDetailModel{
		shared:      shared,
		rowIndex:    rowIndex,
		selectedCol: 0,
	}
}

func (m *rowDetailModel) Init() tea.Cmd {
	return nil
}

func (m *rowDetailModel) Update(msg tea.Msg) (subModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleNavigation(msg)
	}
	return m, nil
}

func (m *rowDetailModel) handleNavigation(msg tea.KeyMsg) (subModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg {
			return switchToTableDataMsg{tableIndex: m.shared.selectedTable}
		}

	case "enter":
		if len(m.shared.filteredData) > 0 && m.rowIndex < len(m.shared.filteredData) && 
		   m.selectedCol < len(m.shared.columns) {
			return m, func() tea.Msg {
				return switchToEditCellMsg{rowIndex: m.rowIndex, colIndex: m.selectedCol}
			}
		}

	case "up", "k":
		if m.selectedCol > 0 {
			m.selectedCol--
		}

	case "down", "j":
		if m.selectedCol < len(m.shared.columns)-1 {
			m.selectedCol++
		}

	case "r":
		return m, func() tea.Msg { return refreshDataMsg{} }
	}
	return m, nil
}

func (m *rowDetailModel) getVisibleRowCount() int {
	reservedLines := 9
	return max(1, m.shared.height-reservedLines)
}

func (m *rowDetailModel) View() string {
	var content strings.Builder

	tableName := m.shared.filteredTables[m.shared.selectedTable]
	content.WriteString(titleStyle.Render(fmt.Sprintf("Row Detail: %s", tableName)))
	content.WriteString("\n\n")

	if m.rowIndex >= len(m.shared.filteredData) {
		content.WriteString("Invalid row selection")
	} else {
		row := m.shared.filteredData[m.rowIndex]
		
		// Show as 2-column table: Column | Value
		colWidth := max(15, m.shared.width/3)
		valueWidth := max(20, m.shared.width-colWidth-5)
		
		// Header
		headerRow := fmt.Sprintf("%-*s | %-*s", colWidth, "Column", valueWidth, "Value")
		content.WriteString(selectedStyle.Render(headerRow))
		content.WriteString("\n")
		
		// Separator
		separator := strings.Repeat("-", colWidth) + "-+-" + strings.Repeat("-", valueWidth)
		content.WriteString(separator)
		content.WriteString("\n")
		
		// Data rows
		visibleRows := m.getVisibleRowCount()
		displayRows := min(len(m.shared.columns), visibleRows)
		
		for i := 0; i < displayRows; i++ {
			if i >= len(m.shared.columns) || i >= len(row) {
				break
			}
			
			col := m.shared.columns[i]
			val := row[i]
			
			// For long values, show them wrapped on multiple lines
			if len(val) > valueWidth {
				// First line with column name
				firstLine := fmt.Sprintf("%-*s | %-*s", 
					colWidth, truncateString(col, colWidth),
					valueWidth, truncateString(val, valueWidth))
				
				if i == m.selectedCol {
					content.WriteString(selectedStyle.Render(firstLine))
				} else {
					content.WriteString(normalStyle.Render(firstLine))
				}
				content.WriteString("\n")
				
				// Additional lines for wrapped text (if there's space)
				if len(val) > valueWidth && visibleRows > displayRows {
					wrappedLines := wrapText(val, valueWidth)
					for j, wrappedLine := range wrappedLines[1:] { // Skip first line already shown
						if j >= 2 { // Limit to 3 total lines per field
							break
						}
						continuationLine := fmt.Sprintf("%-*s | %-*s", 
							colWidth, "", valueWidth, wrappedLine)
						if i == m.selectedCol {
							content.WriteString(selectedStyle.Render(continuationLine))
						} else {
							content.WriteString(normalStyle.Render(continuationLine))
						}
						content.WriteString("\n")
					}
				}
			} else {
				// Normal single line
				dataRow := fmt.Sprintf("%-*s | %-*s", 
					colWidth, truncateString(col, colWidth),
					valueWidth, val)
				
				if i == m.selectedCol {
					content.WriteString(selectedStyle.Render(dataRow))
				} else {
					content.WriteString(normalStyle.Render(dataRow))
				}
				content.WriteString("\n")
			}
		}
	}

	content.WriteString("\n")
	content.WriteString(helpStyle.Render("↑/↓: select field • enter: edit • esc: back • q: quit"))

	return content.String()
}