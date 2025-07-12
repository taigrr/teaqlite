package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Query Model
type queryModel struct {
	shared     *sharedData
	queryInput string
	results    [][]string
	columns    []string
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

	case "backspace":
		if len(m.queryInput) > 0 {
			m.queryInput = m.queryInput[:len(m.queryInput)-1]
		}

	default:
		// In query mode, all single characters should be treated as input
		if len(msg.String()) == 1 {
			m.queryInput += msg.String()
		}
	}
	return m, nil
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
	
	content.WriteString("Query: ")
	content.WriteString(m.queryInput)
	content.WriteString("_") // cursor
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
	content.WriteString(helpStyle.Render("enter: execute query • esc: back • q: quit"))

	return content.String()
}