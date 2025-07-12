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
)

type model struct {
	db             *sql.DB
	mode           viewMode
	tables         []string
	filteredTables []string
	selectedTable  int
	tableListPage  int
	tableData      [][]string
	columns        []string
	currentPage    int
	totalRows      int
	query          string
	queryInput     string
	searchInput    string
	searching      bool
	cursor         int
	err            error
	width          int
	height         int
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
		db:             db,
		mode:           modeTableList,
		currentPage:    0,
		tableListPage:  0,
		filteredTables: []string{},
		searchInput:    "",
		searching:      false,
		width:          80,  // default width
		height:         24,  // default height
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
	
	// Get column info
	rows, err := m.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		m.err = err
		return
	}
	defer rows.Close()

	m.columns = []string{}
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
		// Handle search mode first
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

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			switch m.mode {
			case modeTableData, modeQuery:
				m.mode = modeTableList
				m.err = nil
			}

		case "/":
			if m.mode == modeTableList {
				m.searching = true
				m.searchInput = ""
			}

		case "enter":
			switch m.mode {
			case modeTableList:
				if len(m.filteredTables) > 0 {
					m.mode = modeTableData
					m.currentPage = 0
					m.loadTableData()
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
			}

		case "left", "h":
			switch m.mode {
			case modeTableData:
				if m.currentPage > 0 {
					m.currentPage--
					m.loadTableData()
				}
			case modeTableList:
				if m.tableListPage > 0 {
					m.tableListPage--
					// Adjust selection to stay in view
					visibleCount := m.getVisibleTableCount()
					m.selectedTable = m.tableListPage * visibleCount
				}
			}

		case "right", "l":
			switch m.mode {
			case modeTableData:
				maxPage := (m.totalRows - 1) / pageSize
				if m.currentPage < maxPage {
					m.currentPage++
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
			}

		case "s":
			if m.mode == modeTableList {
				m.mode = modeQuery
				m.queryInput = ""
				m.query = ""
				m.cursor = 0
			}

		case "r":
			if m.mode == modeTableList {
				m.loadTables()
			} else if m.mode == modeTableData {
				m.loadTableData()
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
		content.WriteString("\n\n")

		if len(m.tableData) == 0 {
			content.WriteString("No data in table")
		} else {
			// Limit rows to fit screen
			visibleRows := m.getVisibleDataRowCount()
			displayRows := min(len(m.tableData), visibleRows)
			
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

			// Add data rows
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
		}

		content.WriteString("\n")
		content.WriteString(helpStyle.Render(fmt.Sprintf("←/→: prev/next page • Total rows: %d • esc: back • r: refresh • q: quit", m.totalRows)))

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