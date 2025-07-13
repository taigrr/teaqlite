package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "modernc.org/sqlite" // Import SQLite driver
)

const (
	pageSize = 20
)

// Custom message types
type (
	switchToTableListMsg          struct{}
	switchToTableDataMsg          struct{ tableIndex int }
	switchToRowDetailMsg          struct{ rowIndex int }
	switchToRowDetailFromQueryMsg struct{ rowIndex int }
	switchToEditCellMsg           struct{ rowIndex, colIndex int }
	switchToQueryMsg              struct{}
	returnToQueryMsg              struct{} // Return to query mode from row detail
	refreshDataMsg                struct{}
	updateCellMsg                 struct {
		rowIndex, colIndex int
		value              string
	}
	executeQueryMsg struct{ query string }
)

// Main application model
type model struct {
	db          *sql.DB
	currentView tea.Model
	width       int
	height      int
	err         error
}

// Shared data that all models need access to
type sharedData struct {
	db             *sql.DB
	tables         []string
	filteredTables []string
	tableData      [][]string
	filteredData   [][]string
	columns        []string
	primaryKeys    []string
	selectedTable  int
	totalRows      int
	currentPage    int
	width          int
	height         int
	// Query result context
	isQueryResult  bool
	queryTableName string // For simple queries, store the source table
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

func newSharedData(db *sql.DB) *sharedData {
	return &sharedData{
		db:             db,
		filteredTables: []string{},
		filteredData:   [][]string{},
		width:          80,
		height:         24,
	}
}

func (s *sharedData) loadTables() error {
	query := `SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	s.tables = []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		s.tables = append(s.tables, name)
	}
	s.filteredTables = make([]string, len(s.tables))
	copy(s.filteredTables, s.tables)
	return nil
}

func (s *sharedData) loadTableData() error {
	if s.selectedTable >= len(s.filteredTables) {
		return fmt.Errorf("invalid table selection")
	}

	tableName := s.filteredTables[s.selectedTable]

	// Get column info and primary keys
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	s.columns = []string{}
	s.primaryKeys = []string{}
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		s.columns = append(s.columns, name)
		if pk == 1 {
			s.primaryKeys = append(s.primaryKeys, name)
		}
	}

	// Get total row count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	err = s.db.QueryRow(countQuery).Scan(&s.totalRows)
	if err != nil {
		return err
	}

	// Get paginated data
	offset := s.currentPage * pageSize
	dataQuery := fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", tableName, pageSize, offset)

	rows, err = s.db.Query(dataQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	s.tableData = [][]string{}
	for rows.Next() {
		values := make([]any, len(s.columns))
		valuePtrs := make([]any, len(s.columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		row := make([]string, len(s.columns))
		for i, val := range values {
			if val == nil {
				row[i] = "NULL"
			} else {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		s.tableData = append(s.tableData, row)
	}

	s.filteredData = make([][]string, len(s.tableData))
	copy(s.filteredData, s.tableData)

	// Reset query result context since this is regular table data
	s.isQueryResult = false
	s.queryTableName = ""

	return nil
}

func (s *sharedData) updateCell(rowIndex, colIndex int, newValue string) error {
	if rowIndex >= len(s.filteredData) || colIndex >= len(s.columns) {
		return fmt.Errorf("invalid row or column index")
	}

	var tableName string
	var err error

	if s.isQueryResult {
		// For query results, try to determine the source table
		if s.queryTableName != "" {
			tableName = s.queryTableName
		} else {
			// Try to infer table from column names and data
			tableName, err = s.inferTableFromQueryResult(rowIndex, colIndex)
			if err != nil {
				return fmt.Errorf("cannot determine source table for query result: %v", err)
			}
		}
	} else {
		// For regular table data
		tableName = s.filteredTables[s.selectedTable]
	}

	columnName := s.columns[colIndex]

	// Get table info for the target table to find primary keys
	tableColumns, tablePrimaryKeys, err := s.getTableInfo(tableName)
	if err != nil {
		return fmt.Errorf("failed to get table info for %s: %v", tableName, err)
	}

	// Build WHERE clause using primary keys or all columns if no primary key
	var whereClause strings.Builder
	var args []any

	if len(tablePrimaryKeys) > 0 {
		// Use primary keys for WHERE clause
		for i, pkCol := range tablePrimaryKeys {
			if i > 0 {
				whereClause.WriteString(" AND ")
			}

			// Find the value for this primary key in our data
			pkValue, err := s.findColumnValue(rowIndex, pkCol, tableColumns)
			if err != nil {
				return fmt.Errorf("failed to find primary key value for %s: %v", pkCol, err)
			}

			whereClause.WriteString(fmt.Sprintf("%s = ?", pkCol))
			args = append(args, pkValue)
		}
	} else {
		// Use all columns for WHERE clause (less reliable but works)
		for i, col := range tableColumns {
			if i > 0 {
				whereClause.WriteString(" AND ")
			}

			colValue, err := s.findColumnValue(rowIndex, col, tableColumns)
			if err != nil {
				return fmt.Errorf("failed to find column value for %s: %v", col, err)
			}

			whereClause.WriteString(fmt.Sprintf("%s = ?", col))
			args = append(args, colValue)
		}
	}

	// Execute UPDATE
	updateQuery := fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s", tableName, columnName, whereClause.String())
	args = append([]any{newValue}, args...)

	_, err = s.db.Exec(updateQuery, args...)
	if err != nil {
		return err
	}

	// Update local data
	s.filteredData[rowIndex][colIndex] = newValue
	// Also update the original data if it exists
	for i, row := range s.tableData {
		if len(row) > colIndex {
			match := true
			for j, cell := range row {
				if j < len(s.filteredData[rowIndex]) && cell != s.filteredData[rowIndex][j] && j != colIndex {
					match = false
					break
				}
			}
			if match {
				s.tableData[i][colIndex] = newValue
				break
			}
		}
	}

	return nil
}

// Helper function to get table info
func (s *sharedData) getTableInfo(tableName string) ([]string, []string, error) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var columns []string
	var primaryKeys []string

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return nil, nil, err
		}
		columns = append(columns, name)
		if pk == 1 {
			primaryKeys = append(primaryKeys, name)
		}
	}

	return columns, primaryKeys, nil
}

// Helper function to find a column value in the current row
func (s *sharedData) findColumnValue(rowIndex int, columnName string, _ []string) (string, error) {
	// First try to find it in our current columns (for query results)
	for i, col := range s.columns {
		if col == columnName && i < len(s.filteredData[rowIndex]) {
			return s.filteredData[rowIndex][i], nil
		}
	}

	// If not found, this might be a column that's not in the query result
	// We'll need to query the database to get the current value
	if s.isQueryResult && len(s.primaryKeys) > 0 {
		// Build a query to get the missing column value using available primary keys
		var whereClause strings.Builder
		var args []any

		for i, pkCol := range s.primaryKeys {
			if i > 0 {
				whereClause.WriteString(" AND ")
			}

			// Find primary key value in our data
			pkIndex := -1
			for j, col := range s.columns {
				if col == pkCol {
					pkIndex = j
					break
				}
			}

			if pkIndex >= 0 {
				whereClause.WriteString(fmt.Sprintf("%s = ?", pkCol))
				args = append(args, s.filteredData[rowIndex][pkIndex])
			}
		}

		if whereClause.Len() > 0 {
			tableName := s.queryTableName
			if tableName == "" {
				// Try to infer table name
				tableName, _ = s.inferTableFromQueryResult(rowIndex, 0)
			}

			query := fmt.Sprintf("SELECT %s FROM %s WHERE %s", columnName, tableName, whereClause.String())
			var value string
			err := s.db.QueryRow(query, args...).Scan(&value)
			if err != nil {
				return "", err
			}
			return value, nil
		}
	}

	return "", fmt.Errorf("column %s not found in current data", columnName)
}

// Helper function to try to infer the source table from query results
func (s *sharedData) inferTableFromQueryResult(_, _ int) (string, error) {
	// This is a simple heuristic - try to find a table that has all our columns
	for _, tableName := range s.tables {
		tableColumns, _, err := s.getTableInfo(tableName)
		if err != nil {
			continue
		}

		// Check if this table has all our columns
		hasAllColumns := true
		for _, queryCol := range s.columns {
			found := slices.Contains(tableColumns, queryCol)
			if !found {
				hasAllColumns = false
				break
			}
		}

		if hasAllColumns {
			// Cache this for future use
			s.queryTableName = tableName
			return tableName, nil
		}
	}

	return "", fmt.Errorf("could not infer source table from query result")
}

// Styles
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
)

// Utility functions
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
		if len(currentLine)+len(word)+1 > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
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

func initialModel(dbPath string) model {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return model{err: err}
	}

	shared := newSharedData(db)
	if err := shared.loadTables(); err != nil {
		return model{err: err}
	}

	return model{
		db:          db,
		currentView: newTableListModel(shared),
		width:       80,
		height:      24,
	}
}

func (m model) Init() tea.Cmd {
	return m.currentView.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update current view with new dimensions
		if tableList, ok := m.currentView.(*tableListModel); ok {
			tableList.shared.width = m.width
			tableList.shared.height = m.height
		}
		// Add similar updates for other model types as needed

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case switchToTableListMsg:
		m.currentView = newTableListModel(m.getSharedData())
		return m, nil

	case switchToTableDataMsg:
		shared := m.getSharedData()
		shared.selectedTable = msg.tableIndex
		if err := shared.loadTableData(); err != nil {
			m.err = err
			return m, nil
		}
		m.currentView = newTableDataModel(shared)
		return m, nil

	case switchToRowDetailMsg:
		m.currentView = newRowDetailModel(m.getSharedData(), msg.rowIndex)
		return m, nil

	case switchToRowDetailFromQueryMsg:
		rowDetail := newRowDetailModel(m.getSharedData(), msg.rowIndex)
		rowDetail.fromQuery = true
		m.currentView = rowDetail
		return m, nil

	case switchToEditCellMsg:
		m.currentView = newEditCellModel(m.getSharedData(), msg.rowIndex, msg.colIndex)
		return m, nil

	case switchToQueryMsg:
		m.currentView = newQueryModel(m.getSharedData())
		return m, nil

	case returnToQueryMsg:
		// Return to query mode, preserving the query state if possible
		if queryView, ok := m.currentView.(*queryModel); ok {
			// If we're already in query mode, just switch focus back to results
			queryView.focusOnInput = false
		} else {
			// Create new query model
			m.currentView = newQueryModel(m.getSharedData())
		}
		return m, nil

	case refreshDataMsg:
		shared := m.getSharedData()
		if err := shared.loadTableData(); err != nil {
			m.err = err
		}
		return m, nil

	case updateCellMsg:
		shared := m.getSharedData()
		if err := shared.updateCell(msg.rowIndex, msg.colIndex, msg.value); err != nil {
			m.err = err
		}
		return m, func() tea.Msg { return switchToRowDetailMsg{msg.rowIndex} }
	}

	if m.err != nil {
		return m, nil
	}

	var cmd tea.Cmd
	m.currentView, cmd = m.currentView.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress 'q' to quit", m.err))
	}
	return m.currentView.View()
}

func (m model) getSharedData() *sharedData {
	// Extract shared data from current view
	switch v := m.currentView.(type) {
	case *tableListModel:
		return v.shared
	case *tableDataModel:
		return v.shared
	case *rowDetailModel:
		return v.shared
	case *editCellModel:
		return v.shared
	case *queryModel:
		return v.shared
	default:
		// Fallback - create new shared data
		return newSharedData(m.db)
	}
}
