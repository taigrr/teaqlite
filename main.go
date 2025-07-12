package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/mattn/go-sqlite3"
)

const (
	pageSize = 20
)

type viewMode int

const (
	modeTableList viewMode = iota
	modeTableData
	modeQuery
	modeRowDetail
	modeEditCell
)

type model struct {
	db               *sql.DB
	mode             viewMode
	tables           []string
	filteredTables   []string
	selectedTable    int
	tableListPage    int
	tableData        [][]string
	filteredData     [][]string
	columns          []string
	primaryKeys      []string
	currentPage      int
	totalRows        int
	selectedRow      int
	selectedCol      int
	query            string
	queryInput       string
	searchInput      string
	dataSearchInput  string
	searching        bool
	dataSearching    bool
	editingValue     string
	originalValue    string
	cursor           int
	err              error
	width            int
	height           int
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#F25D94"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	tableStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2)
)

func initialModel(dbPath string) model {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return model{err: err}
	}

	m := model{
		db:              db,
		mode:            modeTableList,
		currentPage:     0,
		tableListPage:   0,
		filteredTables:  []string{},
		filteredData:    [][]string{},
		searchInput:     "",
		dataSearchInput: "",
		searching:       false,
		dataSearching:   false,
		selectedRow:     0,
		selectedCol:     0,
		width:           80, // default width
		height:          24, // default height
	}

	m.loadTables()
	return m
}

func (m *model) loadTables() {
	rows, err := m.db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		m.err = err
		return
	}
	defer rows.Close()

	m.tables = []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			m.err = err
			return
		}
		m.tables = append(m.tables, name)
	}
	m.filterTables()
}

func (m *model) filterTables() {
	if m.searchInput == "" {
		m.filteredTables = make([]string, len(m.tables))
		copy(m.filteredTables, m.tables)
	} else {
		m.filteredTables = []string{}
		searchLower := strings.ToLower(m.searchInput)
		for _, table := range m.tables {
			if strings.Contains(strings.ToLower(table), searchLower) {
				m.filteredTables = append(m.filteredTables, table)
			}
		}
	}
	
	// Reset selection and page if needed
	if m.selectedTable >= len(m.filteredTables) {
		m.selectedTable = 0
		m.tableListPage = 0
	}
}

func (m *model) getVisibleTableCount() int {
	// Reserve space for title (3 lines), help (3 lines), search bar (2 lines if searching)
	reservedLines := 8
	if m.searching {
		reservedLines += 2
	}
	return max(1, m.height-reservedLines)
}

func (m *model) getVisibleDataRowCount() int {
	// Reserve space for title (3 lines), header (3 lines), help (3 lines)
	reservedLines := 9
	return max(1, m.height-reservedLines)
}

func (m *model) loadTableData() {
	if m.selectedTable >= len(m.filteredTables) {
		return
	}

	tableName := m.filteredTables[m.selectedTable]
	
	// Get column info and primary keys
	rows, err := m.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		m.err = err
		return
	}
	defer rows.Close()

	m.columns = []string{}
	m.primaryKeys = []string{}
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString
		
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			m.err = err
			return
		}
		m.columns = append(m.columns, name)
		if pk == 1 {
			m.primaryKeys = append(m.primaryKeys, name)
		}
	}

	// Get total row count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	err = m.db.QueryRow(countQuery).Scan(&m.totalRows)
	if err != nil {
		m.err = err
		return
	}

	// Get paginated data
	offset := m.currentPage * pageSize
	dataQuery := fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", tableName, pageSize, offset)
	
	rows, err = m.db.Query(dataQuery)
	if err != nil {
		m.err = err
		return
	}
	defer rows.Close()

	m.tableData = [][]string{}
	for rows.Next() {
		values := make([]any, len(m.columns))
		valuePtrs := make([]any, len(m.columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			m.err = err
			return
		}

		row := make([]string, len(m.columns))
		for i, val := range values {
			if val == nil {
				row[i] = "NULL"
			} else {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		m.tableData = append(m.tableData, row)
	}
	
	// Apply data filtering
	m.filterData()
	
	// Reset row selection if needed
	if m.selectedRow >= len(m.filteredData) {
		m.selectedRow = 0
	}
}

func (m *model) filterData() {
	if m.dataSearchInput == "" {
		m.filteredData = make([][]string, len(m.tableData))
		copy(m.filteredData, m.tableData)
	} else {
		m.filteredData = [][]string{}
		searchLower := strings.ToLower(m.dataSearchInput)
		for _, row := range m.tableData {
			// Search in all columns of the row
			found := false
			for _, cell := range row {
				if strings.Contains(strings.ToLower(cell), searchLower) {
					found = true
					break
				}
			}
			if found {
				m.filteredData = append(m.filteredData, row)
			}
		}
	}
}

func (m *model) updateCell(rowIndex, colIndex int, newValue string) error {
	if rowIndex >= len(m.filteredData) || colIndex >= len(m.columns) {
		return fmt.Errorf("invalid row or column index")
	}
	
	tableName := m.filteredTables[m.selectedTable]
	columnName := m.columns[colIndex]
	
	// Build WHERE clause using primary keys or all columns if no primary key
	var whereClause strings.Builder
	var args []any
	
	if len(m.primaryKeys) > 0 {
		// Use primary keys for WHERE clause
		for i, pkCol := range m.primaryKeys {
			if i > 0 {
				whereClause.WriteString(" AND ")
			}
			// Find the column index for this primary key
			pkIndex := -1
			for j, col := range m.columns {
				if col == pkCol {
					pkIndex = j
					break
				}
			}
			if pkIndex >= 0 {
				whereClause.WriteString(fmt.Sprintf("%s = ?", pkCol))
				args = append(args, m.filteredData[rowIndex][pkIndex])
			}
		}
	} else {
		// Use all columns for WHERE clause (less reliable but works)
		for i, col := range m.columns {
			if i > 0 {
				whereClause.WriteString(" AND ")
			}
			whereClause.WriteString(fmt.Sprintf("%s = ?", col))
			args = append(args, m.filteredData[rowIndex][i])
		}
	}
	
	// Execute UPDATE
	updateQuery := fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s", tableName, columnName, whereClause.String())
	args = append([]any{newValue}, args...)
	
	_, err := m.db.Exec(updateQuery, args...)
	if err != nil {
		return err
	}
	
	// Update local data
	m.filteredData[rowIndex][colIndex] = newValue
	// Also update the original data if it exists
	for i, row := range m.tableData {
		if len(row) > colIndex {
			// Simple comparison - this might not work perfectly for all cases
			match := true
			for j, cell := range row {
				if j < len(m.filteredData[rowIndex]) && cell != m.filteredData[rowIndex][j] && j != colIndex {
					match = false
					break
				}
			}
			if match {
				m.tableData[i][colIndex] = newValue
				break
			}
		}
	}
	
	return nil
}

func (m *model) executeQuery() {
	if strings.TrimSpace(m.query) == "" {
		return
	}

	rows, err := m.db.Query(m.query)
	if err != nil {
		m.err = err
		return
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		m.err = err
		return
	}
	m.columns = columns

	// Get data
	m.tableData = [][]string{}
	for rows.Next() {
		values := make([]any, len(m.columns))
		valuePtrs := make([]any, len(m.columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			m.err = err
			return
		}

		row := make([]string, len(m.columns))
		for i, val := range values {
			if val == nil {
				row[i] = "NULL"
			} else {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		m.tableData = append(m.tableData, row)
	}

	m.totalRows = len(m.tableData)
	m.currentPage = 0
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		// Handle edit mode first
		if m.mode == modeEditCell {
			switch msg.String() {
			case "esc":
				m.mode = modeRowDetail
				m.editingValue = ""
			case "enter":
				if err := m.updateCell(m.selectedRow, m.selectedCol, m.editingValue); err != nil {
					m.err = err
				} else {
					m.mode = modeRowDetail
					m.editingValue = ""
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

		// Handle search modes
		if m.searching {
			switch msg.String() {
			case "esc":
				m.searching = false
				m.searchInput = ""
				m.filterTables()
			case "enter":
				m.searching = false
				m.filterTables()
			case "backspace":
				if len(m.searchInput) > 0 {
					m.searchInput = m.searchInput[:len(m.searchInput)-1]
					m.filterTables()
				}
			default:
				if len(msg.String()) == 1 {
					m.searchInput += msg.String()
					m.filterTables()
				}
			}
			return m, nil
		}

		if m.dataSearching {
			switch msg.String() {
			case "esc":
				m.dataSearching = false
				m.dataSearchInput = ""
				m.filterData()
			case "enter":
				m.dataSearching = false
				m.filterData()
			case "backspace":
				if len(m.dataSearchInput) > 0 {
					m.dataSearchInput = m.dataSearchInput[:len(m.dataSearchInput)-1]
					m.filterData()
				}
			default:
				if len(msg.String()) == 1 {
					m.dataSearchInput += msg.String()
					m.filterData()
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			switch m.mode {
			case modeTableData, modeQuery:
				m.mode = modeTableList
				m.err = nil
			case modeRowDetail:
				m.mode = modeTableData
			}

		case "/":
			switch m.mode {
			case modeTableList:
				m.searching = true
				m.searchInput = ""
			case modeTableData:
				m.dataSearching = true
				m.dataSearchInput = ""
			}

		case "enter":
			switch m.mode {
			case modeTableList:
				if len(m.filteredTables) > 0 {
					m.mode = modeTableData
					m.currentPage = 0
					m.selectedRow = 0
					m.loadTableData()
				}
			case modeTableData:
				if len(m.filteredData) > 0 {
					m.mode = modeRowDetail
					m.selectedCol = 0
				}
			case modeRowDetail:
				if len(m.filteredData) > 0 && m.selectedRow < len(m.filteredData) && m.selectedCol < len(m.columns) {
					m.mode = modeEditCell
					m.originalValue = m.filteredData[m.selectedRow][m.selectedCol]
					m.editingValue = m.originalValue
				}
			case modeQuery:
				m.executeQuery()
			}

		case "up", "k":
			switch m.mode {
			case modeTableList:
				if m.selectedTable > 0 {
					m.selectedTable--
					// Check if we need to scroll up
					visibleCount := m.getVisibleTableCount()
					if m.selectedTable < m.tableListPage*visibleCount {
						m.tableListPage--
					}
				}
			case modeTableData:
				if m.selectedRow > 0 {
					m.selectedRow--
				}
			case modeRowDetail:
				if m.selectedCol > 0 {
					m.selectedCol--
				}
			case modeQuery:
				// In query mode, these should be treated as input
				if len(msg.String()) == 1 {
					m.queryInput += msg.String()
					m.query = m.queryInput
				}
			}

		case "down", "j":
			switch m.mode {
			case modeTableList:
				if m.selectedTable < len(m.filteredTables)-1 {
					m.selectedTable++
					// Check if we need to scroll down
					visibleCount := m.getVisibleTableCount()
					if m.selectedTable >= (m.tableListPage+1)*visibleCount {
						m.tableListPage++
					}
				}
			case modeTableData:
				if m.selectedRow < len(m.filteredData)-1 {
					m.selectedRow++
				}
			case modeRowDetail:
				if m.selectedCol < len(m.columns)-1 {
					m.selectedCol++
				}
			case modeQuery:
				// In query mode, these should be treated as input
				if len(msg.String()) == 1 {
					m.queryInput += msg.String()
					m.query = m.queryInput
				}
			}

		case "left", "h":
			switch m.mode {
			case modeTableData:
				if m.currentPage > 0 {
					m.currentPage--
					m.selectedRow = 0
					m.loadTableData()
				}
			case modeTableList:
				if m.tableListPage > 0 {
					m.tableListPage--
					// Adjust selection to stay in view
					visibleCount := m.getVisibleTableCount()
					m.selectedTable = m.tableListPage * visibleCount
				}
			case modeQuery:
				// In query mode, these should be treated as input
				if len(msg.String()) == 1 {
					m.queryInput += msg.String()
					m.query = m.queryInput
				}
			}

		case "right", "l":
			switch m.mode {
			case modeTableData:
				maxPage := max(0, (m.totalRows-1)/pageSize)
				if m.currentPage < maxPage {
					m.currentPage++
					m.selectedRow = 0
					m.loadTableData()
				}
			case modeTableList:
				visibleCount := m.getVisibleTableCount()
				maxPage := (len(m.filteredTables) - 1) / visibleCount
				if m.tableListPage < maxPage {
					m.tableListPage++
					// Adjust selection to stay in view
					m.selectedTable = m.tableListPage * visibleCount
					if m.selectedTable >= len(m.filteredTables) {
						m.selectedTable = len(m.filteredTables) - 1
					}
				}
			case modeQuery:
				// In query mode, these should be treated as input
				if len(msg.String()) == 1 {
					m.queryInput += msg.String()
					m.query = m.queryInput
				}
			}

		case "s":
			if m.mode == modeTableList {
				m.mode = modeQuery
				m.queryInput = ""
				m.query = ""
				m.cursor = 0
			} else if m.mode == modeQuery {
				// In query mode, 's' should be treated as input
				m.queryInput += msg.String()
				m.query = m.queryInput
			}

		case "r":
			switch m.mode {
			case modeTableList:
				m.loadTables()
			case modeTableData:
				m.loadTableData()
			case modeRowDetail:
				m.loadTableData()
			case modeQuery:
				// In query mode, 'r' should be treated as input
				m.queryInput += msg.String()
				m.query = m.queryInput
			}

		case "backspace":
			if m.mode == modeQuery && len(m.queryInput) > 0 {
				m.queryInput = m.queryInput[:len(m.queryInput)-1]
				m.query = m.queryInput
			}

		default:
			if m.mode == modeQuery {
				if len(msg.String()) == 1 {
					m.queryInput += msg.String()
					m.query = m.queryInput
				}
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress 'esc' to continue or 'q' to quit", m.err))
	}

	var content strings.Builder

	switch m.mode {
	case modeTableList:
		// Title
		content.WriteString(titleStyle.Render("SQLite TUI - Tables"))
		content.WriteString("\n")
		
		// Search bar
		if m.searching {
			content.WriteString("\nSearch: " + m.searchInput + "_")
			content.WriteString("\n")
		} else if m.searchInput != "" {
			content.WriteString(fmt.Sprintf("\nFiltered by: %s (%d/%d tables)", m.searchInput, len(m.filteredTables), len(m.tables)))
			content.WriteString("\n")
		}
		content.WriteString("\n")

		// Table list with pagination
		if len(m.filteredTables) == 0 {
			if m.searchInput != "" {
				content.WriteString("No tables match your search")
			} else {
				content.WriteString("No tables found in database")
			}
		} else {
			visibleCount := m.getVisibleTableCount()
			startIdx := m.tableListPage * visibleCount
			endIdx := min(startIdx+visibleCount, len(m.filteredTables))
			
			for i := startIdx; i < endIdx; i++ {
				table := m.filteredTables[i]
				if i == m.selectedTable {
					content.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", table)))
				} else {
					content.WriteString(normalStyle.Render(fmt.Sprintf("  %s", table)))
				}
				content.WriteString("\n")
			}
			
			// Show pagination info
			if len(m.filteredTables) > visibleCount {
				totalPages := (len(m.filteredTables) - 1) / visibleCount + 1
				content.WriteString(fmt.Sprintf("\nPage %d/%d", m.tableListPage+1, totalPages))
			}
		}

		// Help
		content.WriteString("\n")
		if m.searching {
			content.WriteString(helpStyle.Render("Type to search • enter/esc: finish search"))
		} else {
			content.WriteString(helpStyle.Render("↑/↓: navigate • ←/→: page • /: search • enter: view • s: SQL • r: refresh • q: quit"))
		}

	case modeTableData:
		tableName := m.filteredTables[m.selectedTable]
		maxPage := max(0, (m.totalRows-1)/pageSize)
		
		content.WriteString(titleStyle.Render(fmt.Sprintf("Table: %s (Page %d/%d)", tableName, m.currentPage+1, maxPage+1)))
		content.WriteString("\n")
		
		// Search bar for data
		if m.dataSearching {
			content.WriteString("\nSearch data: " + m.dataSearchInput + "_")
			content.WriteString("\n")
		} else if m.dataSearchInput != "" {
			content.WriteString(fmt.Sprintf("\nFiltered by: %s (%d/%d rows)", m.dataSearchInput, len(m.filteredData), len(m.tableData)))
			content.WriteString("\n")
		}
		content.WriteString("\n")

		if len(m.filteredData) == 0 {
			if m.dataSearchInput != "" {
				content.WriteString("No rows match your search")
			} else {
				content.WriteString("No data in table")
			}
		} else {
			// Limit rows to fit screen
			visibleRows := m.getVisibleDataRowCount()
			displayRows := min(len(m.filteredData), visibleRows)
			
			// Create table header
			var headerRow strings.Builder
			colWidth := 10 // default minimum width
			if len(m.columns) > 0 && m.width > 0 {
				colWidth = max(10, (m.width-len(m.columns)*3)/len(m.columns))
			}
			for i, col := range m.columns {
				if i > 0 {
					headerRow.WriteString(" | ")
				}
				headerRow.WriteString(fmt.Sprintf("%-*s", colWidth, truncateString(col, colWidth)))
			}
			content.WriteString(selectedStyle.Render(headerRow.String()))
			content.WriteString("\n")

			// Add separator
			var separator strings.Builder
			for i := range m.columns {
				if i > 0 {
					separator.WriteString("-+-")
				}
				separator.WriteString(strings.Repeat("-", colWidth))
			}
			content.WriteString(separator.String())
			content.WriteString("\n")

			// Add data rows with highlighting
			for i := 0; i < displayRows; i++ {
				if i >= len(m.filteredData) {
					break
				}
				row := m.filteredData[i]
				var dataRow strings.Builder
				for j, cell := range row {
					if j > 0 {
						dataRow.WriteString(" | ")
					}
					dataRow.WriteString(fmt.Sprintf("%-*s", colWidth, truncateString(cell, colWidth)))
				}
				if i == m.selectedRow {
					content.WriteString(selectedStyle.Render(dataRow.String()))
				} else {
					content.WriteString(normalStyle.Render(dataRow.String()))
				}
				content.WriteString("\n")
			}
		}

		content.WriteString("\n")
		if m.dataSearching {
			content.WriteString(helpStyle.Render("Type to search • enter/esc: finish search"))
		} else {
			content.WriteString(helpStyle.Render("↑/↓: select row • ←/→: page • /: search • enter: view row • esc: back • r: refresh • q: quit"))
		}

	case modeRowDetail:
		tableName := m.filteredTables[m.selectedTable]
		content.WriteString(titleStyle.Render(fmt.Sprintf("Row Detail: %s", tableName)))
		content.WriteString("\n\n")

		if m.selectedRow >= len(m.filteredData) {
			content.WriteString("Invalid row selection")
		} else {
			row := m.filteredData[m.selectedRow]
			
			// Show as 2-column table: Column | Value
			colWidth := max(15, m.width/3)
			valueWidth := max(20, m.width-colWidth-5)
			
			// Header
			headerRow := fmt.Sprintf("%-*s | %-*s", colWidth, "Column", valueWidth, "Value")
			content.WriteString(selectedStyle.Render(headerRow))
			content.WriteString("\n")
			
			// Separator
			separator := strings.Repeat("-", colWidth) + "-+-" + strings.Repeat("-", valueWidth)
			content.WriteString(separator)
			content.WriteString("\n")
			
			// Data rows
			visibleRows := m.getVisibleDataRowCount() - 4 // Account for header and title
			displayRows := min(len(m.columns), visibleRows)
			
			for i := 0; i < displayRows; i++ {
				if i >= len(m.columns) || i >= len(row) {
					break
				}
				
				col := m.columns[i]
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

	case modeEditCell:
		tableName := m.filteredTables[m.selectedTable]
		columnName := ""
		if m.selectedCol < len(m.columns) {
			columnName = m.columns[m.selectedCol]
		}
		
		content.WriteString(titleStyle.Render(fmt.Sprintf("Edit: %s.%s", tableName, columnName)))
		content.WriteString("\n\n")
		
		// Calculate available width for text (leave some margin)
		textWidth := max(20, m.width-4)
		
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

	case modeQuery:
		content.WriteString(titleStyle.Render("SQL Query"))
		content.WriteString("\n\n")
		
		content.WriteString("Query: ")
		content.WriteString(m.queryInput)
		content.WriteString("_") // cursor
		content.WriteString("\n\n")

		if len(m.tableData) > 0 {
			// Limit rows to fit screen
			visibleRows := m.getVisibleDataRowCount() - 2 // Account for query input
			displayRows := min(len(m.tableData), visibleRows)
			
			// Show query results
			colWidth := 10 // default minimum width
			if len(m.columns) > 0 && m.width > 0 {
				colWidth = max(10, (m.width-len(m.columns)*3)/len(m.columns))
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
				row := m.tableData[i]
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
			
			if len(m.tableData) > displayRows {
				content.WriteString(helpStyle.Render(fmt.Sprintf("... and %d more rows", len(m.tableData)-displayRows)))
				content.WriteString("\n")
			}
		}

		content.WriteString("\n")
		content.WriteString(helpStyle.Render("enter: execute query • esc: back • q: quit"))
	}

	// Ensure content fits in screen height
	lines := strings.Split(content.String(), "\n")
	maxLines := max(1, m.height-2)
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		content.Reset()
		content.WriteString(strings.Join(lines, "\n"))
	}

	return content.String()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	
	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}
	
	currentLine := ""
	for _, word := range words {
		// If adding this word would exceed the width
		if len(currentLine)+len(word)+1 > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				// Word is longer than width, break it
				for len(word) > width {
					lines = append(lines, word[:width])
					word = word[width:]
				}
				currentLine = word
			}
		} else {
			if currentLine != "" {
				currentLine += " " + word
			} else {
				currentLine = word
			}
		}
	}
	
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	
	return lines
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go-sqlite-tui <database.db>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Printf("Database file '%s' does not exist\n", dbPath)
		os.Exit(1)
	}

	m := initialModel(dbPath)
	if m.err != nil {
		log.Fatal(m.err)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}