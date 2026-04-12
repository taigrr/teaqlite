package app

import (
	"testing"

	_ "modernc.org/sqlite"
)

func TestEditCellModelDefaults(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	shared.SelectedTable = 1 // users
	_ = shared.LoadTableData()

	m := NewEditCellModel(shared, 0, 1) // row 0, col 1 (name)

	if !m.Focused() {
		t.Error("should be focused by default")
	}
	if m.ID() == 0 {
		t.Error("ID should be non-zero")
	}
	if m.rowIndex != 0 {
		t.Errorf("expected rowIndex 0, got %d", m.rowIndex)
	}
	if m.colIndex != 1 {
		t.Errorf("expected colIndex 1, got %d", m.colIndex)
	}
}

func TestEditCellModelFocusBlur(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	shared.SelectedTable = 1
	_ = shared.LoadTableData()

	m := NewEditCellModel(shared, 0, 1)

	m.Blur()
	if m.Focused() {
		t.Error("should not be focused after Blur()")
	}

	m.Focus()
	if !m.Focused() {
		t.Error("should be focused after Focus()")
	}
}

func TestEditCellModelWithOptions(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	shared.SelectedTable = 1
	_ = shared.LoadTableData()

	km := DefaultEditCellKeyMap()
	m := NewEditCellModel(shared, 0, 1, WithEditCellKeyMap(km))

	if !m.Focused() {
		t.Error("should be focused")
	}
}

func TestEditCellModelView(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	shared := NewSharedData(db)
	shared.Width = 80
	shared.Height = 24
	_ = shared.LoadTables()
	shared.SelectedTable = 1
	_ = shared.LoadTableData()

	m := NewEditCellModel(shared, 0, 1) // editing "name" column

	view := m.View()
	if !containsSubstring(view, "Edit Cell") {
		t.Error("view should contain title")
	}
	if !containsSubstring(view, "name") {
		t.Error("view should show column name")
	}
}

func TestEditCellModelOutOfBounds(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	shared.SelectedTable = 1
	_ = shared.LoadTableData()

	// Out of bounds row/col should not panic
	m := NewEditCellModel(shared, 99, 99)
	if m == nil {
		t.Error("should create model even with out of bounds indices")
	}
}
