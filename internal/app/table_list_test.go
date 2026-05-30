package app

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestTableListModelDefaults(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	m := NewTableListModel(shared)

	if !m.Focused() {
		t.Error("should be focused by default")
	}
	if m.selectedTable != 0 {
		t.Errorf("expected selectedTable 0, got %d", m.selectedTable)
	}
	if m.searching {
		t.Error("should not be searching by default")
	}
	if m.ID() == 0 {
		t.Error("ID should be non-zero")
	}
}

func TestTableListModelFocusBlur(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	m := NewTableListModel(shared)

	m.Blur()
	if m.Focused() {
		t.Error("should not be focused after Blur()")
	}

	m.Focus()
	if !m.Focused() {
		t.Error("should be focused after Focus()")
	}
}

func TestTableListModelWithOptions(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	km := DefaultTableListKeyMap()
	m := NewTableListModel(shared, WithTableListKeyMap(km))

	if !m.Focused() {
		t.Error("should be focused")
	}
}

func TestFuzzyScore(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	m := NewTableListModel(shared)

	tests := []struct {
		name      string
		text      string
		pattern   string
		wantZero  bool
		wantAbove int // minimum expected score
	}{
		{"exact match", "users", "users", false, 999},
		{"prefix match", "users", "use", false, 899},
		{"contains match", "all_users", "user", false, 799},
		{"fuzzy match", "user_table", "ut", false, 1},
		{"no match", "users", "xyz", true, 0},
		{"empty pattern", "users", "", false, 0},
		{"pattern longer than text", "ab", "abcd", true, 0},
		{"word boundary match", "user_name", "un", false, 1},
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

	// Exact match should score higher than prefix match
	exactScore := m.fuzzyScore("users", "users")
	prefixScore := m.fuzzyScore("users_table", "users")
	if exactScore <= prefixScore {
		t.Errorf("exact match (%d) should score higher than prefix match (%d)", exactScore, prefixScore)
	}
}

func TestFilterTables(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	m := NewTableListModel(shared)

	// Empty search returns all tables
	m.searchInput.SetValue("")
	m.filterTables()
	if len(m.Shared.FilteredTables) != 2 {
		t.Errorf("expected 2 tables with empty filter, got %d", len(m.Shared.FilteredTables))
	}

	// Filter for "user" should match "users"
	m.searchInput.SetValue("user")
	m.filterTables()
	if len(m.Shared.FilteredTables) != 1 || m.Shared.FilteredTables[0] != "users" {
		t.Errorf("expected [users], got %v", m.Shared.FilteredTables)
	}

	// Filter for "prod" should match "products"
	m.searchInput.SetValue("prod")
	m.filterTables()
	if len(m.Shared.FilteredTables) != 1 || m.Shared.FilteredTables[0] != "products" {
		t.Errorf("expected [products], got %v", m.Shared.FilteredTables)
	}

	// Filter with no match
	m.searchInput.SetValue("zzzzz")
	m.filterTables()
	if len(m.Shared.FilteredTables) != 0 {
		t.Errorf("expected 0 tables, got %d", len(m.Shared.FilteredTables))
	}

	// Selected table resets when filter has no results
	if m.selectedTable != 0 {
		t.Errorf("selectedTable should reset to 0 when no results, got %d", m.selectedTable)
	}
}

func TestGetVisibleCount(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	shared.Height = 24
	_ = shared.LoadTables()
	m := NewTableListModel(shared)

	count := m.getVisibleCount()
	if count != 16 { // 24 - 8 reserved lines
		t.Errorf("expected 16 visible count, got %d", count)
	}

	// While searching, fewer visible
	m.searching = true
	count = m.getVisibleCount()
	if count != 14 { // 24 - 10 reserved lines
		t.Errorf("expected 14 visible count while searching, got %d", count)
	}

	// Very small terminal
	shared.Height = 5
	m.searching = false
	count = m.getVisibleCount()
	if count < 1 {
		t.Error("visible count should be at least 1")
	}
}

func TestAdjustPage(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	shared.Height = 24
	_ = shared.LoadTables()
	m := NewTableListModel(shared)

	m.selectedTable = 0
	m.adjustPage()
	if m.currentPage != 0 {
		t.Errorf("expected page 0, got %d", m.currentPage)
	}
}

func TestTableListModelView(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	shared.Width = 80
	shared.Height = 24
	_ = shared.LoadTables()
	m := NewTableListModel(shared)

	view := m.View()
	if !containsSubstring(view, "SQLite TUI") {
		t.Error("view should contain title")
	}
	if !containsSubstring(view, "products") {
		t.Error("view should contain table names")
	}
	if !containsSubstring(view, "users") {
		t.Error("view should contain table names")
	}

	// View while searching
	m.searching = true
	view = m.View()
	if !containsSubstring(view, "Search:") {
		t.Error("view should show search input while searching")
	}

	// View with filter applied
	m.searching = false
	m.searchInput.SetValue("user")
	m.filterTables()
	view = m.View()
	if !containsSubstring(view, "Filtered by:") {
		t.Error("view should show filter info")
	}
}

func TestTableListModelViewEmptyDB(t *testing.T) {
	db, err := openTestDB(t)
	if err != nil {
		t.Fatal(err)
	}

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	m := NewTableListModel(shared)

	view := m.View()
	if !containsSubstring(view, "No tables found") {
		t.Error("empty db view should show 'No tables found'")
	}
}

func openTestDB(t *testing.T) (*sql.DB, error) {
	t.Helper()
	return sql.Open("sqlite", ":memory:")
}
