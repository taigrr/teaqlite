package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Table Data Model
type tableDataModel struct {
	shared      *sharedData
	selectedRow int
	searchInput string
	searching   bool
}

func newTableDataModel(shared *sharedData) *tableDataModel {
	return &tableDataModel{
		shared:      shared,
		selectedRow: 0,
	}
}

func (m *tableDataModel) Init() tea.Cmd {
	return nil
}

func (m *tableDataModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searching {
			return m.handleSearchInput(msg)
		}
		return m.handleNavigation(msg)
	}
	return m, nil
}

func (m *tableDataModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.searching = false
		m.filterData()
	case "backspace":
		if len(m.searchInput) > 0 {
			m.searchInput = m.searchInput[:len(m.searchInput)-1]
			m.filterData()
		}
	default:
		if len(msg.String()) == 1 {
			m.searchInput += msg.String()
			m.filterData()
		}
	}
	return m, nil
}

func (m *tableDataModel) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return switchToTableListMsg{} }

	case "/":
		m.searching = true
		m.searchInput = ""
		return m, nil

	case "enter":
		if len(m.shared.filteredData) > 0 {
			return m, func() tea.Msg {
				return switchToRowDetailMsg{rowIndex: m.selectedRow}
			}
		}

	case "r":
		return m, func() tea.Msg { return refreshDataMsg{} }

	case "up", "k":
		if m.selectedRow > 0 {
			m.selectedRow--
		}

	case "down", "j":
		if m.selectedRow < len(m.shared.filteredData)-1 {
			m.selectedRow++
		}

	case "left", "h":
		if m.shared.currentPage > 0 {
			m.shared.currentPage--
			m.selectedRow = 0
			return m, func() tea.Msg { return refreshDataMsg{} }
		}

	case "right", "l":
		maxPage := max(0, (m.shared.totalRows-1)/pageSize)
		if m.shared.currentPage < maxPage {
			m.shared.currentPage++
			m.selectedRow = 0
			return m, func() tea.Msg { return refreshDataMsg{} }
		}
	}
	return m, nil
}

func (m *tableDataModel) filterData() {
	if m.searchInput == "" {
		m.shared.filteredData = make([][]string, len(m.shared.tableData))
		copy(m.shared.filteredData, m.shared.tableData)
	} else {
		m.shared.filteredData = [][]string{}
		searchLower := strings.ToLower(m.searchInput)
		for _, row := range m.shared.tableData {
			found := false
			for _, cell := range row {
				if strings.Contains(strings.ToLower(cell), searchLower) {
					found = true
					break
				}
			}
			if found {
				m.shared.filteredData = append(m.shared.filteredData, row)
			}
		}
	}

	if m.selectedRow >= len(m.shared.filteredData) {
		m.selectedRow = 0
	}
}

func (m *tableDataModel) getVisibleRowCount() int {
	reservedLines := 9
	if m.searching {
		reservedLines += 2
	}
	return max(1, m.shared.height-reservedLines)
}

func (m *tableDataModel) View() string {
	var content strings.Builder

	tableName := m.shared.filteredTables[m.shared.selectedTable]
	maxPage := max(0, (m.shared.totalRows-1)/pageSize)

	content.WriteString(titleStyle.Render(fmt.Sprintf("Table: %s (Page %d/%d)",
		tableName, m.shared.currentPage+1, maxPage+1)))
	content.WriteString("\n")

	if m.searching {
		content.WriteString("\nSearch data: " + m.searchInput + "_")
		content.WriteString("\n")
	} else if m.searchInput != "" {
		content.WriteString(fmt.Sprintf("\nFiltered by: %s (%d/%d rows)",
			m.searchInput, len(m.shared.filteredData), len(m.shared.tableData)))
		content.WriteString("\n")
	}
	content.WriteString("\n")

	if len(m.shared.filteredData) == 0 {
		if m.searchInput != "" {
			content.WriteString("No rows match your search")
		} else {
			content.WriteString("No data in table")
		}
	} else {
		visibleRows := m.getVisibleRowCount()
		displayRows := min(len(m.shared.filteredData), visibleRows)

		// Create table header
		colWidth := 10
		if len(m.shared.columns) > 0 && m.shared.width > 0 {
			colWidth = max(10, (m.shared.width-len(m.shared.columns)*3)/len(m.shared.columns))
		}

		var headerRow strings.Builder
		for i, col := range m.shared.columns {
			if i > 0 {
				headerRow.WriteString(" | ")
			}
			headerRow.WriteString(fmt.Sprintf("%-*s", colWidth, truncateString(col, colWidth)))
		}
		content.WriteString(selectedStyle.Render(headerRow.String()))
		content.WriteString("\n")

		// Add separator
		var separator strings.Builder
		for i := range m.shared.columns {
			if i > 0 {
				separator.WriteString("-+-")
			}
			separator.WriteString(strings.Repeat("-", colWidth))
		}
		content.WriteString(separator.String())
		content.WriteString("\n")

		// Add data rows with highlighting
		for i := range displayRows {
			if i >= len(m.shared.filteredData) {
				break
			}
			row := m.shared.filteredData[i]
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
	if m.searching {
		content.WriteString(helpStyle.Render("Type to search • enter/esc: finish search"))
	} else {
		content.WriteString(helpStyle.Render("↑/↓: select row • ←/→: page • /: search • enter: view row • esc: back • r: refresh • ctrl+c: quit"))
	}

	return content.String()
}
