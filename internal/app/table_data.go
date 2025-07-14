package app

import (
	"fmt"
	"sort"
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
		return m, func() tea.Msg { return SwitchToTableListClearMsg{} }

	case "esc":
		m.gPressed = false
		if m.searchInput != "" {
			// Clear search filter
			m.searchInput = ""
			m.filterData()
			return m, nil
		}
		return m, func() tea.Msg { return SwitchToTableListClearMsg{} }

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
		// Fuzzy search with scoring for rows
		type rowMatch struct {
			row   []string
			score int
		}
		
		var matches []rowMatch
		searchLower := strings.ToLower(m.searchInput)
		
		for _, row := range m.Shared.TableData {
			bestScore := 0
			// Check each cell in the row and take the best score
			for _, cell := range row {
				score := m.fuzzyScore(strings.ToLower(cell), searchLower)
				if score > bestScore {
					bestScore = score
				}
			}
			
			if bestScore > 0 {
				matches = append(matches, rowMatch{row: row, score: bestScore})
			}
		}
		
		// Sort by score (highest first)
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].score > matches[j].score
		})
		
		// Extract sorted rows
		m.Shared.FilteredData = make([][]string, len(matches))
		for i, match := range matches {
			m.Shared.FilteredData[i] = match.row
		}
	}

	if m.selectedRow >= len(m.Shared.FilteredData) {
		m.selectedRow = 0
	}
}

// fuzzyScore calculates a fuzzy match score between text and pattern
// Returns 0 for no match, higher scores for better matches
func (m *TableDataModel) fuzzyScore(text, pattern string) int {
	if pattern == "" {
		return 1
	}
	
	textLen := len(text)
	patternLen := len(pattern)
	
	if patternLen > textLen {
		return 0
	}
	
	// Exact match gets highest score
	if text == pattern {
		return 1000
	}
	
	// Prefix match gets high score
	if strings.HasPrefix(text, pattern) {
		return 900
	}
	
	// Contains match gets medium score
	if strings.Contains(text, pattern) {
		return 800
	}
	
	// Fuzzy character sequence matching
	score := 0
	textIdx := 0
	patternIdx := 0
	consecutiveMatches := 0
	
	for textIdx < textLen && patternIdx < patternLen {
		if text[textIdx] == pattern[patternIdx] {
			score += 10
			consecutiveMatches++
			
			// Bonus for consecutive matches
			if consecutiveMatches > 1 {
				score += consecutiveMatches * 5
			}
			
			// Bonus for matches at word boundaries
			if textIdx == 0 || text[textIdx-1] == '_' || text[textIdx-1] == '-' || text[textIdx-1] == ' ' {
				score += 20
			}
			
			patternIdx++
		} else {
			consecutiveMatches = 0
		}
		textIdx++
	}
	
	// Must match all pattern characters
	if patternIdx < patternLen {
		return 0
	}
	
	// Bonus for shorter text (more precise match)
	score += (100 - textLen)
	
	return score
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
