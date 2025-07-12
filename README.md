# SQLite TUI

A fully-featured terminal user interface for browsing SQLite databases built with Bubble Tea v2.

## Features

- **Table Browser**: Browse all tables in your SQLite database with pagination
- **Search Functionality**: Search tables by name using `/` key
- **Data Viewer**: View table data with pagination and row highlighting
- **Row-Level Navigation**: Navigate through data rows with cursor highlighting
- **Data Search**: Search within table data using `/` key
- **Row Detail Modal**: View individual rows in a 2-column format (Column | Value)
- **Cell Editing**: Edit individual cell values with live database updates
- **SQL Query Interface**: Execute custom SQL queries with parameter support
- **Responsive Design**: Adapts to terminal size and fits content to screen
- **Navigation**: Intuitive keyboard navigation throughout all modes

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
- `←/→` or `h/l`: Navigate between table list pages
- `/`: Start searching tables
- `Enter`: View selected table data
- `s`: Switch to SQL query mode
- `r`: Refresh table list
- `q` or `Ctrl+C`: Quit

### Search Mode (when searching tables)
- Type to search table names
- `Enter` or `Esc`: Finish search
- `Backspace`: Delete characters

### Table Data Mode
- `↑/↓` or `k/j`: Navigate between data rows (with highlighting)
- `←/→` or `h/l`: Navigate between data pages
- `/`: Start searching within table data
- `Enter`: View selected row in detail modal
- `Esc`: Return to table list
- `r`: Refresh current table data
- `q` or `Ctrl+C`: Quit

### Data Search Mode (when searching within table data)
- Type to search within all columns of the table
- `Enter` or `Esc`: Finish search
- `Backspace`: Delete characters

### Row Detail Modal
- `↑/↓` or `k/j`: Navigate between fields (Column | Value format)
- `Enter`: Edit selected field value
- `Esc`: Return to table data view
- `q` or `Ctrl+C`: Quit

### Cell Edit Mode
- **Readline-style Editing**: Full cursor control and advanced editing
- **Cursor Movement**: `←/→` arrows, `Ctrl+←/→` for word navigation
- **Line Navigation**: `Home`/`Ctrl+A` (start), `End`/`Ctrl+E` (end)
- **Deletion**: `Backspace`, `Delete`/`Ctrl+D`, `Ctrl+W` (word), `Ctrl+K` (to end), `Ctrl+U` (to start)
- **Text Wrapping**: Long values are automatically wrapped for better visibility
- `Enter`: Save changes to database
- `Esc`: Cancel editing and return to row detail

### SQL Query Mode
- **Advanced Text Editing**: Full readline-style editing controls
- **Dual Focus Mode**: Switch between query input and results with `Tab`
- **Query Input Focus**:
  - **Cursor Movement**: `←/→` arrows, `Ctrl+←/→` for word navigation  
  - **Line Navigation**: `Home`/`Ctrl+A` (start), `End`/`Ctrl+E` (end)
  - **Deletion**: `Backspace`, `Delete`/`Ctrl+D`, `Ctrl+W` (word), `Ctrl+K` (to end), `Ctrl+U` (to start)
  - `Enter`: Execute query
  - `Tab`: Switch focus to results (when available)
- **Results Focus**:
  - `↑/↓` or `k/j`: Navigate between result rows
  - `Enter`: View selected row in detail modal
  - `Tab`: Switch focus back to query input
- Type your SQL query (all keys work as input, no conflicts with navigation)
- `Esc`: Return to table list
- `q` or `Ctrl+C`: Quit

## Features Implemented

1. **Table Browsing**: Lists all tables in the database with pagination
2. **Table Search**: Filter tables by name using `/` to search
3. **Paginated Data View**: Shows table data with pagination (20 rows per page)
4. **Row Highlighting**: Cursor-based row selection with visual highlighting
5. **Data Search**: Search within table data across all columns
6. **Row Detail Modal**: 2-column view showing Column | Value for selected row
7. **Cell Editing**: Live editing of individual cell values with database updates
8. **Readline-style Editing**: Full cursor control with word navigation, line navigation, and advanced deletion
9. **Text Wrapping**: Long values are automatically wrapped in edit and detail views
10. **Primary Key Detection**: Uses primary keys for reliable row updates
11. **Screen-Aware Display**: Content automatically fits terminal size
12. **SQL Query Execution**: Execute custom SQL queries with advanced text editing and dual-focus mode
13. **Query Results Navigation**: Navigate and interact with query results just like table data
14. **Error Handling**: Displays database errors gracefully
15. **Responsive UI**: Clean, styled interface that adapts to terminal size
16. **Column Information**: Shows column names and handles NULL values
17. **Navigation**: Intuitive keyboard shortcuts for all operations
18. **Dynamic Column Width**: Columns adjust to terminal width

## Navigation Flow

```
Table List → Table Data → Row Detail → Cell Edit
     ↓           ↓           ↓           ↓
   Search     Data Search   Field Nav   Value Edit
     ↓           ↓           ↓           ↓
SQL Query    Row Select   Cell Select  Save/Cancel
```

## Sample Database

The included `sample.db` contains:
- `users` table with id, name, email, age columns
- `products` table with id, name, price, category columns

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver

## Database Updates

The application supports live editing of database records:
- Uses primary keys when available for reliable row identification
- Falls back to full-row matching when no primary key exists
- Updates are immediately reflected in the interface
- All changes are committed to the database in real-time