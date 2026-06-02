package app

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"empty string", "", 5, ""},
		{"min truncation", "abcdef", 4, "a..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
		want  []string
	}{
		{"short text", "hello", 20, []string{"hello"}},
		{"wrap at word boundary", "hello world foo", 12, []string{"hello world", "foo"}},
		{"zero width", "hello", 0, []string{"hello"}},
		{"empty text", "", 10, []string{""}},
		{"single long word", "abcdefghij", 5, []string{"abcde", "fghij"}},
		{"multiple words wrapping", "one two three four", 10, []string{"one two", "three four"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WrapText(tt.text, tt.width)
			if len(got) != len(tt.want) {
				t.Errorf("WrapText(%q, %d) returned %d lines, want %d: %v", tt.text, tt.width, len(got), len(tt.want), got)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("WrapText(%q, %d)[%d] = %q, want %q", tt.text, tt.width, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNextID(t *testing.T) {
	// nextID should return monotonically increasing values
	id1 := nextID()
	id2 := nextID()
	id3 := nextID()

	if id2 <= id1 {
		t.Errorf("nextID() not monotonically increasing: %d, %d", id1, id2)
	}
	if id3 <= id2 {
		t.Errorf("nextID() not monotonically increasing: %d, %d", id2, id3)
	}
}

func TestDefaultAppKeyMap(t *testing.T) {
	km := DefaultAppKeyMap()

	if len(km.Quit.Keys()) == 0 {
		t.Error("Quit keybinding has no keys")
	}
	if len(km.Suspend.Keys()) == 0 {
		t.Error("Suspend keybinding has no keys")
	}
	if len(km.ToggleHelp.Keys()) == 0 {
		t.Error("ToggleHelp keybinding has no keys")
	}
}

// createTestDB creates an in-memory SQLite database with test data
func createTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		);
		INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com');
		INSERT INTO users (name, email) VALUES ('Bob', 'bob@example.com');
		INSERT INTO users (name, email) VALUES ('Charlie', 'charlie@example.com');

		CREATE TABLE products (
			id INTEGER PRIMARY KEY,
			title TEXT NOT NULL,
			price REAL
		);
		INSERT INTO products (title, price) VALUES ('Widget', 9.99);
		INSERT INTO products (title, price) VALUES ('Gadget', 19.99);
	`)
	if err != nil {
		t.Fatalf("failed to create test data: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close test database: %v", err)
		}
	})

	return db
}

func TestNewSharedData(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	if shared.DB != db {
		t.Error("NewSharedData should store the database reference")
	}
	if shared.Width != 80 || shared.Height != 24 {
		t.Errorf("default dimensions should be 80x24, got %dx%d", shared.Width, shared.Height)
	}
}

func TestSharedDataLoadTables(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	if err := shared.LoadTables(); err != nil {
		t.Fatalf("LoadTables() failed: %v", err)
	}

	if len(shared.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d: %v", len(shared.Tables), shared.Tables)
	}

	// Tables should be sorted alphabetically
	if shared.Tables[0] != "products" || shared.Tables[1] != "users" {
		t.Errorf("expected [products, users], got %v", shared.Tables)
	}

	// FilteredTables should be a copy of Tables
	if len(shared.FilteredTables) != len(shared.Tables) {
		t.Errorf("FilteredTables length mismatch: %d vs %d", len(shared.FilteredTables), len(shared.Tables))
	}
}

func TestSharedDataLoadTableData(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	if err := shared.LoadTables(); err != nil {
		t.Fatalf("LoadTables() failed: %v", err)
	}

	// Load "users" table (index 1 since sorted alphabetically)
	shared.SelectedTable = 1
	if err := shared.LoadTableData(); err != nil {
		t.Fatalf("LoadTableData() failed: %v", err)
	}

	if len(shared.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d: %v", len(shared.Columns), shared.Columns)
	}
	if shared.Columns[0] != "id" || shared.Columns[1] != "name" || shared.Columns[2] != "email" {
		t.Errorf("unexpected columns: %v", shared.Columns)
	}

	if shared.TotalRows != 3 {
		t.Errorf("expected 3 total rows, got %d", shared.TotalRows)
	}

	if len(shared.TableData) != 3 {
		t.Fatalf("expected 3 data rows, got %d", len(shared.TableData))
	}

	// Check primary keys detected
	if len(shared.PrimaryKeys) != 1 || shared.PrimaryKeys[0] != "id" {
		t.Errorf("expected primary key [id], got %v", shared.PrimaryKeys)
	}

	// Should not be marked as query result
	if shared.IsQueryResult {
		t.Error("regular table load should not be marked as query result")
	}
}

func TestSharedDataLoadTableDataInvalidIndex(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	if err := shared.LoadTables(); err != nil {
		t.Fatalf("LoadTables() failed: %v", err)
	}

	shared.SelectedTable = 99
	if err := shared.LoadTableData(); err == nil {
		t.Error("LoadTableData() should fail with invalid table index")
	}
}

func TestSharedDataUpdateCell(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	if err := shared.LoadTables(); err != nil {
		t.Fatalf("LoadTables() failed: %v", err)
	}

	// Load users table
	shared.SelectedTable = 1
	if err := shared.LoadTableData(); err != nil {
		t.Fatalf("LoadTableData() failed: %v", err)
	}

	// Update name of first user (row 0, col 1 = name)
	if err := shared.UpdateCell(0, 1, "Alicia"); err != nil {
		t.Fatalf("UpdateCell() failed: %v", err)
	}

	// Verify local data updated
	if shared.FilteredData[0][1] != "Alicia" {
		t.Errorf("FilteredData not updated, got %q", shared.FilteredData[0][1])
	}

	// Verify database updated
	var name string
	err := db.QueryRow("SELECT name FROM users WHERE id = 1").Scan(&name)
	if err != nil {
		t.Fatalf("failed to verify update: %v", err)
	}
	if name != "Alicia" {
		t.Errorf("database not updated, got %q", name)
	}
}

func TestSharedDataUpdateCellInvalidIndex(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	if err := shared.LoadTables(); err != nil {
		t.Fatalf("LoadTables() failed: %v", err)
	}
	shared.SelectedTable = 1
	if err := shared.LoadTableData(); err != nil {
		t.Fatalf("LoadTableData() failed: %v", err)
	}

	if err := shared.UpdateCell(99, 0, "value"); err == nil {
		t.Error("UpdateCell() should fail with invalid row index")
	}
	if err := shared.UpdateCell(0, 99, "value"); err == nil {
		t.Error("UpdateCell() should fail with invalid column index")
	}
}

func TestSharedDataPagination(t *testing.T) {
	db := createTestDB(t)

	// Insert enough rows to test pagination
	for i := 0; i < 25; i++ {
		_, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)",
			"User"+string(rune('A'+i)), "user@example.com")
		if err != nil {
			t.Fatalf("failed to insert test data: %v", err)
		}
	}

	shared := NewSharedData(db)
	if err := shared.LoadTables(); err != nil {
		t.Fatalf("LoadTables() failed: %v", err)
	}

	shared.SelectedTable = 1
	shared.CurrentPage = 0
	if err := shared.LoadTableData(); err != nil {
		t.Fatalf("LoadTableData() page 0 failed: %v", err)
	}

	if shared.TotalRows != 28 { // 3 original + 25 new
		t.Errorf("expected 28 total rows, got %d", shared.TotalRows)
	}
	if len(shared.TableData) != PageSize {
		t.Errorf("page 0 should have %d rows, got %d", PageSize, len(shared.TableData))
	}

	// Load page 2
	shared.CurrentPage = 1
	if err := shared.LoadTableData(); err != nil {
		t.Fatalf("LoadTableData() page 1 failed: %v", err)
	}
	if len(shared.TableData) != 8 { // 28 - 20 = 8
		t.Errorf("page 1 should have 8 rows, got %d", len(shared.TableData))
	}
}

func TestInitialModel(t *testing.T) {
	db := createTestDB(t)

	m := InitialModel(db)
	if m.Err() != nil {
		t.Fatalf("InitialModel() returned error: %v", m.Err())
	}
	if m.width != 80 || m.height != 24 {
		t.Errorf("default dimensions should be 80x24, got %dx%d", m.width, m.height)
	}
	if !m.Focused() {
		t.Error("model should be focused by default")
	}
}

func TestInitialModelWithOptions(t *testing.T) {
	db := createTestDB(t)

	km := DefaultAppKeyMap()
	m := InitialModel(db, WithKeyMap(km), WithDimensions(120, 40))
	if m.Err() != nil {
		t.Fatalf("InitialModel() returned error: %v", m.Err())
	}
	if m.width != 120 || m.height != 40 {
		t.Errorf("custom dimensions should be 120x40, got %dx%d", m.width, m.height)
	}
}

func TestModelFocusBlur(t *testing.T) {
	db := createTestDB(t)

	m := InitialModel(db)
	if !m.Focused() {
		t.Error("model should be focused initially")
	}

	m.Blur()
	if m.Focused() {
		t.Error("model should not be focused after Blur()")
	}

	m.Focus()
	if !m.Focused() {
		t.Error("model should be focused after Focus()")
	}
}

func TestInferTableFromQueryResult(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	if err := shared.LoadTables(); err != nil {
		t.Fatalf("LoadTables() failed: %v", err)
	}

	// Set up columns matching the users table
	shared.Columns = []string{"id", "name", "email"}
	shared.IsQueryResult = true
	shared.FilteredData = [][]string{{"1", "Alice", "alice@example.com"}}

	tableName, err := shared.inferTableFromQueryResult(0, 0)
	if err != nil {
		t.Fatalf("inferTableFromQueryResult() failed: %v", err)
	}
	if tableName != "users" {
		t.Errorf("expected 'users', got %q", tableName)
	}

	// Verify it was cached
	if shared.QueryTableName != "users" {
		t.Errorf("QueryTableName should be cached as 'users', got %q", shared.QueryTableName)
	}
}

func TestGetTableInfo(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	cols, pks, err := shared.getTableInfo("users")
	if err != nil {
		t.Fatalf("getTableInfo() failed: %v", err)
	}

	if len(cols) != 3 {
		t.Errorf("expected 3 columns, got %d", len(cols))
	}
	if len(pks) != 1 || pks[0] != "id" {
		t.Errorf("expected primary key [id], got %v", pks)
	}
}

func TestSharedDataEmptyDatabase(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	shared := NewSharedData(db)
	if err := shared.LoadTables(); err != nil {
		t.Fatalf("LoadTables() on empty db failed: %v", err)
	}

	if len(shared.Tables) != 0 {
		t.Errorf("expected 0 tables in empty db, got %d", len(shared.Tables))
	}
}
