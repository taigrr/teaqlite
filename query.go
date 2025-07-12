package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Query Model
type queryModel struct {
	shared      *sharedData
	queryInput  string
	cursorPos   int
	results     [][]string
	columns     []string
}

func newQueryModel(shared *sharedData) *queryModel {
	return &queryModel{
		shared: shared,
	}
}

func (m *queryModel) Init() tea.Cmd {
	return nil
}

func (m *queryModel) Update(msg tea.Msg) (subModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleInput(msg)
	}
	return m, nil
}

func (m *queryModel) handleInput(msg tea.KeyMsg) (subModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return switchToTableListMsg{} }

	case "enter":
		if err := m.executeQuery(); err != nil {
			// Handle error - could set an error field
		}

	// Cursor movement
	case "left":
		if m.cursorPos > 0 {
			m.cursorPos--
		}

	case "right":
		if m.cursorPos < len(m.queryInput) {
			m.cursorPos++
		}

	case "ctrl+left":
		m.cursorPos = m.wordLeft(m.cursorPos)

	case "ctrl+right":
		m.cursorPos = m.wordRight(m.cursorPos)

	case "home", "ctrl+a":
		m.cursorPos = 0

	case "end", "ctrl+e":
		m.cursorPos = len(m.queryInput)

	// Deletion
	case "backspace":
		if m.cursorPos > 0 {
			m.queryInput = m.queryInput[:m.cursorPos-1] + m.queryInput[m.cursorPos:]
			m.cursorPos--
		}

	case "delete", "ctrl+d":
		if m.cursorPos < len(m.queryInput) {
			m.queryInput = m.queryInput[:m.cursorPos] + m.queryInput[m.cursorPos+1:]
		}

	case "ctrl+w":
		// Delete word backward
		newPos := m.wordLeft(m.cursorPos)
		m.queryInput = m.queryInput[:newPos] + m.queryInput[m.cursorPos:]
		m.cursorPos = newPos

	case "ctrl+k":
		// Delete from cursor to end of line
		m.queryInput = m.queryInput[:m.cursorPos]

	case "ctrl+u":
		// Delete from beginning of line to cursor
		m.queryInput = m.queryInput[m.cursorPos:]
		m.cursorPos = 0

	default:
		// Insert character at cursor position
		if len(msg.String()) == 1 {
			m.queryInput = m.queryInput[:m.cursorPos] + msg.String() + m.queryInput[m.cursorPos:]
			m.cursorPos++
		}
	}
	return m, nil
}

// Helper functions for word navigation
func (m *queryModel) wordLeft(pos int) int {
	if pos == 0 {
		return 0
	}
	
	// Skip whitespace
	for pos > 0 && isWhitespace(m.queryInput[pos-1]) {
		pos--
	}
	
	// Skip non-whitespace
	for pos > 0 && !isWhitespace(m.queryInput[pos-1]) {
		pos--
	}
	
	return pos
}

func (m *queryModel) wordRight(pos int) int {
	length := len(m.queryInput)
	if pos >= length {
		return length
	}
	
	// Skip non-whitespace
	for pos < length && !isWhitespace(m.queryInput[pos]) {
		pos++
	}
	
	// Skip whitespace
	for pos < length && isWhitespace(m.queryInput[pos]) {
		pos++
	}
	
	return pos
}

func isWhitespace(r byte) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func (m *queryModel) executeQuery() error {
	if strings.TrimSpace(m.queryInput) == "" {
		return nil
	}

	rows, err := m.shared.db.Query(m.queryInput)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	m.columns = columns

	// Get data
	m.results = [][]string{}
	for rows.Next() {
		values := make([]any, len(m.columns))
		valuePtrs := make([]any, len(m.columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		row := make([]string, len(m.columns))
		for i, val := range values {
			if val == nil {
				row[i] = "NULL"
			} else {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		m.results = append(m.results, row)
	}

	return nil
}

func (m *queryModel) getVisibleRowCount() int {
	reservedLines := 9
	return max(1, m.shared.height-reservedLines)
}

func (m *queryModel) View() string {
	var content strings.Builder

	content.WriteString(titleStyle.Render("SQL Query"))
	content.WriteString("\n\n")
	
	// Display query with cursor
	content.WriteString("Query: ")
	if m.cursorPos <= len(m.queryInput) {
		before := m.queryInput[:m.cursorPos]
		after := m.queryInput[m.cursorPos:]
		content.WriteString(before)
		content.WriteString("█") // Block cursor
		content.WriteString(after)
	} else {
		content.WriteString(m.queryInput)
		content.WriteString("█")
	}
	content.WriteString("\n\n")

	if len(m.results) > 0 {
		// Limit rows to fit screen
		visibleRows := m.getVisibleRowCount()
		displayRows := min(len(m.results), visibleRows)
		
		// Show query results
		colWidth := 10
		if len(m.columns) > 0 && m.shared.width > 0 {
			colWidth = max(10, (m.shared.width-len(m.columns)*3)/len(m.columns))
		}
		
		var headerRow strings.Builder
		for i, col := range m.columns {
			if i > 0 {
				headerRow.WriteString(" | ")
			}
			headerRow.WriteString(fmt.Sprintf("%-*s", colWidth, truncateString(col, colWidth)))
		}
		content.WriteString(selectedStyle.Render(headerRow.String()))
		content.WriteString("\n")

		var separator strings.Builder
		for i := range m.columns {
			if i > 0 {
				separator.WriteString("-+-")
			}
			separator.WriteString(strings.Repeat("-", colWidth))
		}
		content.WriteString(separator.String())
		content.WriteString("\n")

		for i := 0; i < displayRows; i++ {
			row := m.results[i]
			var dataRow strings.Builder
			for j, cell := range row {
				if j > 0 {
					dataRow.WriteString(" | ")
				}
				dataRow.WriteString(fmt.Sprintf("%-*s", colWidth, truncateString(cell, colWidth)))
			}
			content.WriteString(normalStyle.Render(dataRow.String()))
			content.WriteString("\n")
		}
		
		if len(m.results) > displayRows {
			content.WriteString(helpStyle.Render(fmt.Sprintf("... and %d more rows", len(m.results)-displayRows)))
			content.WriteString("\n")
		}
	}

	content.WriteString("\n")
	content.WriteString(helpStyle.Render("enter: execute • ←/→: move cursor • ctrl+←/→: word nav • home/end: line nav • ctrl+w/k/u: delete • esc: back"))

	return content.String()
}