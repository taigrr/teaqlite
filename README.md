# TeaQLite

A colorful-but-minimal terminal user interface for browsing SQLite databases built with Bubble Tea.

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
- `Ctrl+C`: Quit

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
  - `Enter`: View selected row in detail modal (editable for simple queries)
  - `Tab`: Switch focus back to query input
- **Note**: Query results from simple single-table queries can be edited; complex queries (JOINs, etc.) are automatically detected and handled safely
- Type your SQL query (all keys work as input, no conflicts with navigation)
- `Esc`: Return to table list
- `q` or `Ctrl+C`: Quit
