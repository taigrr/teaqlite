package app

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestExtractTableName(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	m := NewQueryModel(shared)

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{"simple select", "SELECT * FROM users", "users"},
		{"select with where", "SELECT name FROM products WHERE price > 10", "products"},
		{"case insensitive", "select * from Users", "Users"},
		{"with alias", "SELECT * FROM users u WHERE u.id = 1", "users"},
		{"quoted table", `SELECT * FROM "users"`, "users"},
		{"backtick table", "SELECT * FROM `users`", "users"},
		{"no from clause", "SELECT 1", ""},
		{"empty query", "", ""},
		{"with limit", "SELECT * FROM users LIMIT 10", "users"},
		{"with join", "SELECT * FROM users JOIN products ON users.id = products.id", "users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.extractTableName(tt.query)
			if got != tt.want {
				t.Errorf("extractTableName(%q) = %q, want %q", tt.query, got, tt.want)
			}
		})
	}
}

func TestGetTablePrimaryKeys(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	m := NewQueryModel(shared)

	pks := m.getTablePrimaryKeys("users")
	if len(pks) != 1 || pks[0] != "id" {
		t.Errorf("expected [id], got %v", pks)
	}

	pks = m.getTablePrimaryKeys("nonexistent")
	if len(pks) != 0 {
		t.Errorf("expected empty for nonexistent table, got %v", pks)
	}
}

func TestEnsureIDColumns(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	m := NewQueryModel(shared)

	tests := []struct {
		name      string
		query     string
		contains  string // expected substring in result
		unchanged bool   // if true, expect query unchanged
	}{
		{"select star unchanged", "SELECT * FROM users", "SELECT *", true},
		{"already has id", "SELECT id, name FROM users", "SELECT id", true},
		{"adds id column", "SELECT name, email FROM users", "id", false},
		{"non-select unchanged", "INSERT INTO users VALUES (1, 'test', 'test@test.com')", "INSERT", true},
		{"no table found", "SELECT 1", "SELECT 1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.ensureIDColumns(tt.query)
			if tt.unchanged && got != tt.query {
				t.Errorf("expected unchanged query, got %q", got)
			}
			if tt.contains != "" && !containsStr(got, tt.contains) {
				t.Errorf("expected result to contain %q, got %q", tt.contains, got)
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestQueryModelNewDefaults(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	m := NewQueryModel(shared)

	if !m.FocusOnInput {
		t.Error("new query model should focus on input")
	}
	if !m.Focused() {
		t.Error("new query model should be focused")
	}
	if m.selectedRow != 0 {
		t.Errorf("expected selectedRow 0, got %d", m.selectedRow)
	}
	if m.ID() == 0 {
		t.Error("ID should be non-zero")
	}
}

func TestQueryModelFocusBlur(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	m := NewQueryModel(shared)

	m.Blur()
	if m.Focused() {
		t.Error("should not be focused after Blur()")
	}

	m.Focus()
	if !m.Focused() {
		t.Error("should be focused after Focus()")
	}
}

func TestQueryModelWithOptions(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	km := DefaultQueryKeyMap()
	m := NewQueryModel(shared, WithQueryKeyMap(km))

	if !m.Focused() {
		t.Error("should be focused")
	}
}

func TestHandleQueryCompletion(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	m := NewQueryModel(shared)

	// Successful completion
	msg := QueryCompletedMsg{
		Results: [][]string{{"1", "Alice", "alice@example.com"}},
		Columns: []string{"id", "name", "email"},
		Error:   nil,
	}

	m.handleQueryCompletion(msg)

	if m.FocusOnInput {
		t.Error("should not focus on input after completion")
	}
	if len(m.results) != 1 {
		t.Errorf("expected 1 result, got %d", len(m.results))
	}
	if len(m.columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(m.columns))
	}
	if !m.Shared.IsQueryResult {
		t.Error("shared should be marked as query result")
	}

	// Error completion
	m2 := NewQueryModel(shared)
	errMsg := QueryCompletedMsg{
		Error: sql.ErrNoRows,
	}
	m2.handleQueryCompletion(errMsg)
	if m2.err == nil {
		t.Error("expected error to be set")
	}
}

func TestQueryModelView(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	shared.Width = 80
	shared.Height = 24
	m := NewQueryModel(shared)

	// Basic view with input focused
	view := m.View()
	if !containsSubstring(view, "SQL Query") {
		t.Error("view should contain title")
	}
	if !containsSubstring(view, "Query:") {
		t.Error("view should contain query label")
	}

	// View with results
	m.handleQueryCompletion(QueryCompletedMsg{
		Results: [][]string{{"1", "Alice"}, {"2", "Bob"}},
		Columns: []string{"id", "name"},
	})

	view = m.View()
	if !containsSubstring(view, "2 rows returned") {
		t.Error("view should show row count")
	}
}
