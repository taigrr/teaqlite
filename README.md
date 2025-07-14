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
