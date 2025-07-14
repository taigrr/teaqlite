package app

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "modernc.org/sqlite" // Import SQLite driver
)

const (
	PageSize = 20
)

// Custom message types
type (
	SwitchToTableListMsg          struct{}
	SwitchToTableListClearMsg     struct{} // Switch to table list and clear any filter
	SwitchToTableDataMsg          struct{ TableIndex int }
	SwitchToRowDetailMsg          struct{ RowIndex int }
	SwitchToRowDetailFromQueryMsg struct{ RowIndex int }
	SwitchToEditCellMsg           struct{ RowIndex, ColIndex int }
	SwitchToQueryMsg              struct{}
	ReturnToQueryMsg              struct{} // Return to query mode from row detail
	RefreshDataMsg                struct{}
	UpdateCellMsg                 struct {
		RowIndex, ColIndex int
		Value              string
	}
	ExecuteQueryMsg   struct{ Query string }
	QueryCompletedMsg struct {
		Results [][]string
		Columns []string
		Error   error
	}
)

// Model is the main application model
type Model struct {
	db          *sql.DB
	currentView tea.Model
	width       int
	height      int
	err         error
}

// SharedData that all models need access to
type SharedData struct {
	DB             *sql.DB
	Tables         []string
	FilteredTables []string
	TableData      [][]string
	FilteredData   [][]string
	Columns        []string
	PrimaryKeys    []string
	SelectedTable  int
	TotalRows      int
	CurrentPage    int
	Width          int
	Height         int
	// Query result context
	IsQueryResult  bool
	QueryTableName string // For simple queries, store the source table
}

func NewSharedData(db *sql.DB) *SharedData {
	return &SharedData{
		DB:             db,
		FilteredTables: []string{},
		FilteredData:   [][]string{},
		Width:          80,
		Height:         24,
	}
}

func (s *SharedData) LoadTables() error {
	query := `SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`
	rows, err := s.DB.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	s.Tables = []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		s.Tables = append(s.Tables, name)
	}
	s.FilteredTables = make([]string, len(s.Tables))
	copy(s.FilteredTables, s.Tables)
	return nil
}

func (s *SharedData) LoadTableData() error {
	if s.SelectedTable >= len(s.FilteredTables) {
		return fmt.Errorf("invalid table selection")
	}

	tableName := s.FilteredTables[s.SelectedTable]

	// Get column info and primary keys
	rows, err := s.DB.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	s.Columns = []string{}
	s.PrimaryKeys = []string{}
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue sql.NullString

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		s.Columns = append(s.Columns, name)
		if pk == 1 {
			s.PrimaryKeys = append(s.PrimaryKeys, name)
		}
	}

	// Get total row count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	err = s.DB.QueryRow(countQuery).Scan(&s.TotalRows)
	if err != nil {
		return err
	}

	// Get paginated data
	offset := s.CurrentPage * PageSize
	dataQuery := fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", tableName, PageSize, offset)

	rows, err = s.DB.Query(dataQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	s.TableData = [][]string{}
	for rows.Next() {
		values := make([]any, len(s.Columns))
		valuePtrs := make([]any, len(s.Columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		row := make([]string, len(s.Columns))
		for i, val := range values {
			if val == nil {
				row[i] = "NULL"
			} else {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		s.TableData = append(s.TableData, row)
	}

	s.FilteredData = make([][]string, len(s.TableData))
	copy(s.FilteredData, s.TableData)

	// Reset query result context since this is regular table data
	s.IsQueryResult = false
	s.QueryTableName = ""

	return nil
}

func (s *SharedData) UpdateCell(rowIndex, colIndex int, newValue string) error {
	if rowIndex >= len(s.FilteredData) || colIndex >= len(s.Columns) {
		return fmt.Errorf("invalid row or column index")
	}

	var tableName string
	var err error

	if s.IsQueryResult {
		// For query results, try to determine the source table
		if s.QueryTableName != "" {
			tableName = s.QueryTableName
		} else {
			// Try to infer table from column names and data
			tableName, err = s.inferTableFromQueryResult(rowIndex, colIndex)
			if err != nil {
				return fmt.Errorf("cannot determine source table for query result: %v", err)
			}
		}
	} else {
		// For regular table data
		tableName = s.FilteredTables[s.SelectedTable]
	}

	columnName := s.Columns[colIndex]

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

	_, err = s.DB.Exec(updateQuery, args...)
	if err != nil {
		return err
	}

	// Update local data
	s.FilteredData[rowIndex][colIndex] = newValue
	// Also update the original data if it exists
	for i, row := range s.TableData {
		if len(row) > colIndex {
			match := true
			for j, cell := range row {
				if j < len(s.FilteredData[rowIndex]) && cell != s.FilteredData[rowIndex][j] && j != colIndex {
					match = false
					break
				}
			}
			if match {
				s.TableData[i][colIndex] = newValue
				break
			}
		}
	}

	return nil
}

// Helper function to get table info
func (s *SharedData) getTableInfo(tableName string) ([]string, []string, error) {
	rows, err := s.DB.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
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
func (s *SharedData) findColumnValue(rowIndex int, columnName string, _ []string) (string, error) {
	// First try to find it in our current columns (for query results)
	for i, col := range s.Columns {
		if col == columnName && i < len(s.FilteredData[rowIndex]) {
			return s.FilteredData[rowIndex][i], nil
		}
	}

	// If not found, this might be a column that's not in the query result
	// We'll need to query the database to get the current value
	if s.IsQueryResult && len(s.PrimaryKeys) > 0 {
		// Build a query to get the missing column value using available primary keys
		var whereClause strings.Builder
		var args []any

		for i, pkCol := range s.PrimaryKeys {
			if i > 0 {
				whereClause.WriteString(" AND ")
			}

			// Find primary key value in our data
			pkIndex := -1
			for j, col := range s.Columns {
				if col == pkCol {
					pkIndex = j
					break
				}
			}

			if pkIndex >= 0 {
				whereClause.WriteString(fmt.Sprintf("%s = ?", pkCol))
				args = append(args, s.FilteredData[rowIndex][pkIndex])
			}
		}

		if whereClause.Len() > 0 {
			tableName := s.QueryTableName
			if tableName == "" {
				// Try to infer table name
				tableName, _ = s.inferTableFromQueryResult(rowIndex, 0)
			}

			query := fmt.Sprintf("SELECT %s FROM %s WHERE %s", columnName, tableName, whereClause.String())
			var value string
			err := s.DB.QueryRow(query, args...).Scan(&value)
			if err != nil {
				return "", err
			}
			return value, nil
		}
	}

	return "", fmt.Errorf("column %s not found in current data", columnName)
}

// Helper function to try to infer the source table from query results
func (s *SharedData) inferTableFromQueryResult(_, _ int) (string, error) {
	// This is a simple heuristic - try to find a table that has all our columns
	for _, tableName := range s.Tables {
		tableColumns, _, err := s.getTableInfo(tableName)
		if err != nil {
			continue
		}

		// Check if this table has all our columns
		hasAllColumns := true
		for _, queryCol := range s.Columns {
			found := slices.Contains(tableColumns, queryCol)
			if !found {
				hasAllColumns = false
				break
			}
		}

		if hasAllColumns {
			// Cache this for future use
			s.QueryTableName = tableName
			return tableName, nil
		}
	}

	return "", fmt.Errorf("could not infer source table from query result")
}

// Styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#F25D94"))

	NormalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))
)

// Utility functions
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func WrapText(text string, width int) []string {
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

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func InitialModel(db *sql.DB) *Model {
	shared := NewSharedData(db)
	if err := shared.LoadTables(); err != nil {
		return &Model{err: err}
	}

	return &Model{
		db:          db,
		currentView: NewTableListModel(shared),
		width:       80,
		height:      24,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.currentView.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update current view with new dimensions
		if tableList, ok := m.currentView.(*TableListModel); ok {
			tableList.Shared.Width = m.width
			tableList.Shared.Height = m.height
		}
		// Add similar updates for other model types as needed

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "ctrl+z" {
			return m, tea.Suspend
		}

	case SwitchToTableListMsg:
		m.currentView = NewTableListModel(m.getSharedData())
		return m, nil

	case SwitchToTableListClearMsg:
		shared := m.getSharedData()
		// Clear any table filter
		shared.FilteredTables = make([]string, len(shared.Tables))
		copy(shared.FilteredTables, shared.Tables)
		m.currentView = NewTableListModel(shared)
		return m, nil

	case SwitchToTableDataMsg:
		shared := m.getSharedData()
		shared.SelectedTable = msg.TableIndex
		if err := shared.LoadTableData(); err != nil {
			m.err = err
			return m, nil
		}
		m.currentView = NewTableDataModel(shared)
		return m, nil

	case SwitchToRowDetailMsg:
		m.currentView = NewRowDetailModel(m.getSharedData(), msg.RowIndex)
		return m, nil

	case SwitchToRowDetailFromQueryMsg:
		rowDetail := NewRowDetailModel(m.getSharedData(), msg.RowIndex)
		rowDetail.FromQuery = true
		m.currentView = rowDetail
		return m, nil

	case SwitchToEditCellMsg:
		m.currentView = NewEditCellModel(m.getSharedData(), msg.RowIndex, msg.ColIndex)
		return m, nil

	case SwitchToQueryMsg:
		m.currentView = NewQueryModel(m.getSharedData())
		return m, nil

	case ReturnToQueryMsg:
		// Return to query mode, preserving the query state if possible
		if queryView, ok := m.currentView.(*QueryModel); ok {
			// If we're already in query mode, just switch focus back to results
			queryView.FocusOnInput = false
		} else {
			// Create new query model
			m.currentView = NewQueryModel(m.getSharedData())
		}
		return m, nil

	case RefreshDataMsg:
		shared := m.getSharedData()
		if err := shared.LoadTableData(); err != nil {
			m.err = err
		}
		return m, nil

	case UpdateCellMsg:
		shared := m.getSharedData()
		if err := shared.UpdateCell(msg.RowIndex, msg.ColIndex, msg.Value); err != nil {
			m.err = err
		}
		return m, func() tea.Msg { return SwitchToRowDetailMsg{msg.RowIndex} }

	case QueryCompletedMsg:
		// Forward the query completion to the query model
		if queryModel, ok := m.currentView.(*QueryModel); ok {
			queryModel.handleQueryCompletion(msg)
		}
		return m, nil
	}

	if m.err != nil {
		return m, nil
	}

	var cmd tea.Cmd
	m.currentView, cmd = m.currentView.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v\n\nPress 'ctrl+c' to quit", m.err))
	}
	return m.currentView.View()
}

func (m *Model) Err() error {
	return m.err
}

func (m *Model) getSharedData() *SharedData {
	// Extract shared data from current view
	switch v := m.currentView.(type) {
	case *TableListModel:
		return v.Shared
	case *TableDataModel:
		return v.Shared
	case *RowDetailModel:
		return v.Shared
	case *EditCellModel:
		return v.Shared
	case *QueryModel:
		return v.Shared
	default:
		// Fallback - create new shared data
		return NewSharedData(m.db)
	}
}
