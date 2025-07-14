package app

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
)

type TableListModel struct {
	Shared        *SharedData
	searchInput   textinput.Model
	searching     bool
	selectedTable int
	currentPage   int
	gPressed      bool
	keyMap        TableListKeyMap
	help          help.Model
	showFullHelp  bool
	focused       bool
	id            int
}

// TableListOption is a functional option for configuring TableListModel
type TableListOption func(*TableListModel)

// WithTableListKeyMap sets the key map
func WithTableListKeyMap(km TableListKeyMap) TableListOption {
	return func(m *TableListModel) {
		m.keyMap = km
	}
}

func NewTableListModel(shared *SharedData, opts ...TableListOption) *TableListModel {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search tables..."
	searchInput.CharLimit = 50
	searchInput.Width = 30

	m := &TableListModel{
		Shared:        shared,
		searchInput:   searchInput,
		selectedTable: 0,
		currentPage:   0,
		keyMap:        DefaultTableListKeyMap(),
		help:          help.New(),
		focused:       true,
		id:            nextID(),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// ID returns the unique ID of the model
func (m TableListModel) ID() int {
	return m.id
}

// Focus sets the focus state
func (m *TableListModel) Focus() {
	m.focused = true
	if m.searching {
		m.searchInput.Focus()
	}
}

// Blur removes focus
func (m *TableListModel) Blur() {
	m.focused = false
	m.searchInput.Blur()
}

// Focused returns the focus state
func (m TableListModel) Focused() bool {
	return m.focused
}

func (m *TableListModel) Init() tea.Cmd {
	return nil
}

func (m *TableListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ToggleHelpMsg:
		m.showFullHelp = !m.showFullHelp
		return m, nil

	case tea.KeyMsg:
		if m.searching {
			return m.handleSearchInput(msg)
		}
		return m.handleNavigation(msg)
	}

	// Update search input for non-key messages when searching
	if m.searching {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Update filter when search input changes
		m.filterTables()
	}

	return m, tea.Batch(cmds...)
}

func (m *TableListModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Escape):
		m.searching = false
		m.searchInput.Blur()
		// If there's an existing filter, clear it
		if m.searchInput.Value() != "" {
			m.searchInput.SetValue("")
			m.filterTables()
		}
	case key.Matches(msg, m.keyMap.Enter):
		m.searching = false
		m.searchInput.Blur()
		m.filterTables()
	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.filterTables()
		return m, cmd
	}
	return m, nil
}

func (m *TableListModel) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Escape):
		if m.searchInput.Value() != "" {
			// Clear search filter
			m.searchInput.SetValue("")
			m.filterTables()
			m.gPressed = false
			return m, nil
		}
		// If no filter, escape does nothing (could exit app but that's handled at higher level)
		m.gPressed = false
		return m, nil

	case key.Matches(msg, m.keyMap.Search):
		m.searching = true
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		m.gPressed = false
		return m, nil

	case key.Matches(msg, m.keyMap.GoToStart):
		if m.gPressed {
			// Second g - go to beginning (gg pattern like vim)
			m.selectedTable = 0
			m.currentPage = 0
			m.gPressed = false
		} else {
			// First g - wait for second g to complete gg sequence
			m.gPressed = true
		}
		return m, nil

	case key.Matches(msg, m.keyMap.GoToEnd):
		// Go to end (G pattern like vim)
		if len(m.Shared.FilteredTables) > 0 {
			m.selectedTable = len(m.Shared.FilteredTables) - 1
			m.adjustPage()
		}
		m.gPressed = false
		return m, nil

	case key.Matches(msg, m.keyMap.Enter):
		m.gPressed = false
		if len(m.Shared.FilteredTables) > 0 {
			return m, func() tea.Msg {
				return SwitchToTableDataMsg{TableIndex: m.selectedTable}
			}
		}

	case key.Matches(msg, m.keyMap.SQLMode):
		m.gPressed = false
		return m, func() tea.Msg { return SwitchToQueryMsg{} }

	case key.Matches(msg, m.keyMap.Refresh):
		m.gPressed = false
		if err := m.Shared.LoadTables(); err == nil {
			m.filterTables()
		}

	case key.Matches(msg, m.keyMap.Up):
		m.gPressed = false
		if m.selectedTable > 0 {
			m.selectedTable--
			m.adjustPage()
		}

	case key.Matches(msg, m.keyMap.Down):
		m.gPressed = false
		if m.selectedTable < len(m.Shared.FilteredTables)-1 {
			m.selectedTable++
			m.adjustPage()
		}

	case key.Matches(msg, m.keyMap.Left):
		m.gPressed = false
		if m.currentPage > 0 {
			m.currentPage--
			m.selectedTable = m.currentPage * m.getVisibleCount()
		}

	case key.Matches(msg, m.keyMap.Right):
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
	searchValue := m.searchInput.Value()
	if searchValue == "" {
		m.Shared.FilteredTables = make([]string, len(m.Shared.Tables))
		copy(m.Shared.FilteredTables, m.Shared.Tables)
	} else {
		// Fuzzy search with scoring
		type tableMatch struct {
			name  string
			score int
		}
		
		var matches []tableMatch
		searchLower := strings.ToLower(searchValue)
		
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
		content.WriteString("\nSearch: " + m.searchInput.View())
		content.WriteString("\n")
	} else if m.searchInput.Value() != "" {
		content.WriteString(fmt.Sprintf("\nFiltered by: %s (%d/%d tables)",
			m.searchInput.Value(), len(m.Shared.FilteredTables), len(m.Shared.Tables)))
		content.WriteString("\n")
	}
	content.WriteString("\n")

	if len(m.Shared.FilteredTables) == 0 {
		if m.searchInput.Value() != "" {
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
		if m.showFullHelp {
			content.WriteString(m.help.FullHelpView(m.keyMap.FullHelp()))
		} else {
			content.WriteString(m.help.ShortHelpView(m.keyMap.ShortHelp()))
		}
	}

	return content.String()
}