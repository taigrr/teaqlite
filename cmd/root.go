package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"

	"github.com/taigrr/teaqlite/internal/app"
)

var (
	dbPath string
)

var rootCmd = &cobra.Command{
	Use:   "teaqlite [database.db]",
	Short: "A TUI for SQLite databases",
	Long:  `TeaQLite is a terminal user interface for browsing and editing SQLite databases.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath = args[0]
		
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			return fmt.Errorf("database file '%s' does not exist", dbPath)
		}

		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		m := app.InitialModel(db)
		if m.Err() != nil {
			return m.Err()
		}

		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("failed to run TUI: %w", err)
		}

		return nil
	},
}

func Execute() error {
	return fang.Execute(context.Background(), rootCmd)
}

func init() {
	rootCmd.Flags().StringVarP(&dbPath, "database", "d", "", "Path to SQLite database file")
}