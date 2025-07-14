package app

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
)

type QueryModel struct {
	Shared       *SharedData
	queryInput   textarea.Model
	FocusOnInput bool
	selectedRow  int
	results      [][]string
	columns      []string
	err          error
	blinkState   bool
	gPressed     bool
	keyMap       QueryKeyMap
	help         help.Model
	focused      bool
	id           int
}

// QueryOption is a functional option for configuring QueryModel
type QueryOption func(*QueryModel)

// WithQueryKeyMap sets the key map
func WithQueryKeyMap(km QueryKeyMap) QueryOption {
	return func(m *QueryModel) {
		m.keyMap = km
	}
}

func NewQueryModel(shared *SharedData, opts ...QueryOption) *QueryModel {
	queryInput := textarea.New()
	queryInput.Placeholder = "Enter SQL query..."
	queryInput.SetWidth(60)
	queryInput.SetHeight(3)
	queryInput.Focus()

	m := &QueryModel{
		Shared:       shared,
		queryInput:   queryInput,
		FocusOnInput: true,
		selectedRow:  0,
		blinkState:   true,
		keyMap:       DefaultQueryKeyMap(),
		help:         help.New(),
		focused:      true,
		id:           nextID(),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// ID returns the unique ID of the model
func (m QueryModel) ID() int {
	return m.id
}

// Focus sets the focus state
func (m *QueryModel) Focus() {
	m.focused = true
	if m.FocusOnInput {
		m.queryInput.Focus()
	}
}

// Blur removes focus
func (m *QueryModel) Blur() {
	m.focused = false
	m.queryInput.Blur()
}

// Focused returns the focus state
func (m QueryModel) Focused() bool {
	return m.focused
}

func (m *QueryModel) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
			return blinkMsg{}
		}),
	)
}

func (m *QueryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case blinkMsg:
		m.blinkState = !m.blinkState
		cmds = append(cmds, tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
			return blinkMsg{}
		}))
		
	case tea.KeyMsg:
		if m.FocusOnInput {
			return m.handleQueryInput(msg)
		}
		return m.handleResultsNavigation(msg)
	}

	// Update query input for non-key messages when focused on input
	if m.FocusOnInput {
		var cmd tea.Cmd
		m.queryInput, cmd = m.queryInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *QueryModel) handleQueryInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Escape):
		return m, func() tea.Msg { return SwitchToTableListClearMsg{} }

	case key.Matches(msg, m.keyMap.Execute):
		if strings.TrimSpace(m.queryInput.Value()) != "" {
			return m, m.executeQuery()
		}

	default:
		var cmd tea.Cmd
		m.queryInput, cmd = m.queryInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *QueryModel) handleResultsNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Escape), key.Matches(msg, m.keyMap.Back):
		m.gPressed = false
		return m, func() tea.Msg { return SwitchToTableListClearMsg{} }

	case key.Matches(msg, m.keyMap.GoToStart):
		if m.gPressed {
			// Second g - go to beginning
			m.selectedRow = 0
			m.gPressed = false
		} else {
			// First g - wait for second g
			m.gPressed = true
		}
		return m, nil

	case key.Matches(msg, m.keyMap.GoToEnd):
		// Go to end
		if len(m.results) > 0 {
			m.selectedRow = len(m.results) - 1
		}
		m.gPressed = false
		return m, nil

	case key.Matches(msg, m.keyMap.EditQuery):
		m.gPressed = false
		m.FocusOnInput = true
		m.queryInput.Focus()
		return m, nil

	case key.Matches(msg, m.keyMap.Enter):
		m.gPressed = false
		if len(m.results) > 0 {
			return m, func() tea.Msg {
				return SwitchToRowDetailFromQueryMsg{RowIndex: m.selectedRow}
			}
		}

	case key.Matches(msg, m.keyMap.Up):
		m.gPressed = false
		if m.selectedRow > 0 {
			m.selectedRow--
		}

	case key.Matches(msg, m.keyMap.Down):
		m.gPressed = false
		if m.selectedRow < len(m.results)-1 {
			m.selectedRow++
		}

	default:
		// Any other key resets the g state
		m.gPressed = false
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
		modifiedQuery := m.ensureIDColumns(m.queryInput.Value())

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
	m.queryInput.Blur()
	m.selectedRow = 0
	m.err = nil
}

func (m *QueryModel) View() string {
	var content strings.Builder

	content.WriteString(TitleStyle.Render("SQL Query"))
	content.WriteString("\n\n")

	// Query input
	content.WriteString("Query:\n")
	content.WriteString(m.queryInput.View())
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
		content.WriteString(m.help.View(m.keyMap))
	}

	return content.String()
}