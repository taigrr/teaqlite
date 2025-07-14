package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type QueryModel struct {
	Shared       *SharedData
	query        string
	cursor       int
	FocusOnInput bool
	selectedRow  int
	results      [][]string
	columns      []string
	err          error
}

func NewQueryModel(shared *SharedData) *QueryModel {
	return &QueryModel{
		Shared:       shared,
		FocusOnInput: true,
		selectedRow:  0,
	}
}

func (m *QueryModel) Init() tea.Cmd {
	return nil
}

func (m *QueryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.FocusOnInput {
			return m.handleQueryInput(msg)
		}
		return m.handleResultsNavigation(msg)
	}
	return m, nil
}

func (m *QueryModel) handleQueryInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return SwitchToTableListMsg{} }

	case "enter":
		if strings.TrimSpace(m.query) != "" {
			return m, m.executeQuery()
		}

	case "backspace":
		if m.cursor > 0 {
			m.query = m.query[:m.cursor-1] + m.query[m.cursor:]
			m.cursor--
		}

	case "left":
		if m.cursor > 0 {
			m.cursor--
		}

	case "right":
		if m.cursor < len(m.query) {
			m.cursor++
		}

	default:
		if len(msg.String()) == 1 {
			m.query = m.query[:m.cursor] + msg.String() + m.query[m.cursor:]
			m.cursor++
		}
	}
	return m, nil
}

func (m *QueryModel) handleResultsNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return m, func() tea.Msg { return SwitchToTableListMsg{} }

	case "i":
		m.FocusOnInput = true
		return m, nil

	case "enter":
		if len(m.results) > 0 {
			return m, func() tea.Msg {
				return SwitchToRowDetailFromQueryMsg{RowIndex: m.selectedRow}
			}
		}

	case "up", "k":
		if m.selectedRow > 0 {
			m.selectedRow--
		}

	case "down", "j":
		if m.selectedRow < len(m.results)-1 {
			m.selectedRow++
		}
	}
	return m, nil
}

func (m *QueryModel) ensureIDColumns(query string) string {
	// Convert to lowercase for easier parsing
	lowerQuery := strings.ToLower(strings.TrimSpace(query))

	// Only modify SELECT statements
	if !strings.HasPrefix(lowerQuery, "select") {
		return query
	}

	// Extract table name from FROM clause
	tableName := m.extractTableName(query)
	if tableName == "" {
		return query // Can't determine table, return original query
	}

	// Get primary key columns for this table
	primaryKeys := m.getTablePrimaryKeys(tableName)
	if len(primaryKeys) == 0 {
		return query // No primary keys found
	}

	// Check if any primary key columns are already in the query
	for _, pk := range primaryKeys {
		if strings.Contains(lowerQuery, strings.ToLower(pk)) {
			return query // Primary key already included
		}
	}

	// Check if it's a SELECT * query
	if strings.Contains(lowerQuery, "select *") {
		return query // SELECT * already includes all columns
	}

	// Add primary key columns to the SELECT clause
	selectIndex := strings.Index(lowerQuery, "select")
	fromIndex := strings.Index(lowerQuery, "from")

	if selectIndex == -1 || fromIndex == -1 || fromIndex <= selectIndex {
		return query // Malformed query
	}

	// Extract the column list
	selectClause := strings.TrimSpace(query[selectIndex+6 : fromIndex])

	// Add primary keys to the beginning
	var pkList []string
	for _, pk := range primaryKeys {
		pkList = append(pkList, pk)
	}

	newSelectClause := strings.Join(pkList, ", ") + ", " + selectClause

	// Reconstruct the query
	return "SELECT " + newSelectClause + " " + query[fromIndex:]
}

func (m *QueryModel) extractTableName(query string) string {
	lowerQuery := strings.ToLower(query)

	// Find FROM keyword
	fromIndex := strings.Index(lowerQuery, "from")
	if fromIndex == -1 {
		return ""
	}

	// Extract everything after FROM
	afterFrom := strings.TrimSpace(query[fromIndex+4:])

	// Split by whitespace and take the first word (table name)
	parts := strings.Fields(afterFrom)
	if len(parts) == 0 {
		return ""
	}

	// Remove any alias or additional clauses
	tableName := parts[0]

	// Remove quotes if present
	tableName = strings.Trim(tableName, "\"'`")

	return tableName
}

func (m *QueryModel) getTablePrimaryKeys(tableName string) []string {
	rows, err := m.Shared.DB.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil
	}
	defer rows.Close()

	var primaryKeys []string
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue any

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			continue
		}

		if pk == 1 {
			primaryKeys = append(primaryKeys, name)
		}
	}

	return primaryKeys
}

func (m *QueryModel) executeQuery() tea.Cmd {
	return func() tea.Msg {
		// Modify query to always include ID columns if it's a SELECT statement
		modifiedQuery := m.ensureIDColumns(m.query)

		rows, err := m.Shared.DB.Query(modifiedQuery)
		if err != nil {
			return QueryCompletedMsg{Error: err}
		}
		defer rows.Close()

		// Get column names
		columns, err := rows.Columns()
		if err != nil {
			return QueryCompletedMsg{Error: err}
		}

		// Get results
		var results [][]string
		for rows.Next() {
			values := make([]any, len(columns))
			valuePtrs := make([]any, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return QueryCompletedMsg{Error: err}
			}

			row := make([]string, len(columns))
			for i, val := range values {
				if val == nil {
					row[i] = "NULL"
				} else {
					row[i] = fmt.Sprintf("%v", val)
				}
			}
			results = append(results, row)
		}

		return QueryCompletedMsg{
			Results: results,
			Columns: columns,
			Error:   nil,
		}
	}
}

func (m *QueryModel) handleQueryCompletion(msg QueryCompletedMsg) {
	if msg.Error != nil {
		m.err = msg.Error
		return
	}

	m.results = msg.Results
	m.columns = msg.Columns

	// Update shared data for row detail view
	m.Shared.FilteredData = m.results
	m.Shared.Columns = m.columns
	m.Shared.IsQueryResult = true

	m.FocusOnInput = false
	m.selectedRow = 0
	m.err = nil
}

func (m *QueryModel) View() string {
	var content strings.Builder

	content.WriteString(TitleStyle.Render("SQL Query"))
	content.WriteString("\n\n")

	// Query input
	content.WriteString("Query: ")
	if m.FocusOnInput {
		content.WriteString(m.query + "_")
	} else {
		content.WriteString(m.query)
	}
	content.WriteString("\n\n")

	// Error display
	if m.err != nil {
		content.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		content.WriteString("\n\n")
	}

	// Results
	if len(m.results) > 0 {
		// Column headers
		headerRow := ""
		for i, col := range m.columns {
			if i > 0 {
				headerRow += " | "
			}
			headerRow += TruncateString(col, 15)
		}
		content.WriteString(TitleStyle.Render(headerRow))
		content.WriteString("\n")

		// Data rows with scrolling
		visibleCount := Max(1, m.Shared.Height-10)
		startIdx := 0

		// Adjust start index if selected row is out of view
		if m.selectedRow >= visibleCount {
			startIdx = m.selectedRow - visibleCount + 1
		}

		endIdx := Min(len(m.results), startIdx+visibleCount)

		for i := range endIdx {
			if i < startIdx {
				continue
			}
			row := m.results[i]
			rowStr := ""
			for j, cell := range row {
				if j > 0 {
					rowStr += " | "
				}
				rowStr += TruncateString(cell, 15)
			}

			if i == m.selectedRow && !m.FocusOnInput {
				content.WriteString(SelectedStyle.Render("> " + rowStr))
			} else {
				content.WriteString(NormalStyle.Render("  " + rowStr))
			}
			content.WriteString("\n")
		}

		content.WriteString(fmt.Sprintf("\n%d rows returned\n", len(m.results)))
	}

	content.WriteString("\n")
	if m.FocusOnInput {
		content.WriteString(HelpStyle.Render("enter: execute • esc: back"))
	} else {
		content.WriteString(HelpStyle.Render("↑/↓: navigate • enter: details • i: edit query • q: back"))
	}

	return content.String()
}
