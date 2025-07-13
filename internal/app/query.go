package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Query Model
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

func (m *QueryModel) executeQuery() tea.Cmd {
	return func() tea.Msg {
		rows, err := m.Shared.DB.Query(m.query)
		if err != nil {
			m.err = err
			return nil
		}
		defer rows.Close()

		// Get column names
		columns, err := rows.Columns()
		if err != nil {
			m.err = err
			return nil
		}
		m.columns = columns

		// Get results
		m.results = [][]string{}
		for rows.Next() {
			values := make([]any, len(columns))
			valuePtrs := make([]any, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				m.err = err
				return nil
			}

			row := make([]string, len(columns))
			for i, val := range values {
				if val == nil {
					row[i] = "NULL"
				} else {
					row[i] = fmt.Sprintf("%v", val)
				}
			}
			m.results = append(m.results, row)
		}

		// Update shared data for row detail view
		m.Shared.FilteredData = m.results
		m.Shared.Columns = m.columns
		m.Shared.IsQueryResult = true

		m.FocusOnInput = false
		m.selectedRow = 0
		m.err = nil

		return nil
	}
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

		// Data rows
		visibleCount := Max(1, m.Shared.Height-10)
		endIdx := Min(len(m.results), visibleCount)

		for i := 0; i < endIdx; i++ {
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