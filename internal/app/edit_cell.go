package app

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
)

type EditCellModel struct {
	Shared       *SharedData
	rowIndex     int
	colIndex     int
	input        textinput.Model
	blinkState   bool
	keyMap       EditCellKeyMap
	help         help.Model
	showFullHelp bool
	focused      bool
	id           int
}

// EditCellOption is a functional option for configuring EditCellModel
type EditCellOption func(*EditCellModel)

// WithEditCellKeyMap sets the key map
func WithEditCellKeyMap(km EditCellKeyMap) EditCellOption {
	return func(m *EditCellModel) {
		m.keyMap = km
	}
}

func blinkCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return blinkMsg{}
	})
}

func NewEditCellModel(shared *SharedData, rowIndex, colIndex int, opts ...EditCellOption) *EditCellModel {
	value := ""
	if rowIndex < len(shared.FilteredData) && colIndex < len(shared.FilteredData[rowIndex]) {
		value = shared.FilteredData[rowIndex][colIndex]
	}

	input := textinput.New()
	input.SetValue(value)
	input.Width = 50
	input.Focus()

	m := &EditCellModel{
		Shared:     shared,
		rowIndex:   rowIndex,
		colIndex:   colIndex,
		input:      input,
		blinkState: true,
		keyMap:     DefaultEditCellKeyMap(),
		help:       help.New(),
		focused:    true,
		id:         nextID(),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// ID returns the unique ID of the model
func (m EditCellModel) ID() int {
	return m.id
}

// Focus sets the focus state
func (m *EditCellModel) Focus() {
	m.focused = true
	m.input.Focus()
}

// Blur removes focus
func (m *EditCellModel) Blur() {
	m.focused = false
	m.input.Blur()
}

// Focused returns the focus state
func (m EditCellModel) Focused() bool {
	return m.focused
}

func (m *EditCellModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		blinkCmd(),
	)
}

func (m *EditCellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ToggleHelpMsg:
		m.showFullHelp = !m.showFullHelp
		return m, nil

	case blinkMsg:
		m.blinkState = !m.blinkState
		cmds = append(cmds, blinkCmd())

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Save):
			return m, func() tea.Msg {
				return UpdateCellMsg{
					RowIndex: m.rowIndex,
					ColIndex: m.colIndex,
					Value:    m.input.Value(),
				}
			}

		case key.Matches(msg, m.keyMap.Cancel):
			return m, func() tea.Msg {
				return SwitchToRowDetailMsg{RowIndex: m.rowIndex}
			}
		}
	}

	// Update the input for all other messages
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *EditCellModel) View() string {
	columnName := ""
	if m.colIndex < len(m.Shared.Columns) {
		columnName = m.Shared.Columns[m.colIndex]
	}

	content := fmt.Sprintf("%s\n\n", TitleStyle.Render(fmt.Sprintf("Edit Cell: %s", columnName)))
	content += fmt.Sprintf("Value: %s\n\n", m.input.View())
	
	if m.showFullHelp {
		content += m.help.FullHelpView(m.keyMap.FullHelp())
	} else {
		content += m.help.ShortHelpView(m.keyMap.ShortHelp())
	}

	return content
}