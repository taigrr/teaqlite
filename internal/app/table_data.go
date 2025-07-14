package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type TableDataModel struct {
	Shared      *SharedData
	selectedRow int
	searchInput string
	searching   bool
	gPressed    bool
}

func NewTableDataModel(shared *SharedData) *TableDataModel {
	return &TableDataModel{
		Shared:      shared,
		selectedRow: 0,
	}
}

func (m *TableDataModel) Init() tea.Cmd {
	return nil
}

func (m *TableDataModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searching {
			return m.handleSearchInput(msg)
		}
		return m.handleNavigation(msg)
	}
	return m, nil
}

func (m *TableDataModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m *TableDataModel) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.gPressed = false
		return m, func() tea.Msg { return SwitchToTableListMsg{} }

	case "esc":
		m.gPressed = false
		if m.searchInput != "" {
			// Clear search filter
			m.searchInput = ""
			m.filterData()
			return m, nil
		}
		return m, func() tea.Msg { return SwitchToTableListMsg{} }

	case "g":
		if m.gPressed {
			// Second g - go to absolute beginning
			m.Shared.CurrentPage = 0
			m.Shared.LoadTableData()
			m.filterData()
			m.selectedRow = 0
			m.gPressed = false
		} else {
			// First g - wait for second g
			m.gPressed = true
		}
		return m, nil

	case "G":
		// Go to absolute end
		maxPage := (m.Shared.TotalRows - 1) / PageSize
		m.Shared.CurrentPage = maxPage
		m.Shared.LoadTableData()
		m.filterData()
		m.selectedRow = len(m.Shared.FilteredData) - 1
		m.gPressed = false
		return m, nil

	case "enter":
		m.gPressed = false
		if len(m.Shared.FilteredData) > 0 {
			return m, func() tea.Msg {
				return SwitchToRowDetailMsg{RowIndex: m.selectedRow}
			}
		}

	case "/":
		m.gPressed = false
		m.searching = true
		m.searchInput = ""
		return m, nil

	case "s":
		m.gPressed = false
		return m, func() tea.Msg { return SwitchToQueryMsg{} }

	case "r":
		m.gPressed = false
		if err := m.Shared.LoadTableData(); err == nil {
			m.filterData()
		}

	case "up", "k":
		m.gPressed = false
		if m.selectedRow > 0 {
			m.selectedRow--
		} else if m.Shared.CurrentPage > 0 {
			// At top of current page, go to previous page
			m.Shared.CurrentPage--
			m.Shared.LoadTableData()
			m.filterData()
			m.selectedRow = len(m.Shared.FilteredData) - 1 // Go to last row of previous page
		}

	case "down", "j":
		m.gPressed = false
		if m.selectedRow < len(m.Shared.FilteredData)-1 {
			m.selectedRow++
		} else {
			// At bottom of current page, try to go to next page
			maxPage := (m.Shared.TotalRows - 1) / PageSize
			if m.Shared.CurrentPage < maxPage {
				m.Shared.CurrentPage++
				m.Shared.LoadTableData()
				m.filterData()
				m.selectedRow = 0 // Go to first row of next page
			}
		}

	case "left", "h":
		m.gPressed = false
		if m.Shared.CurrentPage > 0 {
			m.Shared.CurrentPage--
			m.Shared.LoadTableData()
			m.selectedRow = 0
		}

	case "right", "l":
		m.gPressed = false
		maxPage := (m.Shared.TotalRows - 1) / PageSize
		if m.Shared.CurrentPage < maxPage {
			m.Shared.CurrentPage++
			m.Shared.LoadTableData()
			m.selectedRow = 0
		}

	default:
		// Any other key resets the g state
		m.gPressed = false
	}
	return m, nil
}

func (m *TableDataModel) filterData() {
	if m.searchInput == "" {
		m.Shared.FilteredData = make([][]string, len(m.Shared.TableData))
		copy(m.Shared.FilteredData, m.Shared.TableData)
	} else {
		m.Shared.FilteredData = [][]string{}
		searchLower := strings.ToLower(m.searchInput)
		for _, row := range m.Shared.TableData {
			for _, cell := range row {
				if strings.Contains(strings.ToLower(cell), searchLower) {
					m.Shared.FilteredData = append(m.Shared.FilteredData, row)
					break
				}
			}
		}
	}

	if m.selectedRow >= len(m.Shared.FilteredData) {
		m.selectedRow = 0
	}
}

func (m *TableDataModel) View() string {
	var content strings.Builder

	tableName := ""
	if m.Shared.SelectedTable < len(m.Shared.FilteredTables) {
		tableName = m.Shared.FilteredTables[m.Shared.SelectedTable]
	}

	content.WriteString(TitleStyle.Render(fmt.Sprintf("Table: %s", tableName)))
	content.WriteString("\n")

	if m.searching {
		content.WriteString("\nSearch: " + m.searchInput + "_")
		content.WriteString("\n")
	} else if m.searchInput != "" {
		content.WriteString(fmt.Sprintf("\nFiltered by: %s (%d/%d rows)",
			m.searchInput, len(m.Shared.FilteredData), len(m.Shared.TableData)))
		content.WriteString("\n")
	}

	// Show pagination info
	totalPages := (m.Shared.TotalRows-1)/PageSize + 1
	content.WriteString(fmt.Sprintf("Page %d/%d (%d total rows)\n\n",
		m.Shared.CurrentPage+1, totalPages, m.Shared.TotalRows))

	if len(m.Shared.FilteredData) == 0 {
		content.WriteString("No data found")
	} else {
		// Show column headers
		headerRow := ""
		for i, col := range m.Shared.Columns {
			if i > 0 {
				headerRow += " | "
			}
			headerRow += TruncateString(col, 15)
		}
		content.WriteString(TitleStyle.Render(headerRow))
		content.WriteString("\n")

		// Show data rows with scrolling within current page
		visibleCount := Max(1, m.Shared.Height-10)
		totalRows := len(m.Shared.FilteredData)
		startIdx := 0
		
		// If there are more rows than can fit on screen, scroll the view
		if totalRows > visibleCount && m.selectedRow >= visibleCount {
			startIdx = m.selectedRow - visibleCount + 1
			// Ensure we don't scroll past the end
			if startIdx > totalRows-visibleCount {
				startIdx = totalRows - visibleCount
			}
		}
		
		endIdx := Min(totalRows, startIdx+visibleCount)

		for i := startIdx; i < endIdx; i++ {
			row := m.Shared.FilteredData[i]
			rowStr := ""
			for j, cell := range row {
				if j > 0 {
					rowStr += " | "
				}
				rowStr += TruncateString(cell, 15)
			}

			if i == m.selectedRow {
				content.WriteString(SelectedStyle.Render("> " + rowStr))
			} else {
				content.WriteString(NormalStyle.Render("  " + rowStr))
			}
			content.WriteString("\n")
		}
	}

	content.WriteString("\n")
	if m.searching {
		content.WriteString(HelpStyle.Render("Type to search • enter/esc: finish search"))
	} else {
		content.WriteString(HelpStyle.Render("↑/↓: navigate • ←/→: page • /: search • enter: details • s: SQL • r: refresh • gg/G: first/last • q: back"))
	}

	return content.String()
}
