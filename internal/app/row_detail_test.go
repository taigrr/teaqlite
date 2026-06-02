package app

import (
	"testing"

	_ "modernc.org/sqlite"
)

func TestRowDetailModelDefaults(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	shared.SelectedTable = 1
	_ = shared.LoadTableData()

	m := NewRowDetailModel(shared, 0)

	if !m.Focused() {
		t.Error("should be focused by default")
	}
	if m.ID() == 0 {
		t.Error("ID should be non-zero")
	}
}

func TestRowDetailModelFocusBlur(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	shared.SelectedTable = 1
	_ = shared.LoadTableData()

	m := NewRowDetailModel(shared, 0)

	m.Blur()
	if m.Focused() {
		t.Error("should not be focused after Blur()")
	}

	m.Focus()
	if !m.Focused() {
		t.Error("should be focused after Focus()")
	}
}

func TestRowDetailModelView(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	shared.Width = 80
	shared.Height = 24
	_ = shared.LoadTables()
	shared.SelectedTable = 1
	_ = shared.LoadTableData()

	m := NewRowDetailModel(shared, 0)

	view := m.View()
	if !containsSubstring(view, "Row Detail") {
		t.Error("view should contain title")
	}
}

func TestRowDetailModelWithOptions(t *testing.T) {
	db := createTestDB(t)

	shared := NewSharedData(db)
	_ = shared.LoadTables()
	shared.SelectedTable = 1
	_ = shared.LoadTableData()

	km := DefaultRowDetailKeyMap()
	m := NewRowDetailModel(shared, 0, WithRowDetailKeyMap(km))

	if !m.Focused() {
		t.Error("should be focused")
	}
}
