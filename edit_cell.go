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
	cursorPos     int
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
		cursorPos:     len(originalValue), // Start cursor at end
	}
}

func (m *editCellModel) Init() tea.Cmd {
	return nil
}

func (m *editCellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleInput(msg)
	}
	return m, nil
}

func (m *editCellModel) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	// Cursor movement
	case "left":
		if m.cursorPos > 0 {
			m.cursorPos--
		}

	case "right":
		if m.cursorPos < len(m.editingValue) {
			m.cursorPos++
		}

	case "ctrl+left":
		m.cursorPos = m.wordLeft(m.cursorPos)

	case "ctrl+right":
		m.cursorPos = m.wordRight(m.cursorPos)

	case "home", "ctrl+a":
		m.cursorPos = 0

	case "end", "ctrl+e":
		m.cursorPos = len(m.editingValue)

	// Deletion
	case "backspace":
		if m.cursorPos > 0 {
			m.editingValue = m.editingValue[:m.cursorPos-1] + m.editingValue[m.cursorPos:]
			m.cursorPos--
		}

	case "delete", "ctrl+d":
		if m.cursorPos < len(m.editingValue) {
			m.editingValue = m.editingValue[:m.cursorPos] + m.editingValue[m.cursorPos+1:]
		}

	case "ctrl+w":
		// Delete word backward
		newPos := m.wordLeft(m.cursorPos)
		m.editingValue = m.editingValue[:newPos] + m.editingValue[m.cursorPos:]
		m.cursorPos = newPos

	case "ctrl+k":
		// Delete from cursor to end of line
		m.editingValue = m.editingValue[:m.cursorPos]

	case "ctrl+u":
		// Delete from beginning of line to cursor
		m.editingValue = m.editingValue[m.cursorPos:]
		m.cursorPos = 0

	default:
		// Insert character at cursor position
		if len(msg.String()) == 1 {
			m.editingValue = m.editingValue[:m.cursorPos] + msg.String() + m.editingValue[m.cursorPos:]
			m.cursorPos++
		}
	}
	return m, nil
}

// Helper functions for word navigation (same as query model)
func (m *editCellModel) wordLeft(pos int) int {
	if pos == 0 {
		return 0
	}
	
	// Skip whitespace
	for pos > 0 && isWhitespace(m.editingValue[pos-1]) {
		pos--
	}
	
	// Skip non-whitespace
	for pos > 0 && !isWhitespace(m.editingValue[pos-1]) {
		pos--
	}
	
	return pos
}

func (m *editCellModel) wordRight(pos int) int {
	length := len(m.editingValue)
	if pos >= length {
		return length
	}
	
	// Skip non-whitespace
	for pos < length && !isWhitespace(m.editingValue[pos]) {
		pos++
	}
	
	// Skip whitespace
	for pos < length && isWhitespace(m.editingValue[pos]) {
		pos++
	}
	
	return pos
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
	
	// Wrap new value with cursor
	content.WriteString("New:")
	content.WriteString("\n")
	
	// Display editing value with cursor
	valueWithCursor := ""
	if m.cursorPos <= len(m.editingValue) {
		before := m.editingValue[:m.cursorPos]
		after := m.editingValue[m.cursorPos:]
		valueWithCursor = before + "█" + after
	} else {
		valueWithCursor = m.editingValue + "█"
	}
	
	newLines := wrapText(valueWithCursor, textWidth)
	for _, line := range newLines {
		content.WriteString("  " + line)
		content.WriteString("\n")
	}
	
	content.WriteString("\n")
	content.WriteString(helpStyle.Render("←/→: move cursor • ctrl+←/→: word nav • home/end: line nav • ctrl+w/k/u: delete • enter: save • esc: cancel"))

	return content.String()
}