package app

import (
	"slices"
	"testing"
)

func TestCompositePrimaryKey(t *testing.T) {
	for _, queryResult := range []bool{false, true} {
		name := "table"
		if queryResult {
			name = "query"
		}
		t.Run(name, func(t *testing.T) {
			db := createTestDB(t)
			_, err := db.Exec(`CREATE TABLE entries (tenant TEXT, item TEXT, value TEXT, PRIMARY KEY (item, tenant));
    INSERT INTO entries VALUES ('one', 'shared', 'first'), ('two', 'shared', 'second')`)
			if err != nil {
				t.Fatal(err)
			}
			shared := NewSharedData(db)
			shared.FilteredTables = []string{"entries"}
			if err := shared.LoadTableData(); err != nil {
				t.Fatal(err)
			}
			wantKeys := []string{"tenant", "item"}
			if !slices.Equal(shared.PrimaryKeys, wantKeys) {
				t.Errorf("loaded keys = %v, want %v", shared.PrimaryKeys, wantKeys)
			}
			_, keys, err := shared.getTableInfo("entries")
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(keys, wantKeys) {
				t.Errorf("table keys = %v, want %v", keys, wantKeys)
			}
			model := NewQueryModel(shared)
			if keys := model.getTablePrimaryKeys("entries"); !slices.Equal(keys, wantKeys) {
				t.Errorf("query keys = %v, want %v", keys, wantKeys)
			}
			shared.IsQueryResult = queryResult
			shared.QueryTableName = "entries"
			// Find the target independently of SQLite's row ordering.
			target := -1
			for index, row := range shared.FilteredData {
				if row[0] == "one" {
					target = index
				}
			}
			if target < 0 {
				t.Fatal("target row missing")
			}
			if err := shared.UpdateCell(target, 2, "changed"); err != nil {
				t.Fatal(err)
			}
			for tenant, want := range map[string]string{"one": "changed", "two": "second"} {
				var got string
				if err := db.QueryRow(`SELECT value FROM entries WHERE tenant = ?`, tenant).Scan(&got); err != nil {
					t.Fatal(err)
				}
				if got != want {
					t.Errorf("tenant %s value = %q, want %q", tenant, got, want)
				}
			}
		})
	}
}
