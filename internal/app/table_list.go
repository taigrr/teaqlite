package app

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type TableListModel struct {
	Shared        *SharedData
	searchInput   string
	searching     bool
	selectedTable int
	currentPage   int
	gPressed      bool
}

func NewTableListModel(shared *SharedData) *TableListModel {
	return &TableListModel{
		Shared:        shared,
		selectedTable: 0,
		currentPage:   0,
	}
}

func (m *TableListModel) Init() tea.Cmd {
	return nil
}

func (m *TableListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searching {
			return m.handleSearchInput(msg)
		}
		return m.handleNavigation(msg)
	}
	return m, nil
}

func (m *TableListModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		// If there's an existing filter, clear it
		if m.searchInput != "" {
			m.searchInput = ""
			m.filterTables()
		}
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

func (m *TableListModel) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.searchInput != "" {
			// Clear search filter
			m.searchInput = ""
			m.filterTables()
			m.gPressed = false
			return m, nil
		}
		// If no filter, escape does nothing (could exit app but that's handled at higher level)
		m.gPressed = false
		return m, nil

	case "/":
		m.searching = true
		m.searchInput = ""
		m.gPressed = false
		return m, nil

	case "g":
		if m.gPressed {
			// Second g - go to beginning
			m.selectedTable = 0
			m.currentPage = 0
			m.gPressed = false
		} else {
			// First g - wait for second g
			m.gPressed = true
		}
		return m, nil

	case "G":
		// Go to end
		if len(m.Shared.FilteredTables) > 0 {
			m.selectedTable = len(m.Shared.FilteredTables) - 1
			m.adjustPage()
		}
		m.gPressed = false
		return m, nil

	case "enter":
		m.gPressed = false
		if len(m.Shared.FilteredTables) > 0 {
			return m, func() tea.Msg {
				return SwitchToTableDataMsg{TableIndex: m.selectedTable}
			}
		}

	case "s":
		m.gPressed = false
		return m, func() tea.Msg { return SwitchToQueryMsg{} }

	case "r":
		m.gPressed = false
		if err := m.Shared.LoadTables(); err == nil {
			m.filterTables()
		}

	case "up", "k":
		m.gPressed = false
		if m.selectedTable > 0 {
			m.selectedTable--
			m.adjustPage()
		}

	case "down", "j":
		m.gPressed = false
		if m.selectedTable < len(m.Shared.FilteredTables)-1 {
			m.selectedTable++
			m.adjustPage()
		}

	case "left", "h":
		m.gPressed = false
		if m.currentPage > 0 {
			m.currentPage--
			m.selectedTable = m.currentPage * m.getVisibleCount()
		}

	case "right", "l":
		m.gPressed = false
		maxPage := (len(m.Shared.FilteredTables) - 1) / m.getVisibleCount()
		if m.currentPage < maxPage {
			m.currentPage++
			m.selectedTable = m.currentPage * m.getVisibleCount()
			if m.selectedTable >= len(m.Shared.FilteredTables) {
				m.selectedTable = len(m.Shared.FilteredTables) - 1
			}
		}

	default:
		// Any other key resets the g state
		m.gPressed = false
	}
	return m, nil
}

func (m *TableListModel) filterTables() {
	if m.searchInput == "" {
		m.Shared.FilteredTables = make([]string, len(m.Shared.Tables))
		copy(m.Shared.FilteredTables, m.Shared.Tables)
	} else {
		// Fuzzy search with scoring
		type tableMatch struct {
			name  string
			score int
		}
		
		var matches []tableMatch
		searchLower := strings.ToLower(m.searchInput)
		
		for _, table := range m.Shared.Tables {
			score := m.fuzzyScore(strings.ToLower(table), searchLower)
			if score > 0 {
				matches = append(matches, tableMatch{name: table, score: score})
			}
		}
		
		// Sort by score (highest first)
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].score > matches[j].score
		})
		
		// Extract sorted table names
		m.Shared.FilteredTables = make([]string, len(matches))
		for i, match := range matches {
			m.Shared.FilteredTables[i] = match.name
		}
	}

	if m.selectedTable >= len(m.Shared.FilteredTables) {
		m.selectedTable = 0
		m.currentPage = 0
	}
}

// fuzzyScore calculates a fuzzy match score between text and pattern
// Returns 0 for no match, higher scores for better matches
func (m *TableListModel) fuzzyScore(text, pattern string) int {
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
			if textIdx == 0 || text[textIdx-1] == '_' || text[textIdx-1] == '-' {
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

func (m *TableListModel) getVisibleCount() int {
	reservedLines := 8
	if m.searching {
		reservedLines += 2
	}
	return Max(1, m.Shared.Height-reservedLines)
}

func (m *TableListModel) adjustPage() {
	visibleCount := m.getVisibleCount()
	m.currentPage = m.selectedTable / visibleCount
}

func (m *TableListModel) View() string {
	var content strings.Builder

	content.WriteString(TitleStyle.Render("SQLite TUI - Tables"))
	content.WriteString("\n")

	if m.searching {
		content.WriteString("\nSearch: " + m.searchInput + "_")
		content.WriteString("\n")
	} else if m.searchInput != "" {
		content.WriteString(fmt.Sprintf("\nFiltered by: %s (%d/%d tables)",
			m.searchInput, len(m.Shared.FilteredTables), len(m.Shared.Tables)))
		content.WriteString("\n")
	}
	content.WriteString("\n")

	if len(m.Shared.FilteredTables) == 0 {
		if m.searchInput != "" {
			content.WriteString("No tables match your search")
		} else {
			content.WriteString("No tables found in database")
		}
	} else {
		visibleCount := m.getVisibleCount()
		startIdx := m.currentPage * visibleCount
		endIdx := Min(startIdx+visibleCount, len(m.Shared.FilteredTables))

		for i := startIdx; i < endIdx; i++ {
			table := m.Shared.FilteredTables[i]
			if i == m.selectedTable {
				content.WriteString(SelectedStyle.Render(fmt.Sprintf("> %s", table)))
			} else {
				content.WriteString(NormalStyle.Render(fmt.Sprintf("  %s", table)))
			}
			content.WriteString("\n")
		}

		if len(m.Shared.FilteredTables) > visibleCount {
			totalPages := (len(m.Shared.FilteredTables)-1)/visibleCount + 1
			content.WriteString(fmt.Sprintf("\nPage %d/%d", m.currentPage+1, totalPages))
		}
	}

	content.WriteString("\n")
	if m.searching {
		content.WriteString(HelpStyle.Render("Type to search • enter/esc: finish search"))
	} else {
		content.WriteString(HelpStyle.Render("↑/↓: navigate • ←/→: page • /: search • enter: view • s: SQL • r: refresh • gg/G: first/last • ctrl+c: quit"))
	}

	return content.String()
}
