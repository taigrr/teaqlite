package app

import (
	"database/sql"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	_ "modernc.org/sqlite"
)

func createPaginatedUsersDB(t *testing.T, rows int) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		);
	`); err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	for i := 1; i <= rows; i++ {
		name := fmt.Sprintf("User %02d", i)
		if i == rows {
			name = "Special Match"
		}
		if _, err := db.Exec(`INSERT INTO users (name, email) VALUES (?, ?)`, name, fmt.Sprintf("user%02d@example.com", i)); err != nil {
			t.Fatalf("failed to insert row %d: %v", i, err)
		}
	}

	return db
}

func newLoadedTableDataModel(t *testing.T, db *sql.DB) *TableDataModel {
	t.Helper()

	shared := NewSharedData(db)
	if err := shared.LoadTables(); err != nil {
		t.Fatalf("LoadTables() failed: %v", err)
	}

	selectedTable := 0
	for index, tableName := range shared.FilteredTables {
		if tableName == "users" {
			selectedTable = index
			break
		}
	}
	shared.SelectedTable = selectedTable
	if err := shared.LoadTableData(); err != nil {
		t.Fatalf("LoadTableData() failed: %v", err)
	}

	return NewTableDataModel(shared)
}

func TestTableDataModelDefaults(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	shared.SelectedTable = 1
	_ = shared.LoadTableData()

	m := NewTableDataModel(shared)

	if !m.Focused() {
		t.Error("should be focused by default")
	}
	if m.selectedRow != 0 {
		t.Errorf("expected selectedRow 0, got %d", m.selectedRow)
	}
	if m.searching {
		t.Error("should not be searching by default")
	}
	if m.ID() == 0 {
		t.Error("ID should be non-zero")
	}
}

func TestTableDataModelFocusBlur(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	shared.SelectedTable = 1
	_ = shared.LoadTableData()

	m := NewTableDataModel(shared)
	m.Blur()
	if m.Focused() {
		t.Error("should not be focused after Blur()")
	}

	m.Focus()
	if !m.Focused() {
		t.Error("should be focused after Focus()")
	}
}

func TestTableDataFuzzyScore(t *testing.T) {
	db := createTestDB(t)

	m := newLoadedTableDataModel(t, db)

	tests := []struct {
		name      string
		text      string
		pattern   string
		wantZero  bool
		wantAbove int
	}{
		{"exact match", "alice@example.com", "alice@example.com", false, 999},
		{"prefix match", "alice@example.com", "alice", false, 899},
		{"contains match", "alice@example.com", "example", false, 799},
		{"fuzzy match", "special match", "sm", false, 1},
		{"no match", "alice@example.com", "xyz", true, 0},
		{"empty pattern", "alice@example.com", "", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := m.fuzzyScore(tt.text, tt.pattern)
			if tt.wantZero && score != 0 {
				t.Errorf("fuzzyScore(%q, %q) = %d, want 0", tt.text, tt.pattern, score)
			}
			if !tt.wantZero && score <= tt.wantAbove-1 {
				t.Errorf("fuzzyScore(%q, %q) = %d, want > %d", tt.text, tt.pattern, score, tt.wantAbove-1)
			}
		})
	}
}

func TestTableDataFilterData(t *testing.T) {
	db := createTestDB(t)

	m := newLoadedTableDataModel(t, db)

	m.searchInput.SetValue("")
	m.filterData()
	if len(m.Shared.FilteredData) != len(m.Shared.TableData) {
		t.Fatalf("expected all rows with empty filter, got %d", len(m.Shared.FilteredData))
	}

	m.searchInput.SetValue("bob")
	m.filterData()
	if len(m.Shared.FilteredData) != 1 || m.Shared.FilteredData[0][1] != "Bob" {
		t.Fatalf("expected Bob row, got %v", m.Shared.FilteredData)
	}

	m.searchInput.SetValue("zzzzz")
	m.filterData()
	if len(m.Shared.FilteredData) != 0 {
		t.Fatalf("expected no rows for unmatched filter, got %d", len(m.Shared.FilteredData))
	}
	if m.selectedRow != 0 {
		t.Errorf("selectedRow should reset to 0 when filtered rows shrink, got %d", m.selectedRow)
	}
}

func TestTableDataPagingPreservesFilter(t *testing.T) {
	db := createPaginatedUsersDB(t, PageSize+1)

	m := newLoadedTableDataModel(t, db)
	m.searchInput.SetValue("special")
	m.filterData()

	if got := len(m.Shared.FilteredData); got != 0 {
		t.Fatalf("expected no matches on first page, got %d", got)
	}

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	if m.Shared.CurrentPage != 1 {
		t.Fatalf("expected to move to page 1, got %d", m.Shared.CurrentPage)
	}
	if got := len(m.Shared.FilteredData); got != 1 {
		t.Fatalf("expected one filtered row on next page, got %d", got)
	}
	if m.Shared.FilteredData[0][1] != "Special Match" {
		t.Fatalf("expected Special Match row after paging, got %v", m.Shared.FilteredData[0])
	}

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	if m.Shared.CurrentPage != 0 {
		t.Fatalf("expected to move back to page 0, got %d", m.Shared.CurrentPage)
	}
	if got := len(m.Shared.FilteredData); got != 0 {
		t.Fatalf("expected filter to be reapplied on previous page, got %d matches", got)
	}
}

func TestTableDataModelView(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	shared.Width = 80
	shared.Height = 24
	_ = shared.LoadTables()
	shared.SelectedTable = 1
	_ = shared.LoadTableData()

	m := NewTableDataModel(shared)
	view := m.View()
	if !containsSubstring(view, "Table: users") {
		t.Error("view should contain selected table title")
	}
	if !containsSubstring(view, "Alice") {
		t.Error("view should contain row data")
	}

	m.searchInput.SetValue("alice")
	m.filterData()
	view = m.View()
	if !containsSubstring(view, "Filtered by: alice") {
		t.Error("view should show active filter summary")
	}
}
