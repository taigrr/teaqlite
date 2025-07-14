package app

import (
	"fmt"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

type EditCellModel struct {
	Shared      *SharedData
	rowIndex    int
	colIndex    int
	value       string
	cursor      int
	blinkState  bool
}

type blinkMsg struct{}

func blinkCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return blinkMsg{}
	})
}

func NewEditCellModel(shared *SharedData, rowIndex, colIndex int) *EditCellModel {
	value := ""
	if rowIndex < len(shared.FilteredData) && colIndex < len(shared.FilteredData[rowIndex]) {
		value = shared.FilteredData[rowIndex][colIndex]
	}

	return &EditCellModel{
		Shared:     shared,
		rowIndex:   rowIndex,
		colIndex:   colIndex,
		value:      value,
		cursor:     len(value),
		blinkState: true,
	}
}

func (m *EditCellModel) Init() tea.Cmd {
	return blinkCmd()
}

func (m *EditCellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case blinkMsg:
		m.blinkState = !m.blinkState
		return m, blinkCmd()
		
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return SwitchToRowDetailMsg{RowIndex: m.rowIndex} }

		case "enter":
			return m, func() tea.Msg {
				return UpdateCellMsg{
					RowIndex: m.rowIndex,
					ColIndex: m.colIndex,
					Value:    m.value,
				}
			}

		case "backspace":
			if m.cursor > 0 {
				m.value = m.value[:m.cursor-1] + m.value[m.cursor:]
				m.cursor--
			}

		case "left":
			if m.cursor > 0 {
				m.cursor--
			}

		case "right":
			if m.cursor < len(m.value) {
				m.cursor++
			}

		case "home", "ctrl+a":
			m.cursor = 0

		case "end", "ctrl+e":
			m.cursor = len(m.value)

		case "ctrl+left":
			m.cursor = m.wordLeft(m.value, m.cursor)

		case "ctrl+right":
			m.cursor = m.wordRight(m.value, m.cursor)

		case "ctrl+w":
			m.deleteWordLeft()

		default:
			if len(msg.String()) == 1 {
				m.value = m.value[:m.cursor] + msg.String() + m.value[m.cursor:]
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m *EditCellModel) View() string {
	columnName := ""
	if m.colIndex < len(m.Shared.Columns) {
		columnName = m.Shared.Columns[m.colIndex]
	}

	content := TitleStyle.Render(fmt.Sprintf("Edit Cell: %s", columnName)) + "\n\n"

	// Display value with properly positioned cursor like bubbles textinput
	content += "Value: "
	value := m.value
	pos := m.cursor
	
	// Text before cursor
	if pos > 0 {
		content += value[:pos]
	}
	
	// Cursor and character at cursor position
	if pos < len(value) {
		// Cursor over existing character
		char := string(value[pos])
		if m.blinkState {
			content += SelectedStyle.Render(char) // Highlight the character
		} else {
			content += char
		}
		// Text after cursor
		if pos+1 < len(value) {
			content += value[pos+1:]
		}
	} else {
		// Cursor at end of text
		if m.blinkState {
			content += "|"
		}
	}

	content += "\n\n"
	content += HelpStyle.Render("enter: save • esc: cancel • ctrl+w: delete word • ctrl+arrows: word nav")

	return content
}

// wordLeft finds the position of the start of the word to the left of the cursor
func (m *EditCellModel) wordLeft(text string, pos int) int {
	if pos == 0 {
		return 0
	}
	
	// Move left past any whitespace
	for pos > 0 && unicode.IsSpace(rune(text[pos-1])) {
		pos--
	}
	
	// Move left past the current word
	for pos > 0 && !unicode.IsSpace(rune(text[pos-1])) {
		pos--
	}
	
	return pos
}

// wordRight finds the position of the start of the word to the right of the cursor
func (m *EditCellModel) wordRight(text string, pos int) int {
	if pos >= len(text) {
		return len(text)
	}
	
	// Move right past the current word
	for pos < len(text) && !unicode.IsSpace(rune(text[pos])) {
		pos++
	}
	
	// Move right past any whitespace
	for pos < len(text) && unicode.IsSpace(rune(text[pos])) {
		pos++
	}
	
	return pos
}

// deleteWordLeft deletes the word to the left of the cursor
func (m *EditCellModel) deleteWordLeft() {
	if m.cursor == 0 {
		return
	}
	
	newPos := m.wordLeft(m.value, m.cursor)
	m.value = m.value[:newPos] + m.value[m.cursor:]
	m.cursor = newPos
}
