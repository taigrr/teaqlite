package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Table List Model
type tableListModel struct {
	shared        *sharedData
	searchInput   string
	searching     bool
	selectedTable int
	currentPage   int
}

func newTableListModel(shared *sharedData) *tableListModel {
	return &tableListModel{
		shared:        shared,
		selectedTable: 0,
		currentPage:   0,
	}
}

func (m *tableListModel) Init() tea.Cmd {
	return nil
}

func (m *tableListModel) Update(msg tea.Msg) (subModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searching {
			return m.handleSearchInput(msg)
		}
		return m.handleNavigation(msg)
	}
	return m, nil
}

func (m *tableListModel) handleSearchInput(msg tea.KeyMsg) (subModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
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

func (m *tableListModel) handleNavigation(msg tea.KeyMsg) (subModel, tea.Cmd) {
	switch msg.String() {
	case "/":
		m.searching = true
		m.searchInput = ""
		return m, nil

	case "enter":
		if len(m.shared.filteredTables) > 0 {
			return m, func() tea.Msg {
				return switchToTableDataMsg{tableIndex: m.selectedTable}
			}
		}

	case "s":
		return m, func() tea.Msg { return switchToQueryMsg{} }

	case "r":
		if err := m.shared.loadTables(); err == nil {
			m.filterTables()
		}

	case "up", "k":
		if m.selectedTable > 0 {
			m.selectedTable--
			m.adjustPage()
		}

	case "down", "j":
		if m.selectedTable < len(m.shared.filteredTables)-1 {
			m.selectedTable++
			m.adjustPage()
		}

	case "left", "h":
		if m.currentPage > 0 {
			m.currentPage--
			m.selectedTable = m.currentPage * m.getVisibleCount()
		}

	case "right", "l":
		maxPage := (len(m.shared.filteredTables) - 1) / m.getVisibleCount()
		if m.currentPage < maxPage {
			m.currentPage++
			m.selectedTable = m.currentPage * m.getVisibleCount()
			if m.selectedTable >= len(m.shared.filteredTables) {
				m.selectedTable = len(m.shared.filteredTables) - 1
			}
		}
	}
	return m, nil
}

func (m *tableListModel) filterTables() {
	if m.searchInput == "" {
		m.shared.filteredTables = make([]string, len(m.shared.tables))
		copy(m.shared.filteredTables, m.shared.tables)
	} else {
		m.shared.filteredTables = []string{}
		searchLower := strings.ToLower(m.searchInput)
		for _, table := range m.shared.tables {
			if strings.Contains(strings.ToLower(table), searchLower) {
				m.shared.filteredTables = append(m.shared.filteredTables, table)
			}
		}
	}
	
	if m.selectedTable >= len(m.shared.filteredTables) {
		m.selectedTable = 0
		m.currentPage = 0
	}
}

func (m *tableListModel) getVisibleCount() int {
	reservedLines := 8
	if m.searching {
		reservedLines += 2
	}
	return max(1, m.shared.height-reservedLines)
}

func (m *tableListModel) adjustPage() {
	visibleCount := m.getVisibleCount()
	m.currentPage = m.selectedTable / visibleCount
}

func (m *tableListModel) View() string {
	var content strings.Builder

	content.WriteString(titleStyle.Render("SQLite TUI - Tables"))
	content.WriteString("\n")
	
	if m.searching {
		content.WriteString("\nSearch: " + m.searchInput + "_")
		content.WriteString("\n")
	} else if m.searchInput != "" {
		content.WriteString(fmt.Sprintf("\nFiltered by: %s (%d/%d tables)", 
			m.searchInput, len(m.shared.filteredTables), len(m.shared.tables)))
		content.WriteString("\n")
	}
	content.WriteString("\n")

	if len(m.shared.filteredTables) == 0 {
		if m.searchInput != "" {
			content.WriteString("No tables match your search")
		} else {
			content.WriteString("No tables found in database")
		}
	} else {
		visibleCount := m.getVisibleCount()
		startIdx := m.currentPage * visibleCount
		endIdx := min(startIdx+visibleCount, len(m.shared.filteredTables))
		
		for i := startIdx; i < endIdx; i++ {
			table := m.shared.filteredTables[i]
			if i == m.selectedTable {
				content.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", table)))
			} else {
				content.WriteString(normalStyle.Render(fmt.Sprintf("  %s", table)))
			}
			content.WriteString("\n")
		}
		
		if len(m.shared.filteredTables) > visibleCount {
			totalPages := (len(m.shared.filteredTables) - 1) / visibleCount + 1
			content.WriteString(fmt.Sprintf("\nPage %d/%d", m.currentPage+1, totalPages))
		}
	}

	content.WriteString("\n")
	if m.searching {
		content.WriteString(helpStyle.Render("Type to search • enter/esc: finish search"))
	} else {
		content.WriteString(helpStyle.Render("↑/↓: navigate • ←/→: page • /: search • enter: view • s: SQL • r: refresh • q: quit"))
	}

	return content.String()
}