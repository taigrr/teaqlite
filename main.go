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
	db           *sql.DB
	mode         viewMode
	tables       []string
	selectedTable int
	tableData    [][]string
	columns      []string
	currentPage  int
	totalRows    int
	query        string
	queryInput   string
	cursor       int
	err          error
	width        int
	height       int
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
		db:          db,
		mode:        modeTableList,
		currentPage: 0,
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
}

func (m *model) loadTableData() {
	if m.selectedTable >= len(m.tables) {
		return
	}

	tableName := m.tables[m.selectedTable]
	
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
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			switch m.mode {
			case modeTableData, modeQuery:
				m.mode = modeTableList
				m.err = nil
			}

		case "enter":
			switch m.mode {
			case modeTableList:
				if len(m.tables) > 0 {
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
				}
			}

		case "down", "j":
			switch m.mode {
			case modeTableList:
				if m.selectedTable < len(m.tables)-1 {
					m.selectedTable++
				}
			}

		case "left", "h":
			switch m.mode {
			case modeTableData:
				if m.currentPage > 0 {
					m.currentPage--
					m.loadTableData()
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
				m.queryInput += msg.String()
				m.query = m.queryInput
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
		content.WriteString(titleStyle.Render("SQLite TUI - Tables"))
		content.WriteString("\n\n")

		if len(m.tables) == 0 {
			content.WriteString("No tables found in database")
		} else {
			for i, table := range m.tables {
				if i == m.selectedTable {
					content.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", table)))
				} else {
					content.WriteString(normalStyle.Render(fmt.Sprintf("  %s", table)))
				}
				content.WriteString("\n")
			}
		}

		content.WriteString("\n")
		content.WriteString(helpStyle.Render("↑/↓: navigate • enter: view table • s: SQL query • r: refresh • q: quit"))

	case modeTableData:
		tableName := m.tables[m.selectedTable]
		maxPage := (m.totalRows - 1) / pageSize
		
		content.WriteString(titleStyle.Render(fmt.Sprintf("Table: %s (Page %d/%d)", tableName, m.currentPage+1, maxPage+1)))
		content.WriteString("\n\n")

		if len(m.tableData) == 0 {
			content.WriteString("No data in table")
		} else {
			// Create table header
			var headerRow strings.Builder
			for i, col := range m.columns {
				if i > 0 {
					headerRow.WriteString(" | ")
				}
				headerRow.WriteString(fmt.Sprintf("%-15s", truncateString(col, 15)))
			}
			content.WriteString(selectedStyle.Render(headerRow.String()))
			content.WriteString("\n")

			// Add separator
			var separator strings.Builder
			for i := range m.columns {
				if i > 0 {
					separator.WriteString("-+-")
				}
				separator.WriteString(strings.Repeat("-", 15))
			}
			content.WriteString(separator.String())
			content.WriteString("\n")

			// Add data rows
			for _, row := range m.tableData {
				var dataRow strings.Builder
				for i, cell := range row {
					if i > 0 {
						dataRow.WriteString(" | ")
					}
					dataRow.WriteString(fmt.Sprintf("%-15s", truncateString(cell, 15)))
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
			// Show query results
			var headerRow strings.Builder
			for i, col := range m.columns {
				if i > 0 {
					headerRow.WriteString(" | ")
				}
				headerRow.WriteString(fmt.Sprintf("%-15s", truncateString(col, 15)))
			}
			content.WriteString(selectedStyle.Render(headerRow.String()))
			content.WriteString("\n")

			var separator strings.Builder
			for i := range m.columns {
				if i > 0 {
					separator.WriteString("-+-")
				}
				separator.WriteString(strings.Repeat("-", 15))
			}
			content.WriteString(separator.String())
			content.WriteString("\n")

			for _, row := range m.tableData {
				var dataRow strings.Builder
				for i, cell := range row {
					if i > 0 {
						dataRow.WriteString(" | ")
					}
					dataRow.WriteString(fmt.Sprintf("%-15s", truncateString(cell, 15)))
				}
				content.WriteString(normalStyle.Render(dataRow.String()))
				content.WriteString("\n")
			}
		}

		content.WriteString("\n")
		content.WriteString(helpStyle.Render("enter: execute query • esc: back • q: quit"))
	}

	return tableStyle.Render(content.String())
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
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