# SQLite TUI

A fully-featured terminal user interface for browsing SQLite databases built with Bubble Tea v2.

## Features

- **Table Browser**: Browse all tables in your SQLite database
- **Data Viewer**: View table data with pagination (20 rows per page)
- **SQL Query Interface**: Execute custom SQL queries with parameter support
- **Navigation**: Intuitive keyboard navigation
- **Responsive Design**: Adapts to terminal size

## Usage

```bash
go run main.go <database.db>
```

Example with the included sample database:
```bash
go run main.go sample.db
```

## Keyboard Controls

### Table List Mode
- `↑/↓` or `k/j`: Navigate between tables
- `Enter`: View selected table data
- `s`: Switch to SQL query mode
- `r`: Refresh table list
- `q` or `Ctrl+C`: Quit

### Table Data Mode
- `←/→` or `h/l`: Navigate between pages
- `Esc`: Return to table list
- `r`: Refresh current table data
- `q` or `Ctrl+C`: Quit

### SQL Query Mode
- Type your SQL query
- `Enter`: Execute query
- `Backspace`: Delete characters
- `Esc`: Return to table list
- `q` or `Ctrl+C`: Quit

## Features Implemented

1. **Table Browsing**: Lists all tables in the database with navigation
2. **Paginated Data View**: Shows table data with pagination (20 rows per page)
3. **SQL Query Execution**: Execute custom SQL queries and view results
4. **Error Handling**: Displays database errors gracefully
5. **Responsive UI**: Clean, styled interface that adapts to terminal size
6. **Column Information**: Shows column names and handles NULL values
7. **Navigation**: Intuitive keyboard shortcuts for all operations

## Sample Database

The included `sample.db` contains:
- `users` table with id, name, email, age columns
- `products` table with id, name, price, category columns

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver